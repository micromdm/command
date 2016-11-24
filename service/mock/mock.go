// Package mock implements command.Service and provides testing utilities.
package mock

import (
	"errors"

	"github.com/micromdm/mdm"
	"golang.org/x/net/context"
)

type CommandService struct {
	NewCommandInvoked bool
	NewCommandFunc    NewCommandFunc
}

type NewCommandFunc func(context.Context, *mdm.CommandRequest) (*mdm.Payload, error)

func (svc *CommandService) NewCommand(ctx context.Context, request *mdm.CommandRequest) (*mdm.Payload, error) {
	svc.NewCommandInvoked = true
	return svc.NewCommandFunc(ctx, request)
}

var MockPayload = &mdm.Payload{
	CommandUUID: "1234",
}

func FailNewCommand(context.Context, *mdm.CommandRequest) (*mdm.Payload, error) {
	return nil, errors.New("command creation failed")
}

func ReturnMockPayload(context.Context, *mdm.CommandRequest) (*mdm.Payload, error) {
	return MockPayload, nil
}

func ReturnPayload(p *mdm.Payload) NewCommandFunc {
	return func(context.Context, *mdm.CommandRequest) (*mdm.Payload, error) {
		return p, nil
	}
}
