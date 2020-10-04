package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ChenWoChong/simple-chat/config"
	"github.com/ChenWoChong/simple-chat/message"
	rabbitMQ "github.com/ChenWoChong/simple-chat/pkg/rabbitmq"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"
)

const (
	logTag  = `[RPC_SERVER]`
	netWork = `tcp`
)

var (
	msgNumber int64 // TODO 为消息排序，临时使用number， 需考虑并发修改安全问题
)

//Server real grpc service server
type Server struct {
	ctx context.Context // 上下文

	opt       *config.ServerRpcOpt // grpc server 配置参数
	rpcServer *grpc.Server         // grpc server

	message.UnimplementedChatroomServer

	sync.Map   // 存储用户连接关系：map[senderID]Chatroom_ChatServer
	allUserMap *UserMap

	Entropy io.Reader

	mqOpt          *config.ServerRabbitmq
	serverFanoutMQ *rabbitMQ.RabbitMQ // rabbitMQ
	serverDirectMQ *rabbitMQ.RabbitMQ // rabbitMQ
}

//NewServer NewServer
func NewServer(ctx context.Context, opt *config.ServerRpcOpt, mqOpt *config.ServerRabbitmq) *Server {
	rpcServer := &Server{ctx: ctx, opt: opt, mqOpt: mqOpt, allUserMap: NewUserMap()}
	if err := rpcServer.init(); err != nil {
		glog.Fatal(logTag, err)
	}

	return rpcServer
}

func (s *Server) init() error {

	// 初始化rpcServer
	serverOptList := make([]grpc.ServerOption, 0)

	if s.opt.IsTLS {
		glog.Infoln(logTag, `创建tls-grpc服务端`)

		credentials, err := credentials.NewServerTLSFromFile(s.opt.CertFilePath, s.opt.KeyFilePath)
		if err != nil {
			glog.Errorln(logTag, `证书创建失败`, err.Error())
			return err
		}

		serverOptList = []grpc.ServerOption{grpc.Creds(credentials)}
	} else {
		glog.Infoln(logTag, `创建明文grpc服务端`)
	}
	s.rpcServer = grpc.NewServer(serverOptList...)

	// 初始化mq
	s.serverFanoutMQ = InitMq(s.mqOpt.URL, s.mqOpt.ExchangeName, rabbitMQ.Fanout)
	s.serverDirectMQ = InitMq(s.mqOpt.URL, s.mqOpt.ExchangeDirectName, rabbitMQ.Direct)

	// setup entropy
	seed := time.Now().UnixNano()
	source := rand.NewSource(seed)
	s.Entropy = &syncReader{reader: rand.New(source)}

	return nil
}

//Serve run grpc service
func (s *Server) Serve() {

	glog.Infoln(logTag, `绑定端口信息`, s.opt.ListenAddr)

	listener, err := net.Listen(netWork, s.opt.ListenAddr)
	if err != nil {
		glog.Errorln(logTag, `绑定ip失败`, err.Error())
		return
	}

	// 注册可调用函数
	message.RegisterChatroomServer(s.rpcServer, s)

	if err := s.rpcServer.Serve(listener); err != nil {
		glog.Errorln(logTag, `grpc服务启动失败`, err.Error())
	}
}

//Run start grpc service
func (s *Server) Run() {
	go s.Serve()
}

//Stop grpc service
func (s *Server) Stop() {
	glog.Infoln(logTag, `grpc 服务退出`)
	s.rpcServer.GracefulStop()
}

/**************************************************************************** Function ****************************************************************************/

func (s *Server) genID() (string, error) {
	currentTime := time.Now()
	id, err := ulid.New(ulid.Timestamp(currentTime), s.Entropy)
	if err != nil {
		glog.Errorln("ERROR|SendMessage|ulid.New", err)
		return "", err
	}
	return id.String(), nil

}

func (s *Server) register(msg *message.Message, srv message.Chatroom_ChatServer) {

	var userInfo *UserInfo
	var ok bool

	// 判断用户是否已经注册
	userInfo, ok = s.allUserMap.GetUserInfo(msg.Sender)
	if !ok {
		userInfo = &UserInfo{
			Cancel:   make(chan bool),
			UserID:   uuid.New().String(),
			UserName: msg.Sender,
			IsOnline: true,
		}
		s.allUserMap.AddUser(userInfo)
	}

	glog.V(3).Infoln(logTag, `register: `, msg.Sender)

	// 存储连接
	s.Map.Store(msg.Sender, srv)

	// 启用消息处理机制
	s.runSubscriber(userInfo.Cancel, msg, srv)
}

func (s *Server) unRegister(userName string) {

	glog.V(3).Infoln(logTag, `unRegister: `, userName)

	// 从连接池中删除
	s.Map.Delete(userName)

	// 用户设置为离线, 并关闭mq连接
	s.allUserMap.SetUserState(userName, false)
}

func (s *Server) getChatServer(userName string) (srv message.Chatroom_ChatServer, ok bool) {
	if srv, ok := s.Map.Load(userName); ok {
		return srv.(message.Chatroom_ChatServer), ok
	} else {
		return nil, false
	}
}

func (s *Server) single(userName string, mes *message.Message) {
	srv, ok := s.getChatServer(userName)
	if !ok {
		// TODO 对方暂时不在线
		return
	}

	if err := srv.Send(mes); err != nil {
		glog.Errorln(logTag, `Err Single TO user: `, userName, err)
	}
}

func (s *Server) broadcast(msg *message.Message) {

	glog.Infoln(logTag, fmt.Sprintf(`Broadcast msg from: %s, msg:%s`, msg.Sender, msg.Content))

	s.Map.Range(
		func(userNameI, srvI interface{}) bool {
			userName := userNameI.(string)
			srv := srvI.(message.Chatroom_ChatServer)

			if err := srv.Send(msg); err != nil {
				glog.Errorln(logTag, `Err send msg err to `, userName, err)
			}

			return true
		},
	)
}

func (s *Server) handle(msg *message.Message, chatServer message.Chatroom_ChatServer) {

	// 判断是否已经创建连接
	if _, ok := s.getChatServer(msg.Sender); !ok {

		// 注册连接 // TODO 关闭连接
		s.register(msg, chatServer)
	}

	// 消息分发
	uID, err := s.genID()
	if err != nil {
		return
	}
	msg.Id = uID

	if msg.SendTo != "" {
		// 发给消息队列
		s.serverDirectMQ.Publish(s.mqOpt.ExchangeDirectName, msg, msg.SendTo)
		// 发给自己
		s.single(msg.Sender, msg)
	} else {
		s.serverFanoutMQ.Publish(s.mqOpt.ExchangeName, msg, "")
		//s.broadcast(msg)
	}
}

func (s *Server) runSubscriber(cancel chan bool, msg *message.Message, chatServer message.Chatroom_ChatServer) {

	glog.Infoln(logTag, `runSubscriber: `, msg.Sender, s.mqOpt.URL, s.mqOpt.ExchangeName, msg.Sender)

	//订阅 广播消息
	fanoutReceiver := NewSubscriber(s.mqOpt.URL, s.mqOpt.ExchangeName, msg.Sender, "")
	broadcastMsgs := fanoutReceiver.Consume()
	go func(cancel chan bool) {
		for {

			select {
			case <-cancel:

				glog.V(3).Infoln(logTag, fmt.Sprintf("[%s] fanoutReceiver Close", msg.Sender))

				fanoutReceiver.Close()
				return

			case d, ok := <-broadcastMsgs:
				if !ok {
					return
				}

				var msg message.Message
				if err := json.Unmarshal(d.Body, &msg); err != nil {
					glog.Errorln(logTag, err)
					return
				}

				if err := chatServer.Send(&msg); err != nil {
					glog.Errorln(logTag, `Err send msg err to `, msg.Sender, err)
					return
				}
			}
		}
	}(cancel)

	// 订阅 私人消息
	directReceiver := NewSubscriber(s.mqOpt.URL, s.mqOpt.ExchangeDirectName, msg.Sender+"_direct", msg.Sender)
	personalMsgs := directReceiver.Consume()
	go func(cancel chan bool) {
		for {

			select {
			case <-cancel:

				glog.V(3).Infoln(logTag, fmt.Sprintf("[%s] directReceiver Close", msg.Sender))

				directReceiver.Close()
				return

			case d, ok := <-personalMsgs:
				if !ok {
					return
				}

				var msg message.Message
				if err := json.Unmarshal(d.Body, &msg); err != nil {
					glog.Errorln(logTag, err)
					return
				}

				if err := chatServer.Send(&msg); err != nil {
					glog.Errorln(logTag, `Err send msg err to `, msg.Sender, err)
					return
				}

			}

		}
	}(cancel)
}

func (s *Server) Login(ctx context.Context, loginReq *message.LoginReq) (*message.LoginRes, error) {

	userName := loginReq.UserName

	// 用户是否已存在
	_, exist := s.allUserMap.GetUserInfo(userName)
	if exist {

		// 用户是否已经登录
		_, ok := s.getChatServer(userName)
		if ok {
			failMes := &message.LoginRes{
				UserName: userName,
				State:    false,
				Info:     `User logged in`,
			}
			return failMes, nil
		}

	} else { // 不存在, 则存储到用户表
		userInfo := UserInfo{
			Cancel:   make(chan bool),
			UserID:   uuid.New().String(),
			UserName: userName,
			IsOnline: true,
		}
		s.allUserMap.AddUser(&userInfo)
	}

	// 设置user为在线
	s.allUserMap.SetUserState(userName, true)

	successMes := &message.LoginRes{
		UserName: userName,
		State:    true,
		Info:     `Login Success`,
	}

	return successMes, nil
}

func (s *Server) GetUserList(ctx context.Context, msg *message.BaseReq) (*message.UserList, error) {

	users := make([]*message.UserInfo, 0)

	for _, userInfo := range s.allUserMap.GetAllUserList() {
		users = append(users, &message.UserInfo{UserName: userInfo.UserName, State: userInfo.IsOnline})
	}

	return &message.UserList{Users: users}, nil
}

func (s *Server) Chat(chatServer message.Chatroom_ChatServer) (err error) {

	var sender string

	for {
		select {
		case <-s.ctx.Done():
			glog.Info(logTag, `cancel server....`)
			return

		default:

			// 消息接收
			msg, err := chatServer.Recv()
			if err == io.EOF {
				if sender != "" {
					s.unRegister(sender)
				}
				return nil
			}

			if err != nil {
				if grpcErr, ok := status.FromError(err); ok {
					if grpcErr.Code() == codes.Canceled {
						if sender != "" {
							s.unRegister(sender)
						}
					}
				}
				glog.Errorf("%s chat error: %+v", logTag, err)

				return err
			}

			sender = msg.Sender

			// 处理传递
			s.handle(msg, chatServer)
		}
	}
}
