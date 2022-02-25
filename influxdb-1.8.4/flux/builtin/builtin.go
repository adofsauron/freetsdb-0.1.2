// Package builtin ensures all packages related to Flux built-ins are imported and initialized.
// This should only be imported from main or test packages.
// It is a mistake to import it from any other package.
package builtin

import (
	"influxdb.cluster/services/flux"

	_ "influxdb.cluster/services/flux/functions/inputs"          // Import the built-in input functions
	_ "influxdb.cluster/services/flux/functions/outputs"         // Import the built-in output functions
	_ "influxdb.cluster/services/flux/functions/transformations" // Import the built-in transformations
	_ "influxdb.cluster/services/flux/options"                   // Import the built-in options
	_ "influxdb.cluster/flux/functions/inputs" // Import the built-in functions
)

func init() {
	flux.FinalizeBuiltIns()
}
