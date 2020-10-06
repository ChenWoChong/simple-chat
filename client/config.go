package client

import (
	"github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
)

var config *Config

type Config struct {
	ClientRpcOpt ClientRpcOpt `yaml:"client_rpc_opt"`
	//MysqlOpt       db.MsOpt       `yaml:"mysql_opt"`
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
