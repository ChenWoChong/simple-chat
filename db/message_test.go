package db

import (
	"testing"
)

func TestGetBroadcastMsgListByID(t *testing.T) {
	Init(MsOpt{
		Host:   "127.0.0.1",
		Port:   3306,
		User:   "test",
		DBName: "test",
		Passwd: "123456",
	})

	datas, err := GetBroadcastMsgListByID("01EKY7Z0TCT3NZFKDKSV125KC3", "01EKYHW9GFB8ZJ6WJX4KKTVXRX")
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(len(datas),datas)
}
