package message

const (
	TextT  = "text"
	ReplyT = "reply"
)

type Message interface {
	Type() string
}

type Text struct {
	Content *string
}

type Reply struct {
	Id int32
}

func (r Reply) Type() string {
	return ReplyT
}

func (t Text) Type() string {
	return TextT
}

func NewText(content *string) Text {
	return Text{Content: content}
}

func NewReply(id int32) Reply {
	return Reply{Id: id}
}
