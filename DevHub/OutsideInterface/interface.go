package OutsideInterface

type SubMessage struct {
	Value string
	Key   string
}

type Interface interface {
	UpdateComponent(key string, value string)
	RegisterWritableComponent(key string) <-chan SubMessage
}
