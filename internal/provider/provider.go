package provider

type ChatProvider interface {
	GetPinnedChats() ([]*string, error)
}

type service struct {
	chatProvider ChatProvider
}

func NewService(s ChatProvider) *service {
	return &service{
		chatProvider: s,
	}
}
