package v2raymanager

import (
	"context"
	"github.com/weeon/log"
	"go.uber.org/zap/zapcore"
	"os"
	"testing"
)

func TestManager(t *testing.T) {
	addr := os.Getenv("ADDR")
	l, _ := log.NewLogger("/dev/stdout", zapcore.DebugLevel)
	cli, err := NewManager(addr, "api", l)

	if err != nil {
		t.Error(err)
		return
	}
	resp, err := cli.GetUserList(context.Background(), false)
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("len %d", len(resp))

	for _, v := range resp {
		t.Log(v.User.GetUUID(), v.TrafficInfo.Down, v.TrafficInfo.Up)
	}

}
