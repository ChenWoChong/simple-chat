package config

import (
	"github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
)

var config *Config

type Config struct {
	ServerRpcOpt ServerRpcOpt `yaml:"server_rpc_opt"`
	ClientRpcOpt ClientRpcOpt `yaml:"client_rpc_opt"`
}

type ServerRpcOpt struct {
	ListenAddr   string `json:"listen_addr" yaml:"listen_addr" `
	IsTLS        bool   `json:"is_tls" yaml:"is_tls"`
	CertFilePath string `json:"cert_file_path" yaml:"cert_file_path"`
	KeyFilePath  string `json:"key_file_path" yaml:"key_file_path"`
}

type ClientRpcOpt struct {
	ServerAddr         string `yaml:"server_addr"`
	CaFilePath         string `yaml:"ca_file_path"`
	ServerHostOverride string `yaml:"server_host_override"`
	IsTLS              bool   `yaml:"is_tls"`
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
