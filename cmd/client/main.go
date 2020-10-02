package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ChenWoChong/simple-chat/client"
	"github.com/ChenWoChong/simple-chat/config"
	"github.com/ChenWoChong/simple-chat/message"
	"github.com/gdamore/tcell"
	"github.com/golang/glog"
	"github.com/rivo/tview"
	"log"
	"os"
	"sort"
	"sync"
	"time"
)

const logTag string = `[main] `

var (
	confFile    = flag.String("conf", "conf.yml", "The configure file")
	showVersion = flag.Bool("version", false, "show build version.")
	//pprof       = flag.String("pprof", "", "[localhost:6060]start debug page.")
	buildstamp = "UNKOWN"
	githash    = "UNKOWN"
	version    = "UNKOWN"

	userName string

	messageMap *MessageMap

	terminal  *tview.Application
	history   *tview.List
	allUser   *tview.List
	input     *tview.InputField
	termFlex  *tview.Flex
	loginForm *tview.Form

	rpcClient  *client.Client
	chatClient message.Chatroom_ChatClient
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

func main() {

	flag.Parse()
	defer glog.Flush()

	if *showVersion {
		println(`Delivery version :`, version)
		println(`Git Commit Hash :`, githash)
		println(`UTC Build Time :`, buildstamp)
		os.Exit(0)
	}

	{
		glog.Infoln("当前Alarm版本: ", version)
		glog.Infoln(`Git Commit Hash :`, githash)
		glog.Infoln(`UTC Build Time :`, buildstamp)
	}

	// init
	config.LoadConfOrDie(*confFile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rpcClient = client.NewClient(ctx, &config.Get().ClientRpcOpt)

	// run
	glog.Infoln(logTag, `Client start...`)

	setupLogin()
	setupTerminal()

	go loopForMessages(ctx, rpcClient)

	if err := terminal.Run(); err != nil {
		log.Fatal("failed to run app:", err)
	}
}

func setupTerminal() {

	messageMap = NewMessageMap()

	history = tview.NewList()
	history.SetSelectedFocusOnly(true).
		ShowSecondaryText(false).
		SetBorder(true).
		SetTitle("CHATROOM")

	allUser = tview.NewList().SetSelectedFocusOnly(true)
	allUser.ShowSecondaryText(false).
		SetBorder(true).
		SetTitle("ALL-USERS")

	input = tview.NewInputField().
		SetLabel(">> ").
		SetLabelColor(tcell.ColorPurple)

	input.SetDoneFunc(func(key tcell.Key) {
		if input.GetText() == "" {
			return
		}
		text := input.GetText()
		go func() {
			err := chatClient.Send(&message.Message{
				Sender:   userName,
				Content:  text,
				SendTime: time.Now().Unix(),
			})
			if err != nil {
				log.Fatal("failed to call message.Send:", err)
			}
		}()
		input.SetText("")
	})

	termFlex = tview.NewFlex().
		AddItem(allUser, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(history, 0, 3, false).
			AddItem(input, 0, 1, true),
			0, 5, false)

	glog.Infoln(logTag, `setupTerminal Success`)

}

func loopForMessages(ctx context.Context, rpcClient *client.Client) {
	//chatClient, err := .Messages(context.Background(), &push.MessagesRequest{
	//	ChatroomID: chatroomID,
	//})

	// query for potential missing messages
	//queryLatestMessages()

	var err error
	chatClient, err = rpcClient.Chat(ctx)
	if err != nil {
		log.Fatal("failed to call push.Messages:", err)
	}

	// loop recving latest messages
	for {
		reply, err := chatClient.Recv()
		if err != nil {
			log.Println(err)
			break
		}
		messageMap.Add(reply)
		terminal.QueueUpdateDraw(func() {
			updateHistory()
			updateUserList()
		})
	}
}

func setupLogin() {

	loginForm = tview.NewForm().
		AddInputField("UserName", "", 20, nil, nil).SetLabelColor(tcell.ColorDarkBlue).
		AddButton("Login", openChatroom).
		AddButton("Quit", func() {
			terminal.Stop()
		})
	loginForm.SetBorder(true).SetTitle("Please inter your userName to login").SetTitleAlign(tview.AlignCenter)

	terminal = tview.NewApplication().SetRoot(loginForm, true).SetFocus(loginForm)
}

func openChatroom() {

	// 用户登录
	userName = loginForm.GetFormItem(0).(*tview.InputField).GetText()

	loginRes, err := rpcClient.Login(context.Background(), userName)
	if err != nil {
		terminal.Stop()
	} else {
		if loginRes.State == false {
			terminal.Stop()
		}
	}

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
}

func updateHistory() {
	messageMap.Lock()
	defer messageMap.Unlock()

	keys := messageMap.sortedMessagesKeys()

	history.Clear()
	for _, k := range keys {
		msg := messageMap.messageM[k]
		history.AddItem(
			fmt.Sprintf(
				"%s <%s>: %s",
				time.Unix(0, msg.SendTime).Format("2006-01-02 15:04:05 MST"),
				msg.Sender,
				msg.Content,
			),
			"", 0, nil)
	}
	history.SetCurrentItem(-1)
}

func updateUserList() {

	userList, err := rpcClient.GetUserList(context.Background())
	if err != nil {
		terminal.Stop()
	}

	allUser.Clear()
	for _, userInfo := range userList.Users {
		allUser.AddItem(
			fmt.Sprintf(
				"用户： <%s>: 状态：%t",
				userInfo.UserName,
				userInfo.State,
			),
			"", 0, nil)
	}

	allUser.SetCurrentItem(-1)
}
