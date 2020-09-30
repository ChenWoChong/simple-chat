package server

import (
	"context"
	"fmt"
	"github.com/ChenWoChong/simple-chat/config"
	"github.com/ChenWoChong/simple-chat/message"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"
	"io"
	"net"
	"sync"
)

const (
	logTag  = `[RPC_SERVER]`
	netWork = `tcp`
)

//Server real grpc service server
type Server struct {
	ctx context.Context // 上下文

	opt       *config.ServerRpcOpt // grpc server 配置参数
	rpcServer *grpc.Server         // grpc server

	message.UnimplementedChatroomServer

	sync.Map // 存储用户连接关系：map[senderID]Chatroom_ChatServer
}

//NewServer NewServer
func NewServer(ctx context.Context, opt *config.ServerRpcOpt) *Server {
	rpcServer := &Server{ctx: ctx, opt: opt}
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

/************************************** Function **************************************/

func (s *Server) register(userID int64, srv message.Chatroom_ChatServer) {
	s.Map.Store(userID, srv)
}

func (s *Server) unRegister(userID int64) {
	s.Map.Delete(userID)
}

func (s *Server) getChatServer(userID int64) (srv message.Chatroom_ChatServer, ok bool) {
	if srv, ok := s.Map.Load(userID); ok {
		return srv.(message.Chatroom_ChatServer), ok
	} else {
		return nil, false
	}
}

func (s *Server) single(userID int64, mes *message.Message) {
	srv, ok := s.getChatServer(userID)
	if !ok {
		// TODO 对方暂时不在线
		return
	}

	if err := srv.Send(mes); err != nil {
		glog.Errorln(logTag, `Err Single TO user: `, userID, err)
	}
}

func (s *Server) broadcast(msg *message.Message) {

	glog.Infoln(logTag, fmt.Sprintf(`Broadcast msg from: %d, msg:%s`, msg.Sender, msg.Content))

	s.Map.Range(
		func(userIDI, srvI interface{}) bool {
			userID := userIDI.(int64)
			srv := srvI.(message.Chatroom_ChatServer)

			if err := srv.Send(msg); err != nil {
				glog.Errorln(logTag, `Err send msg err to `, userID, err)
			}

			return true
		},
	)

}

func (s *Server) deliver(msg *message.Message) {
	if msg.SendTo != 0 {
		s.single(msg.SendTo, msg)
	} else {
		s.broadcast(msg)
	}
}

func (s *Server) Chat(chatServer message.Chatroom_ChatServer) (err error) {

	var senderID int64

	for {
		select {
		case <-s.ctx.Done():
			glog.Info(logTag, `cancel server....`)
			return

		default:
			msg, err := chatServer.Recv()
			if err == io.EOF {
				if senderID != 0 {
					s.unRegister(senderID)
				}
				return nil
			}

			if err != nil {
				if grpcErr, ok := status.FromError(err); ok {
					if grpcErr.Code() == codes.Canceled {
						if senderID != 0 {
							s.unRegister(senderID)
						}
					}
				}
				glog.Errorf("%s chat error: %+v", logTag, err)

				return err
			}

			// 判断是否需要注册
			senderID = msg.Sender

			if _, ok := s.getChatServer(msg.Sender); !ok {
				s.register(msg.Sender, chatServer)
			}

			s.deliver(msg)
		}
	}
}
