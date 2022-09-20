package bot

import (
	"encoding/json"
	"fmt"
	"github.com/Naraku2Night/bot/chat"
	"github.com/Naraku2Night/bot/event"
	"github.com/Naraku2Night/bot/message"
	"log"
	"math"
	"strconv"
	"strings"
)

const (
	MessagePost   = "message"
	MetaEventPost = "meta_event"
)

var GoCQBot *GoCQ

type GoCQ struct {
	*WSBot
	aliasMap  map[string]*Command
	Commands  []*Command
	CmdPrefix string
}

type GoCQSender struct {
	Age      int     `json:"age"`
	Nickname *string `json:"nickname"`
	Sex      *string `json:"sex"`
	UserId   *int64  `json:"user_id"`
}

type GoCQMsg struct {
	Type *string                    `json:"type"`
	Data map[string]json.RawMessage `json:"data"`
}

type Command struct {
	Bot         *GoCQ
	Head        string
	Alias       []string
	Executor    *func(bot *GoCQ, user chat.User, params []string, source chat.Chat, msgEvent *event.MessageEvent) error
	Description *string
	Usage       *string
}

func (c *Command) SetBot(bot *GoCQ) {
	c.Bot = bot
}

func (c *Command) SetDescription(string2 string) *Command {
	c.Description = &string2
	return c
}

func (c *Command) SetUsage(usage string) *Command {
	c.Usage = &usage
	return c
}

func (c *Command) AddAlia(alia string) *Command {
	c.Alias = append(c.Alias, alia)

	if c.Bot != nil {
		c.Bot.aliasMap[alia] = c
	}

	return c
}

func NewCommand(head string,
	exe func(bot *GoCQ, user chat.User, params []string, source chat.Chat, msgEvent *event.MessageEvent) error) *Command {
	return &Command{
		Head:     head,
		Alias:    []string{},
		Executor: &exe,
	}
}

type jsonP struct{}

func (c jsonP) ProcessJson(jsonObj map[string]json.RawMessage) (event.IEvent, error) {
	var s string
	err := json.Unmarshal(jsonObj["post_type"], &s)
	if err != nil {
		return nil, err
	}
	switch s {
	case MessagePost:
		//消息事件
		//处理信息列表
		var msgList []GoCQMsg
		err := json.Unmarshal(jsonObj["message"], &msgList)
		if err != nil {
			return nil, err
		}
		chain, _ := processMsg(msgList)

		//处理发送者
		var sender GoCQSender
		err = json.Unmarshal(jsonObj["sender"], &sender)
		if err != nil {
			return nil, err
		}

		//处理群ID
		var groupId int64
		d, ok := jsonObj["group_id"]
		if ok {
			err := json.Unmarshal(d, &groupId)
			if err != nil {
				return nil, err
			}
		}

		//消息来源
		var source chat.Chat
		user := chat.NewUser(strconv.FormatInt(*sender.UserId, 10), *sender.Nickname)
		if ok {
			group := strconv.FormatInt(groupId, 10)
			source = chat.NewGroup(group)
		} else {
			source = user
		}

		//消息ID
		var msgID int32
		err = json.Unmarshal(jsonObj["message_id"], &msgID)
		if err != nil {
			return nil, err
		}

		var raw string
		err = json.Unmarshal(jsonObj["raw_message"], &raw)
		if err != nil {
			return nil, err
		} else {
			log.Printf("来自%s(%v)的消息:%s", *sender.Nickname, *sender.UserId, raw)
		}

		return &event.MessageEvent{
			Source:  source,
			Sender:  user,
			MsgId:   msgID,
			Msg:     &chain,
			RawMsg:  &raw,
			IsGroup: ok,
		}, nil
	case MetaEventPost:
		//元事件
		var mType string
		err := json.Unmarshal(jsonObj["meta_event_type"], &mType)
		if err != nil {
			return nil, err
		}

		switch mType {
		case event.HeartBeat:
			var interval int64
			err := json.Unmarshal(jsonObj["interval"], &interval)
			if err != nil {
				log.Println("处理心跳事件时出现错误:")
				return nil, err
			}
		default:
			return nil, nil
		}
	}

	return nil, err
}

func (c *GoCQ) AddCommand(cmd *Command) {
	c.aliasMap[cmd.Head] = cmd

	for _, alia := range cmd.Alias {
		c.aliasMap[alia] = cmd
	}

	cmd.SetBot(c)
	c.Commands = append(c.Commands, cmd)
}

func processMsg(msg []GoCQMsg) (chat.MessageChain, error) {
	var msg0 []message.Message

	var err error = nil
	for _, m := range msg {
		switch *m.Type {
		case message.TextT:
			//文本消息
			var data string
			err := json.Unmarshal(m.Data["text"], &data)
			if err != nil {
				return chat.MessageChain{}, err
			}
			msg0 = append(msg0, &message.Text{Content: &data})
		}
	}

	return chat.MessageChain{Messages: msg0}, err
}

const CommandPerPage = 4

func NewGoCQBot(url string, cmdPrefix string) *GoCQ {
	bot := &GoCQ{WSBot: NewWSBot(jsonP{}, url),
		aliasMap:  make(map[string]*Command),
		CmdPrefix: cmdPrefix,
		Commands:  []*Command{},
	}

	//指令监听器
	bot.AddListener(event.Message, func(iEvent event.IEvent) error {
		e := (iEvent).(*event.MessageEvent)
		msg := e.Msg.Messages[0]
		if msg.Type() == message.TextT {
			text := *msg.(*message.Text).Content
			if strings.HasPrefix(text, cmdPrefix) {
				fields := strings.Fields(strings.TrimPrefix(text, cmdPrefix))
				if len(fields) > 0 {
					head := fields[0]
					params := fields[1:]
					cmd, ok := bot.aliasMap[head]

					if ok {
						return (*cmd.Executor)(cmd.Bot, e.Sender, params, e.Source, e)
					} else {
						err := bot.ReplyMessage(e.MsgId, e.Source, chat.NewMsgChain().AddText(fmt.Sprintf("未知指令，发送%shelp查看帮助。", bot.CmdPrefix)), nil)
						if err != nil {
							return err
						}
					}
				} else {
					err := bot.SendMessage(e.Source, chat.NewMsgChain().AddText(fmt.Sprintf("发送%shelp查看指令帮助列表。", bot.CmdPrefix)), nil)
					if err != nil {
						return err
					}
				}
			}

		}
		return nil
	})

	help := NewCommand("help", func(bot *GoCQ, user chat.User, params []string, source chat.Chat, msgEvent *event.MessageEvent) error {

		if len(params) == 0 {
			return bot.SendTextMsg(source, bot.GetCmdHelpList(0), nil)
		} else {
			pIndex, err := strconv.Atoi(params[0])

			if err != nil {
				sb := strings.Builder{}
				cmd, ok := bot.GetCommand(params[0])

				if ok {
					sb.WriteString("[]内为可选参数，()内为必要参数\n")
					writeCommandHelp(&sb, bot.CmdPrefix, cmd)
				} else {
					sb.WriteString("指令不存在。")
				}

				return bot.ReplyText(msgEvent.MsgId, source, sb.String(), nil)
			} else {
				s := bot.GetCmdHelpList(pIndex - 1)

				if s != "" {
					return bot.SendTextMsg(source, s, nil)
				} else {
					return bot.ReplyText(msgEvent.MsgId, source, "页码不存在。", nil)
				}

			}
		}
	}).SetDescription("查看指令帮助(列表)").SetUsage("help [指令名/别名/页码]")

	bot.AddCommand(help)
	return bot
}

func (c *GoCQ) GetCommand(name string) (*Command, bool) {
	cmd, ok := c.aliasMap[name]
	return cmd, ok
}

// GetCmdHelpList TODO 帮助列表缓存
// GetCmdHelpList pageIndex从0开始
func (c *GoCQ) GetCmdHelpList(pageIndex int) string {
	start := pageIndex * CommandPerPage
	cmdCount := len(c.Commands)
	if cmdCount > start && pageIndex >= 0 {
		sb := strings.Builder{}
		sb.WriteString("[]内为可选参数，()内为必要参数")
		var end int
		if start+CommandPerPage < cmdCount {
			end = start + CommandPerPage + 1
		} else {
			end = cmdCount
		}
		for i := start; i < end; i++ {
			writeCommandHelp(&sb, c.CmdPrefix, c.Commands[i])
		}

		sb.WriteString(fmt.Sprintf("\n第%v页，共%v页", pageIndex+1, int(math.Ceil(float64(cmdCount)/float64(CommandPerPage)))))
		return sb.String()
	}

	return ""
}

func writeCommandHelp(sb *strings.Builder, prefix string, cmd *Command) *strings.Builder {
	sb.WriteString(fmt.Sprintf("\n%s%s\n%s", prefix, *cmd.Usage, *cmd.Description))

	if len(cmd.Alias) > 0 {
		sb.WriteString("\n别名:")
		sb.WriteString(cmd.Alias[0])
		l := len(cmd.Alias)
		for i := 1; i < l; i++ {
			sb.WriteString(",")
			sb.WriteString(cmd.Alias[i])
		}
	}

	return sb
}
