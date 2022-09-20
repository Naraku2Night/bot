package cq

import (
	"fmt"
	"github.com/Naraku2Night/bot/chat"
	"github.com/Naraku2Night/bot/message"
	"github.com/Naraku2Night/bot/tool"
)

const (
	Reply = "reply"
)

func getCQCode(typeS string, params ...tool.Pair) *string {
	s := "[CQ:" + typeS
	for _, v := range params {
		s += fmt.Sprintf(",%v=%v", v.Key, v.Value)
	}
	s += "]"
	return &s
}

func ToCQCode(msg *chat.MessageChain) *string {
	code := ""
	for _, msg0 := range msg.Messages {
		switch (msg0).(type) {
		case message.Text:
			code += *(msg0.(message.Text).Content)
		case message.Reply:
			code += *getCQCode(Reply, tool.Pair{
				Key:   "id",
				Value: string(msg0.(message.Reply).Id),
			})
		}
	}

	return &code
}
