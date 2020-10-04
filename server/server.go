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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"
	"io"
	"net"
	"strconv"
	"sync"
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

	mqOpt    *config.ServerRabbitmq
	serverMQ *rabbitMQ.RabbitMQ // rabbitMQ
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
	s.serverMQ = InitFanout(s.mqOpt.URL, s.mqOpt.ExchangeName)

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

func (s *Server) register(userName string, srv message.Chatroom_ChatServer) {
	s.Map.Store(userName, srv)
}

func (s *Server) unRegister(userName string) {
	s.Map.Delete(userName)
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
		// 判断用户是否已经注册
		if _, ok := s.allUserMap.GetUserInfo(msg.Sender); !ok {
			userInfo := UserInfo{
				UserID:   uuid.New().String(),
				UserName: msg.Sender,
				IsOnline: true,
			}
			s.allUserMap.AddUser(userInfo)
		}

		// 注册连接
		s.register(msg.Sender, chatServer)

		// 启用消息处理机制
		s.runSubscriber(msg.Sender, chatServer)
	}

	// 消息分发
	msgNumber++
	msg.Id = strconv.FormatInt(msgNumber, 10)

	s.serverMQ.Publish(s.mqOpt.ExchangeName, msg, msg.SendTo)
	//if msg.SendTo != "" {
	//	s.single(msg.SendTo, msg)
	//} else {
	//	s.broadcast(msg)
	//}
}

func (s *Server) runSubscriber(name string, chatServer message.Chatroom_ChatServer) {

	glog.Infoln(logTag, `runSubscriber: `, name, s.mqOpt.URL, s.mqOpt.ExchangeName, name)

	receiver := NewSubscriber(s.mqOpt.URL, s.mqOpt.ExchangeName, name)

	// TODO 错误的处理， 消息传递方式
	glog.Infoln(logTag, `NewSubscriber: `, name)

	//for {
	//接收消息时，指定
	msgs := receiver.Consume()
	go func() {
		for d := range msgs {

			var msg message.Message
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				glog.Errorln(logTag, err)
				return
			}

			if err := chatServer.Send(&msg); err != nil {
				glog.Errorln(logTag, `Err send msg err to `, name, err)
				return
			}
		}
	}()

	glog.Infoln(logTag, `Consume func: `, name)

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
			UserID:   uuid.New().String(),
			UserName: userName,
			IsOnline: true,
		}
		s.allUserMap.AddUser(userInfo)
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
					s.allUserMap.SetUserState(sender, false)
				}
				return nil
			}

			if err != nil {
				if grpcErr, ok := status.FromError(err); ok {
					if grpcErr.Code() == codes.Canceled {
						if sender != "" {
							s.unRegister(sender)
							s.allUserMap.SetUserState(sender, false)
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
