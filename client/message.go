package client

import (
	"github.com/ChenWoChong/simple-chat/message"
	"sort"
	"sync"
)

type MessageMap struct {
	messageM map[string]*message.Message
	sync.Mutex
}

func NewMessageMap() *MessageMap {
	return &MessageMap{
		messageM: make(map[string]*message.Message),
	}
}

func (m *MessageMap) Get(ID string) *message.Message {
	return m.messageM[ID]
}

func (m *MessageMap) Add(msg *message.Message) {
	m.Lock()
	defer m.Unlock()

	m.messageM[msg.Id] = msg
}

func (m *MessageMap) sortedMessagesKeys() (keys []string) {
	for k := range m.messageM {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return
}
