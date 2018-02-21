package v2raymanager

import (
	"context"
	"github.com/orvice/kit/log"
	"google.golang.org/grpc"
	"v2ray.com/core/app/proxyman/command"
	"v2ray.com/core/common/protocol"
	"v2ray.com/core/common/serial"
	"v2ray.com/core/proxy/vmess"
)

type Manager struct {
	client     command.HandlerServiceClient
	inBoundTag string

	logger log.Logger
}

func NewManager(addr, tag string) (*Manager, error) {
	cc, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client := command.NewHandlerServiceClient(cc)
	m := &Manager{
		client:     client,
		inBoundTag: tag,
		logger:     log.NewDefaultLogger(),
	}
	return m, nil
}

func (m *Manager) SetLogger(l log.Logger) {
	m.logger = l
}

func (m *Manager) AddUser(u User) error {
	resp, err := m.client.AlterInbound(context.Background(), &command.AlterInboundRequest{
		Tag: m.inBoundTag,
		Operation: serial.ToTypedMessage(&command.AddUserOperation{
			User: &protocol.User{
				Level: u.GetLevel(),
				Email: u.GetEmail(),
				Account: serial.ToTypedMessage(&vmess.Account{
					Id:               u.GetUUID(),
					AlterId:          u.GetAlterID(),
					SecuritySettings: &protocol.SecurityConfig{Type: protocol.SecurityType_AUTO},
				}),
			},
		}),
	})
	if err != nil {
		m.logger.Errorf("failed to call add user: %v", err)
		return nil
	}
	m.logger.Debugf("call add user resp: %v", resp)

	return nil
}

func (m *Manager) RemoveUser(u User) error {
	resp, err := m.client.AlterInbound(context.Background(), &command.AlterInboundRequest{
		Tag: m.inBoundTag,
		Operation: serial.ToTypedMessage(&command.RemoveUserOperation{
			Email: u.GetEmail(),
		}),
	})
	if err != nil {
		m.logger.Errorf("failed to call remove user : %v", err)
		return TODOErr
	}
	m.logger.Debugf("call remove user resp: %v", resp)

	return nil
}
