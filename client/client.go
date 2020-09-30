package client

import (
	"context"
	"crypto/x509"
	"github.com/ChenWoChong/simple-chat/config"
	"github.com/ChenWoChong/simple-chat/message"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
	"time"
)

const (
	logTag  string        = `[RPC_CLIENT]`
	timeOut time.Duration = time.Second * 30
)

type Client struct {
	ctx context.Context

	opt            *config.ClientRpcOpt
	rpcEmailClient message.MessageClient
}

func NewClient(ctx context.Context, opt *config.ClientRpcOpt) *Client {
	client := &Client{
		opt: opt,
		ctx: ctx,
	}
	if err := client.init(); err != nil {
		glog.Errorln(logTag, err)
		return nil
	}

	return client
}

func (c *Client) init() error {
	glog.Infoln(logTag, `创建grpc链接`)

	dialOptList := make([]grpc.DialOption, 0)

	if c.opt.IsTLS {
		glog.Infoln(logTag, `创建tls-grpc客户端`)

		var (
			cred credentials.TransportCredentials
			err  error
		)

		if c.opt.CaFilePath == "" {
			cred = credentials.NewClientTLSFromCert(x509.NewCertPool(), c.opt.ServerHostOverride)
		} else {
			cred, err = credentials.NewClientTLSFromFile(c.opt.CaFilePath, c.opt.ServerAddr)
			if err != nil {
				glog.Errorln(logTag, `创建tls验证信息失败`, err.Error())
				return err
			}
		}

		dialOptList = append(dialOptList, grpc.WithTransportCredentials(cred))

	} else {
		glog.Infoln(logTag, `创建明文grpc客户端`)

		dialOptList = append(dialOptList, grpc.WithInsecure())
	}

	dialOptList = append(dialOptList, grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
	dialOptList = append(dialOptList, grpc.WithTimeout(7200*time.Second))

	conn, err := grpc.DialContext(c.ctx, c.opt.ServerAddr, dialOptList...)
	if err != nil {
		glog.Error(logTag, `创建grpc链接失败`, err.Error())
		return err
	}

	glog.Infoln(logTag, conn.GetState())

	c.rpcEmailClient = message.NewMessageClient(conn)

	return nil
}

/*************************************** call Server ***************************************/

func (c *Client) SendMessage(mes *message.ResMes) (err error) {
	ctx, cancel := context.WithTimeout(c.ctx, timeOut*30)
	defer cancel()

	stream, err := c.rpcEmailClient.SendMessage(ctx)
	if err != nil {
		glog.Errorf(logTag, "failed to call: %v", err)
		return
	}

	for {

		time.Sleep(2 * time.Second)

		stream.Send(&message.ReqMes{Content: "haha"})
		if err != nil {
			glog.Errorf(logTag, "failed to send: %v", err)
			break
		}
		reply, err := stream.Recv()
		if err != nil {
			glog.Errorf(logTag, "failed to recv: %v", err)
			break
		}
		glog.Info(logTag, "Greeting: %s", reply.Content)
	}
	return
}