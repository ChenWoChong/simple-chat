package db

import (
	"github.com/golang/glog"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	db        *gorm.DB
	pgIsDebug bool
	logTag    = `[db]`
)

//Init 初始化MS
func Init(cf MsOpt) {

	db = OpenMs(cf)

	AutoMigrate()

}

//Close close DB
func Close() {
	glog.Warningln("关闭本地数据库连接...")
}

//AutoMigrate migrate model
func AutoMigrate() {
	db.AutoMigrate(&BroadcastMessage{})
	db.AutoMigrate(&PrivateMessage{})
	db.AutoMigrate(&User{})
}

//RandomID get uuid
func RandomID() string {
	u := uuid.New()
	return u.String()
}

//GetDB return db point
func GetDB() *gorm.DB {

	if pgIsDebug {
		return db.Debug()
	}

	return db
}
