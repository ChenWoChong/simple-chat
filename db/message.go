package db

import (
	"errors"
	"github.com/ChenWoChong/simple-chat/message"
)

type BroadcastMessage struct {
	ID       string       `json:"id" gorm:"primaryKey"`
	Sender   string       `json:"sender"`
	SendTo   string       `json:"send_to"`
	Content  string       `json:"content"`
	Type     message.Type `json:"type"`
	SendTime int64        `json:"send_time"`
}

//Get Get
func (m *BroadcastMessage) Get() error {
	return db.Model(m).Where(m).First(m).Error
}

//Create use Create
func (m *BroadcastMessage) Create() error {
	return db.Create(m).Error
}

func (m *BroadcastMessage) Delete() error {
	return db.Delete(m).Error
}

func (m *BroadcastMessage) Update() error {
	return db.Updates(m).Error
}

func GetBroadcastMsgList(startTime, endTime int64) (msgs []BroadcastMessage, err error) {

	msgs = make([]BroadcastMessage, 0)

	dbObj := db.Model(&BroadcastMessage{})
	if endTime != 0 {
		dbObj = dbObj.Where("send_time BETWEEN ? AND ?", startTime, endTime)
	}

	err = dbObj.Find(&msgs).Error

	return
}

type PrivateMessage struct {
	ID       string       `json:"id" gorm:"primaryKey"`
	Sender   string       `json:"sender"`
	SendTo   string       `json:"send_to" gorm:"index"`
	Content  string       `json:"content"`
	Type     message.Type `json:"type"`
	SendTime int64        `json:"send_time"`
}

//Get Get
func (m *PrivateMessage) Get() error {
	return db.Model(m).Where(m).First(m).Error
}

//Create use Create
func (m *PrivateMessage) Create() error {
	return db.Create(m).Error
}

func (m *PrivateMessage) Delete() error {
	return db.Delete(m).Error
}

func (m *PrivateMessage) Update() error {
	return db.Updates(m).Error
}

func GetUserPrivateMsgList(userName string) (msgs []PrivateMessage, err error) {

	msgs = make([]PrivateMessage, 0)

	if userName == "" {
		return nil, errors.New("userName is nil")
	}

	err = db.Model(&PrivateMessage{}).
		Where("send_to=?", userName).
		Find(&msgs).Error

	return
}
