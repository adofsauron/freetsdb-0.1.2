package precreator_test

import (
	"os"
	"testing"
	"time"

	"influxdb.cluster/internal"
	"influxdb.cluster/logger"
	"influxdb.cluster/services/precreator"
	"influxdb.cluster/toml"
)

func TestShardPrecreation(t *testing.T) {
	done := make(chan struct{})
	precreate := false

	var mc internal.MetaClientMock
	mc.PrecreateShardGroupsFn = func(now, cutoff time.Time) error {
		if !precreate {
			close(done)
			precreate = true
		}
		return nil
	}

	s := NewTestService()
	s.MetaClient = &mc

	if err := s.Open(); err != nil {
		t.Fatalf("unexpected open error: %s", err)
	}
	defer s.Close() // double close should not cause a panic

	timer := time.NewTimer(100 * time.Millisecond)
	select {
	case <-done:
		timer.Stop()
	case <-timer.C:
		t.Errorf("timeout exceeded while waiting for precreate")
	}

	if err := s.Close(); err != nil {
		t.Fatalf("unexpected close error: %s", err)
	}
}

func NewTestService() *precreator.Service {
	config := precreator.NewConfig()
	config.CheckInterval = toml.Duration(10 * time.Millisecond)

	s := precreator.NewService(config)
	s.WithLogger(logger.New(os.Stderr))
	return s
}
