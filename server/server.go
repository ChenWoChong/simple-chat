package server

import (
	"context"
	"github.com/ChenWoChong/simple-chat/config"
	"github.com/ChenWoChong/simple-chat/message"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
	"io"
	"net"
)

const (
	logTag  = `[RPC_SERVER]`
	netWork = `tcp`
)

//Server real grpc service server
type Server struct {
	ctx context.Context

	opt       *config.ServerRpcOpt
	rpcServer *grpc.Server
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
	message.RegisterMessageServer(s.rpcServer, s)

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

func (s *Server) SendMessage(grpcMes message.Message_SendMessageServer) (err error) {
	for {
		select {
		case <-s.ctx.Done():
			glog.Info(logTag, `cancel server....`)
			return

		default:
			mes, err := grpcMes.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				glog.Errorf("failed to recv: %v", err)
				return err
			}

			glog.Info(logTag, mes.Content)
			grpcMes.Send(&message.ResMes{Content: "Hello Client"})
		}
	}

	return nil
}
