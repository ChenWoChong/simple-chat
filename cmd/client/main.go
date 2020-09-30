package main

import (
	"context"
	"flag"
	"github.com/ChenWoChong/simple-chat/client"
	"github.com/ChenWoChong/simple-chat/config"
	"github.com/golang/glog"
	"os"
	"os/signal"
	"syscall"
)

const logTag string = `[main] `

var (
	confFile    = flag.String("conf", "conf.yml", "The configure file")
	showVersion = flag.Bool("version", false, "show build version.")
	//pprof       = flag.String("pprof", "", "[localhost:6060]start debug page.")
	buildstamp = "UNKOWN"
	githash    = "UNKOWN"
	version    = "UNKOWN"
)

func main() {

	flag.Parse()
	defer glog.Flush()

	if *showVersion {
		println(`Delivery version :`, version)
		println(`Git Commit Hash :`, githash)
		println(`UTC Build Time :`, buildstamp)
		os.Exit(0)
	}

	{
		glog.Infoln("当前Alarm版本: ", version)
		glog.Infoln(`Git Commit Hash :`, githash)
		glog.Infoln(`UTC Build Time :`, buildstamp)
	}

	// init
	config.LoadConfOrDie(*confFile)

	ctx, cancel := context.WithCancel(context.Background())
	rpcClient := client.NewClient(ctx, &config.Get().ClientRpcOpt)

	// run
	glog.Infoln(logTag, `Client start...`)

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	go rpcClient.SendMessage("test")

	<-ch
	glog.Infoln(logTag, "收到 ctrl + c ...")
	cancel()

	glog.Infoln(logTag, `Server shutdown...`)
}
