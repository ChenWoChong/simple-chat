package db

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

//MsOpt  Connect options
type MsOpt struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	User   string `yaml:"user"`
	DBName string `yaml:"dbname"`
	Passwd string `yaml:"passwd"`
}

//ConnectURL return conect url addr
//TODO enable sslmode=disable
func getConnectURLMs(opt MsOpt) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8",
		opt.User,
		opt.Passwd,
		opt.Host,
		opt.Port,
		opt.DBName,
	)
}

//OpenMs connect to db database or exit
func OpenMs(opt MsOpt) *gorm.DB {
	client, err := gorm.Open(mysql.Open(getConnectURLMs(opt)), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return client
	// db.LogMode(true)
}
