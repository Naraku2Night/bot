package tool

type Pair struct {
	Key   string
	Value string
}

var Plugins []*Plugin

type Plugin struct {
	Name   string
	Unload *func() error
}

func RegisterPlugin(name string, onUnload *func() error) error {
	Plugins = append(Plugins, &Plugin{
		Name:   name,
		Unload: onUnload,
	})

	return nil
}
