package bot

import (
	"bot/chat"
	"bot/cq"
	"bot/event"
	"bot/message"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"strconv"
	"time"
)

type IBot interface {
	Start() error
	SendMessage(target chat.Chat, msg *chat.MessageChain, callback func(msgId int32)) error
	SendTextMsg(target chat.Chat, msg string, callback func(msgId int32) error) error
	AddListener(eType string, l func(iEvent event.IEvent) error)
	ReplyMessage(targetMsgId int32, chat chat.Chat, msg *chat.MessageChain, callback func(msgId int32)) error
	ReplyText(targetMsgId int32, chat chat.Chat, msg string, callback func(msgId int32)) error
	Close() error
}

type JsonProcessor interface {
	ProcessJson(jsonObj map[string]json.RawMessage) (event.IEvent, error)
}

type WSBot struct {
	URL      string
	ws       *websocket.Conn
	stop     bool
	msg      chan string
	lMap     map[string][]func(iEvent event.IEvent) error
	jsonP    JsonProcessor
	echos    map[any]any
	echoProc map[any]*event.EchoProcessor
}

func (W *WSBot) ReplyText(targetMsgId int32, chat chat.Chat, msg string, callback func(msgId int32)) error {
	return W.SendTextMsg(chat, fmt.Sprintf("[CQ:%s,id=%v]%s", cq.Reply, targetMsgId, msg), nil)
}

func (W *WSBot) SendTextMsg(target chat.Chat, msg string, callback func(msgId int32) error) error {
	id, err := strconv.ParseInt(*target.Id(), 10, 64)
	if err != nil {
		log.Panic("转换消息目标ID为Int64时出现错误")
		return err
	}

	switch target.(type) {
	case chat.User:
		params := pMsgJson{
			Id:  id,
			Msg: msg,
		}

		return W.JSONAction(SendPMsgAction, params, callback, &event.MsgIDProcessor)
	case chat.Group:
		params := gMsgJson{
			Id:  id,
			Msg: msg,
		}

		return W.JSONAction(SendGMsgAction, params, callback, &event.MsgIDProcessor)
	}

	return err
}

func (W *WSBot) ReplyMessage(targetId int32, chat chat.Chat, msg *chat.MessageChain, callback func(msgId int32)) error {
	msg0 := *&msg
	msg0.Messages = append([]message.Message{message.Reply{Id: targetId}}, msg0.Messages...)
	return W.SendMessage(chat, msg0, callback)
}

type pMsgJson struct {
	Id  int64  `json:"user_id"`
	Msg string `json:"message"`
}

type gMsgJson struct {
	Id  int64  `json:"group_id"`
	Msg string `json:"message"`
}

type post struct {
	Action string `json:"action"`
	Params any    `json:"params"`
	Echo   any    `json:"echo"`
}

const (
	SendPMsgAction = "send_private_msg"
	SendGMsgAction = "send_group_msg"
)

func (W *WSBot) SendMessage(target chat.Chat, msg *chat.MessageChain, callback func(msgId int32)) error {
	id, err := strconv.ParseInt(*target.Id(), 10, 64)
	if err != nil {
		log.Panic("转换消息目标ID为Int64时出现错误")
		return err
	}

	switch target.(type) {
	case chat.User:
		params := pMsgJson{
			Id:  id,
			Msg: *cq.ToCQCode(msg),
		}

		return W.JSONAction(SendPMsgAction, params, callback, &event.MsgIDProcessor)
	case chat.Group:
		params := gMsgJson{
			Id:  id,
			Msg: *cq.ToCQCode(msg),
		}

		return W.JSONAction(SendGMsgAction, params, callback, &event.MsgIDProcessor)
	}

	return err
}

func (W *WSBot) JSONAction(action string, params any, callback any, processor *event.EchoProcessor) error {
	post := post{
		Action: action,
		Params: params,
	}

	if callback != nil {
		ra := rand.Int()
		post.Echo = ra
		W.echos[ra] = callback
		W.echoProc[ra] = processor
	}

	err := W.ws.WriteJSON(post)
	if err != nil {
		log.Panic("JSON上报时出现错误")
		return err
	}

	return nil
}

func NewWSBot(jsonP JsonProcessor, url string) *WSBot {
	return &WSBot{
		URL:      url,
		stop:     false,
		lMap:     make(map[string][]func(iEvent event.IEvent) error),
		jsonP:    jsonP,
		echos:    make(map[any]any),
		echoProc: make(map[any]*event.EchoProcessor),
	}
}

func (W *WSBot) Start() error {
	var err error
	log.Println("连接Websocket...")
	W.ws, _, err = websocket.DefaultDialer.Dial(W.URL, nil)
	if err != nil {
		//连接Websocket时出现错误

		W.stop = true
		log.Println("连接Websocket时出现错误。")
		return err
	}
	log.Printf("成功连接至%v。", W.URL)

	W.msg = make(chan string)
	go func() {
		for !W.stop {
			_, data, err := W.ws.ReadMessage()
			if err != nil {
				log.Fatal(err)
			} else {
				W.msg <- string(data)
			}
			time.Sleep(20)
		}
	}()

	go func() {
		for !W.stop {
			data, _ := <-W.msg
			//log.Println("接收数据\n", data)
			m := make(map[string]json.RawMessage)
			err := json.Unmarshal([]byte(data), &m)
			if err != nil {
				log.Fatal(err)
			} else {
				echo, ok := m["echo"]
				if ok {
					var d any
					err := json.Unmarshal(echo, &d)
					if err != nil {
						log.Panicln("检测Echo时出现错误:", err)
					}
					var data map[string]json.RawMessage
					err = json.Unmarshal(m["data"], &data)
					if err != nil {
						log.Panicln("解析回弹JSON时出现错误:", err)
					}

					proc, ok := W.echoProc[d]
					if ok {
						err := (*proc).Process(data, W.echos[d])
						if err != nil {
							return
						}
					}
					if err != nil {
						log.Panicln(err)
					}
				} else {
					event0, err := W.jsonP.ProcessJson(m)
					if err != nil {
						log.Panicln("处理JSON时出现错误:", err)
					} else {
						if event0 != nil {
							l, ok := W.lMap[event0.Type()]
							if ok {
								//遍历监听器
								for _, f := range l {
									f := f
									go func() {
										err := f(event0)
										if err != nil {
											log.Printf("监听事件%s时出现错误", event0.Type())
											log.Println(err)
										}
									}()
								}
							}
						}
					}
				}
			}
			time.Sleep(20)
		}
	}()

	return err
}

func (W *WSBot) AddListener(eType string, l func(iEvent event.IEvent) error) {
	list, ok := W.lMap[eType]
	if ok {
		list = append(list, l)
	} else {
		list = []func(iEvent event.IEvent) error{l}
	}
	W.lMap[eType] = list
}

func (W *WSBot) Close() error {
	defer close(W.msg)
	err := W.ws.Close()
	return err
}
