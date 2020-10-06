package server

import (
	"github.com/ChenWoChong/simple-chat/db"
	"github.com/golang/glog"
	"sort"
	"sync"
)

type UserMap struct {
	AllUserMap map[string]*UserInfo
	sync.Mutex
}

type UserInfo struct {
	Cancel chan bool

	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	IsOnline bool   `json:"is_online"`
}

func NewUserMap() *UserMap {
	return &UserMap{
		AllUserMap: make(map[string]*UserInfo),
	}
}

func (m *UserMap) LoadForDB() error {

	users, err := db.GetUserList()
	if err != nil {
		glog.Error(logTag, err)
		return err
	}

	for _, user := range users {
		m.AllUserMap[user.UserName] = &UserInfo{
			Cancel:   make(chan bool),
			UserID:   user.UserID,
			UserName: user.UserName,
			IsOnline: false,
		}
	}

	return nil
}

func (m *UserMap) GetUserInfo(userName string) (*UserInfo, bool) {
	m.Lock()
	defer m.Unlock()

	userInfo, ok := m.AllUserMap[userName]
	if !ok {
		return nil, false
	}

	return userInfo, true
}

func (m *UserMap) AddUser(user *UserInfo) {
	m.Lock()
	defer m.Unlock()

	userDB := db.User{
		UserName: user.UserName,
		UserID:   user.UserID,
		IsOnline: user.IsOnline,
	}
	if err := userDB.Create(); err != nil {
		glog.Errorln(logTag, err)
		return
	}

	m.AllUserMap[user.UserName] = user
}

func (m *UserMap) Delete(userName string) {
	m.Lock()
	defer m.Unlock()

	delete(m.AllUserMap, userName)

	userDB, err := db.GetUserByName(userName)
	if err != nil {
		glog.Errorln(logTag, err)
		return
	}

	err = userDB.Delete()
	if err != nil {
		glog.Errorln(logTag, err)
		return
	}
}

func (m *UserMap) SetUserState(userName string, isOnline bool) {
	user, ok := m.GetUserInfo(userName)
	if !ok {
		return
	}
	if isOnline {
		user.Cancel = make(chan bool)
	} else {
		close(user.Cancel)
	}

	user.IsOnline = isOnline

	//
	//userDB, err := db.GetUserByName(userName)
	//if err != nil {
	//	glog.Errorln(logTag, err)
	//	return
	//}
	//err = userDB.UpdateState(isOnline)
	//if err != nil {
	//	glog.Errorln(logTag, err)
	//	return
	//}
}

func (m *UserMap) GetAllUserList() []UserInfo {
	m.Lock()
	defer m.Unlock()

	// sort
	sortKeys := make([]string, 0, len(m.AllUserMap))
	for k := range m.AllUserMap {
		sortKeys = append(sortKeys, k)
	}
	sort.Strings(sortKeys)

	userList := make([]UserInfo, 0, len(m.AllUserMap))
	for _, userName := range sortKeys {
		userList = append(userList, *m.AllUserMap[userName])
	}

	return userList
}
