// Package engine can be imported to initialize and register all available TSDB engines.
//
// Alternatively, you can import any individual subpackage underneath engine.
package engine // import "influxdb.cluster/tsdb/engine"

import (
	// Initialize and register tsm1 engine
	_ "influxdb.cluster/tsdb/engine/tsm1"
)
