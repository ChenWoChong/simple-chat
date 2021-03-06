package server

import (
	"github.com/ChenWoChong/simple-chat/db"
	"github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
)

var config *Config

type Config struct {
	ServerRpcOpt   ServerRpcOpt   `yaml:"server_rpc_opt"`
	ServerRabbitmq ServerRabbitmq `yaml:"server_rabbitmq"`
	MysqlOpt       db.MsOpt       `yaml:"mysql_opt"`
}

type ServerRpcOpt struct {
	ListenAddr   string `json:"listen_addr" yaml:"listen_addr" `
	IsTLS        bool   `json:"is_tls" yaml:"is_tls"`
	CertFilePath string `json:"cert_file_path" yaml:"cert_file_path"`
	KeyFilePath  string `json:"key_file_path" yaml:"key_file_path"`
}

type ServerRabbitmq struct {
	URL               string `yaml:"url"`
	Vhost             string `yaml:"vhost"`
	ExchangeName      string `yaml:"exchange_name"`
	ExchangeTopicName string `yaml:"exchange_topic_name"`
}

// LoadConfOrDie Loading config from local file
func LoadConfOrDie(confFile string) {

	config = &Config{}
	yamlFile, err := ioutil.ReadFile(confFile)
	if err != nil {
		glog.Fatalf("Open Configure file failed, %v ", err)
	}
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		glog.Fatalf("Unmarshal, %v ", err)
	}

}

//Get return config
func Get() *Config {
	return config
}
