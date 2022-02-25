package tsdb

import (
	"influxdb.cluster/platform/query"
)

// EOF represents a "not found" key returned by a Cursor.
const EOF = query.ZeroTime
