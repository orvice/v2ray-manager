package v2raymanager

import (
	"context"
	"fmt"
	"github.com/orvice/kit/log"
	"google.golang.org/grpc"
	"v2ray.com/core/app/proxyman/command"
	statscmd "v2ray.com/core/app/stats/command"
	"v2ray.com/core/common/protocol"
	"v2ray.com/core/common/serial"
	"v2ray.com/core/proxy/vmess"
)

type Manager struct {
	client      command.HandlerServiceClient
	statsClient statscmd.StatsServiceClient

	inBoundTag string
	logger     log.Logger
}

const (
	UplinkFormat   = "user>>>%s>>>traffic>>>uplink"
	DownlinkFormat = "user>>>%s>>>traffic>>>downlink"
)

type TrafficInfo struct {
	Up, Down int64
}

func NewManager(addr, tag string) (*Manager, error) {
	cc, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client := command.NewHandlerServiceClient(cc)
	statsClient := statscmd.NewStatsServiceClient(cc)
	m := &Manager{
		client:      client,
		statsClient: statsClient,
		inBoundTag:  tag,
		logger:      log.NewDefaultLogger(),
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
	if err != nil && !IsAlreadyExistsError(err) {
		m.logger.Errorf("failed to call add user:  resp %v error %v", resp, err)
		return err
	}
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

// @todo error handle
func (m *Manager) GetTrafficAndReset(u User) TrafficInfo {
	ti := TrafficInfo{}
	ctx := context.Background()
	up, err := m.statsClient.GetStats(ctx, &statscmd.GetStatsRequest{
		Name:   fmt.Sprintf(UplinkFormat, u.GetEmail()),
		Reset_: true,
	})
	if err != nil && !IsNotFoundError(err) {
		m.logger.Errorf("get traffic user %v error %v", u, err)
		return ti
	}

	down, err := m.statsClient.GetStats(ctx, &statscmd.GetStatsRequest{
		Name:   fmt.Sprintf(DownlinkFormat, u.GetEmail()),
		Reset_: true,
	})
	if err != nil && !IsNotFoundError(err) {
		m.logger.Errorf("get traffic user %v error %v", u, err)
		return ti
	}

	if up != nil {
		ti.Up = up.Stat.Value
	}
	if down != nil {
		ti.Down = down.Stat.Value
	}
	return ti
}
