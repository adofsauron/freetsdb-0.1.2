package metrics

import (
	"testing"
	"time"

	"influxdb.cluster/pkg/testing/assert"
)

func TestTimer_Update(t *testing.T) {
	var c Timer
	c.Update(100 * time.Millisecond)
	assert.Equal(t, c.Value(), 100*time.Millisecond, "unexpected value")
}
