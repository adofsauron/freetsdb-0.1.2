package coordinator_test

import (
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"influxdb.cluster/coordinator"
)

func TestConfig_Parse(t *testing.T) {
	// Parse configuration.
	var c coordinator.Config
	if _, err := toml.Decode(`
shard-writer-timeout = "10s"
write-timeout = "20s"
`, &c); err != nil {
		t.Fatal(err)
	}

	// Validate configuration.
	if time.Duration(c.ShardWriterTimeout) != 10*time.Second {
		t.Fatalf("unexpected shard-writer timeout: %s", c.ShardWriterTimeout)
	} else if time.Duration(c.WriteTimeout) != 20*time.Second {
		t.Fatalf("unexpected write timeout s: %s", c.WriteTimeout)
	}
}
