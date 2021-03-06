package transformations

import (
	"time"

	"influxdb.cluster/services/flux"
	"influxdb.cluster/services/flux/semantic"
	"influxdb.cluster/services/flux/values"
)

var systemTimeFuncName = "systemTime"

func init() {
	nowFunc := SystemTime()
	flux.RegisterBuiltInValue(systemTimeFuncName, nowFunc)
}

// SystemTime return a function value that when called will give the current system time
func SystemTime() values.Value {
	name := systemTimeFuncName
	ftype := semantic.NewFunctionType(semantic.FunctionSignature{
		Return: semantic.Time,
	})
	call := func(args values.Object) (values.Value, error) {
		return values.NewTime(values.ConvertTime(time.Now().UTC())), nil
	}
	sideEffect := false
	return values.NewFunction(name, ftype, call, sideEffect)
}
