package tunnel

import (
	"github.com/google/uuid"
	"github.com/brudi/go-micro/transport"
)

func newLink(s transport.Socket) *link {
	return &link{
		Socket: s,
		id:     uuid.New().String(),
	}
}
