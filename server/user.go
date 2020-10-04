package server

import (
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

	m.AllUserMap[user.UserName] = user
}

func (m *UserMap) Delete(userName string) {
	m.Lock()
	defer m.Unlock()

	delete(m.AllUserMap, userName)
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
}

func (m *UserMap) GetAllUserList() []UserInfo {
	m.Lock()
	defer m.Unlock()

	userList := make([]UserInfo, 0, len(m.AllUserMap))
	for _, userInfo := range m.AllUserMap {
		userList = append(userList, *userInfo)
	}

	return userList
}
