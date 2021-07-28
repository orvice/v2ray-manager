package v2raymanager

import (
	"context"
	"fmt"
	"github.com/v2fly/v2ray-core/v4/app/proxyman/command"
	statscmd "github.com/v2fly/v2ray-core/v4/app/stats/command"
	"github.com/v2fly/v2ray-core/v4/common/protocol"
	"github.com/v2fly/v2ray-core/v4/common/serial"
	"github.com/v2fly/v2ray-core/v4/proxy/vmess"
	"github.com/weeon/contract"
	"github.com/weeon/log"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"strings"
)

type Manager struct {
	client      command.HandlerServiceClient
	statsClient statscmd.StatsServiceClient

	inBoundTag string
	logger     contract.Logger
}

const (
	UplinkFormat   = "user>>>%s>>>traffic>>>uplink"
	DownlinkFormat = "user>>>%s>>>traffic>>>downlink"
)

type TrafficInfo struct {
	Up, Down int64
}

func NewManager(addr, tag string, l contract.Logger) (*Manager, error) {
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
		logger:      l,
	}
	if m.logger == nil {
		m.logger, _ = log.NewLogger("/app/log/sdk.log", zapcore.DebugLevel)
	}
	return m, nil
}

func (m *Manager) SetLogger(l contract.Logger) {
	m.logger = l
}

// return is exist,and error
func (m *Manager) AddUser(ctx context.Context, u User) (bool, error) {
	resp, err := m.client.AlterInbound(ctx, &command.AlterInboundRequest{
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
		m.logger.Errorw("failed to call add user",
			"resp", resp,
			"error", err,
		)
		return false, err
	}
	return IsAlreadyExistsError(err), nil
}

func (m *Manager) RemoveUser(ctx context.Context, u User) error {
	resp, err := m.client.AlterInbound(ctx, &command.AlterInboundRequest{
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
func (m *Manager) GetTrafficAndReset(ctx context.Context, u User) (TrafficInfo, error) {
	ti := TrafficInfo{}
	up, err := m.statsClient.GetStats(ctx, &statscmd.GetStatsRequest{
		Name:   fmt.Sprintf(UplinkFormat, u.GetEmail()),
		Reset_: true,
	})
	if err != nil && !IsNotFoundError(err) {
		m.logger.Errorf("get traffic user %v error %v", u, err)
		return ti, err
	}

	down, err := m.statsClient.GetStats(ctx, &statscmd.GetStatsRequest{
		Name:   fmt.Sprintf(DownlinkFormat, u.GetEmail()),
		Reset_: true,
	})
	if err != nil && !IsNotFoundError(err) {
		m.logger.Errorw("get traffic user fail",
			"user", u,
			"error", err)
		return ti, nil
	}

	if up != nil {
		ti.Up = up.Stat.Value
	}
	if down != nil {
		ti.Down = down.Stat.Value
	}
	return ti, nil
}

func (m *Manager) GetUserList(ctx context.Context) ([]User, error) {
	resp, err := m.statsClient.QueryStats(ctx, &statscmd.QueryStatsRequest{
		Reset_: false,
	})
	if err != nil {
		return nil, err
	}

	var users = make([]User, 0, len(resp.Stat)/2)

	for _, v := range resp.Stat {

		if !strings.Contains(v.GetName(), "downlink") {
			continue
		}
		email := getEmailFromStatName(v.GetName())

		users = append(users, user{
			email: email,
			uuid:  getUUDIFromEmail(email),
		})
	}

	return users, nil
}

func getEmailFromStatName(s string) string {
	arr := strings.Split(s, ">>>")
	if len(arr) > 1 {
		return arr[1]
	}
	return s
}

func getUUDIFromEmail(s string) string {
	arr := strings.Split(s, "@")
	if len(arr) > 0 {
		return arr[0]
	}
	return s
}
