package options

import (
	"influxdb.cluster/services/flux"
	"influxdb.cluster/services/flux/functions/transformations"
)

func init() {
	flux.RegisterBuiltInOption("now", transformations.SystemTime())
}
