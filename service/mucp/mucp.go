// Package mucp initialises a mucp service
package mucp

import (
	// TODO: change to go-micro/service
	"github.com/brudi/go-micro"
	cmucp "github.com/brudi/go-micro/client/mucp"
	smucp "github.com/brudi/go-micro/server/mucp"
)

// NewService returns a new mucp service
func NewService(opts ...micro.Option) micro.Service {
	options := []micro.Option{
		micro.Client(cmucp.NewClient()),
		micro.Server(smucp.NewServer()),
	}

	options = append(options, opts...)

	return micro.NewService(options...)
}
