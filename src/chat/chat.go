package chat

import (
	"bot/message"
)

type MessageChain struct {
	Messages []message.Message
}

func NewMsgChain() *MessageChain {
	return &MessageChain{Messages: []message.Message{}}
}

func (c *MessageChain) Add(message2 message.Message) *MessageChain {
	c.Messages = append(c.Messages, message2)
	return c
}

func (c *MessageChain) AddText(text string) *MessageChain {
	c.Messages = append(c.Messages, message.NewText(&text))
	return c
}

type Chat interface {
	Name() *string
	Id() *string
}

type User struct {
	name *string
	id   *string
}

func NewUser(id string, name string) User {
	return User{name: &name, id: &id}
}

func (u User) Name() *string {
	return u.name
}

func (u User) Id() *string {
	return u.id
}

func NewGroup(id string) Group {
	return Group{id: &id}
}

type Group struct {
	id *string
}

func (g Group) Id() *string {
	return g.id
}

func (g Group) Name() *string {
	s := "err"
	return &s
}
