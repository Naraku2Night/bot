package event

import (
	"encoding/json"
	"github.com/Naraku2Night/bot/chat"
)

const (
	HeartBeat = "heartbeat"
	Message   = "message"
)

type EchoProcessor interface {
	Process(data map[string]json.RawMessage, fun any) error
}

type msgIDProcessor struct{}

var MsgIDProcessor EchoProcessor = msgIDProcessor{}

func (m msgIDProcessor) Process(data map[string]json.RawMessage, fun any) error {
	var id int32
	err := json.Unmarshal(data["message_id"], &id)
	if err != nil {
		return err
	}

	return fun.(func(int322 int32) error)(id)
}

type IEvent interface {
	Type() string
}

type HeartBeatEvent struct {
	IntervalMS int64
}

func (h *HeartBeatEvent) Type() string {
	return HeartBeat
}

type MessageEvent struct {
	Source  chat.Chat
	Sender  chat.User
	MsgId   int32
	Msg     *chat.MessageChain
	RawMsg  *string
	IsGroup bool
}

func (m *MessageEvent) Type() string {
	return Message
}
