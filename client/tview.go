package client

import (
	"context"
	"fmt"
	"github.com/ChenWoChong/simple-chat/message"
	"github.com/ChenWoChong/simple-chat/pkg/utils"
	"github.com/gdamore/tcell"
	"github.com/golang/glog"
	"github.com/rivo/tview"
	"time"
)

var (
	userName string

	messageMap *MessageMap

	terminal *tview.Application

	history   *tview.List
	allUser   *tview.List
	input     *tview.InputField
	termFlex  *tview.Flex
	loginForm *tview.Form

	rpcClient  *Client
	chatClient message.Chatroom_ChatClient
)

func SetupLogin(ctx context.Context, client *Client) *tview.Application {

	rpcClient = client

	messageMap = NewMessageMap()

	loginForm = tview.NewForm().
		AddInputField("UserName", "", 20, nil, nil).SetLabelColor(tcell.ColorDarkBlue).
		AddButton("Login", openChatroom).
		AddButton("Quit", func() {
			terminal.Stop()
		})
	loginForm.SetBorder(true).SetTitle("Please inter your userName to login").SetTitleAlign(tview.AlignCenter)

	terminal = tview.NewApplication().SetRoot(loginForm, true).SetFocus(loginForm)

	return terminal
}

func setupChatroom() {

	history = tview.NewList()
	history.SetSelectedFocusOnly(true).
		ShowSecondaryText(false).
		SetBorder(true)

	allUser = tview.NewList().SetSelectedFocusOnly(true)
	allUser.ShowSecondaryText(false).
		SetBorder(true).
		SetTitle("ALL-USERS")

	input = tview.NewInputField().
		SetLabel(">> ").
		SetLabelColor(tcell.ColorPurple)

	input.SetDoneFunc(inputHandle)

	termFlex = tview.NewFlex().
		AddItem(allUser, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(history, 0, 3, false).
			AddItem(input, 0, 1, true),
			0, 5, false)

	//glog.Infoln(logTag, `setupChatroom Success`)
}

func inputHandle(key tcell.Key) {
	if input.GetText() == "" {
		return
	}

	text := input.GetText()

	if text == "exit" || text == "EXIT" {
		terminal.Stop()
	}

	sendTo, content, err := utils.ParseContent(text)
	if err != nil {
		glog.Errorln(logTag, err)
		return
	}

	var msgType message.Type
	if sendTo == "" {
		msgType = message.Type_ALL
	} else {
		msgType = message.Type_SIMPLE
	}

	msg := &message.Message{
		Sender:   userName,
		Content:  content,
		SendTo:   sendTo,
		SendTime: time.Now().Unix(),
		Type:     msgType,
	}

	// 发送到服务器
	go func() {
		err := chatClient.Send(msg)
		if err != nil {
			glog.Fatal(logTag, "failed to call message.Send:", err)
		}
	}()
	input.SetText("")
}

func queryLatestMessages() {

	keys := messageMap.sortedMessagesKeys()
	startID := "0"
	if len(keys) != 0 {
		startID = keys[len(keys)-1]
	}

	msgs, err := rpcClient.GetLatestHistoryMsgs(context.Background(), &message.HistoryMsgReq{
		UserName: userName,
		StartID:  startID,
		EndID:    "a",
	})
	if err != nil {
		glog.Fatal(logTag, "failed to call message.QueryMessagesInRange:", err)
	}
	for _, m := range msgs {
		messageMap.Add(&message.Message{
			Id:       m.Id,
			Sender:   m.Sender,
			SendTo:   m.SendTo,
			Content:  m.Content,
			SendTime: m.SendTime,
			Type:     m.Type,
		})
	}
	terminal.QueueUpdateDraw(func() {
		updateHistory()
	})
}

func loopForMessages(ctx context.Context, rpcClient *Client) {
	//chatClient, err := .Messages(context.Background(), &push.MessagesRequest{
	//	ChatroomID: chatroomID,
	//})

	// query for potential missing messages
	queryLatestMessages()

	var err error
	chatClient, err = rpcClient.Chat(ctx)
	if err != nil {
		glog.Fatal(logTag, "failed to call push.Messages:", err)
	}

	// loop recving latest messages
	for {
		reply, err := chatClient.Recv()
		if err != nil {
			glog.Errorln(logTag, err)
			break
		}
		messageMap.Add(reply)
		terminal.QueueUpdateDraw(func() {
			updateHistory()
			updateUserList()
		})
	}
}

func openChatroom() {

	// 用户登录
	userName = loginForm.GetFormItem(0).(*tview.InputField).GetText()

	loginRes, err := rpcClient.Login(context.Background(), userName)
	if err != nil {
		terminal.Stop()
	} else {
		if loginRes.State == false {
			loginForm.SetTitle(fmt.Sprintf(`该用户已经登录! [重新输入：UserName]`))
			terminal.SetRoot(loginForm, true).SetFocus(loginForm.GetFormItemByLabel("UserName"))
			return
		}
	}

	// 打开 chatroom
	setupChatroom()

	history.SetTitle(fmt.Sprintf("Chatroom<%s>", userName))

	// open chatroom
	terminal.SetRoot(termFlex, true).SetFocus(input)

	historyFocus := false
	terminal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTAB {
			historyFocus = !historyFocus
			if historyFocus {
				terminal.SetFocus(history)
			} else {
				terminal.SetFocus(input)
			}
			return nil
		}
		return event
	})

	go loopForMessages(context.Background(), rpcClient)

	//updateHistory()
	updateUserList()
}

func updateHistory() {
	messageMap.Lock()
	defer messageMap.Unlock()

	keys := messageMap.sortedMessagesKeys()

	history.Clear()
	for _, k := range keys {
		msg := messageMap.messageM[k]

		var text string
		if msg.SendTo != "" {
			text = fmt.Sprintf(
				"%s <%s> SendTo <%s>: %s",
				time.Unix(0, msg.SendTime).Format("2006-01-02 15:04:05 MST"),
				msg.Sender,
				msg.SendTo,
				msg.Content,
			)
		} else {
			text = fmt.Sprintf(
				"%s <%s>: %s",
				time.Unix(0, msg.SendTime).Format("2006-01-02 15:04:05 MST"),
				msg.Sender,
				msg.Content,
			)
		}
		history.AddItem(text, "", 0, nil)
	}
	history.SetCurrentItem(-1)
}

func updateUserList() {

	userList, err := rpcClient.GetUserList(context.Background())
	if err != nil || userList == nil {
		glog.Errorln(logTag, `updateUserList Err`)
		return
	}

	allUser.Clear()
	for _, userInfo := range userList.Users {
		var stateString string
		if userInfo.State {
			stateString = `Online`
		} else {
			stateString = ` *Offline* `
		}
		allUser.AddItem(
			fmt.Sprintf("%s\t<%s>", userInfo.UserName, stateString),
			"", 0, nil)
	}

	allUser.SetCurrentItem(-1)
}
