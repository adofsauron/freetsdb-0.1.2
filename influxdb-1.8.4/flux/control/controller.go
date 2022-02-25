package control

import (
	"influxdb.cluster/services/flux/control"
	"influxdb.cluster/services/flux/execute"
	_ "influxdb.cluster/flux/builtin"
	"influxdb.cluster/flux/functions/inputs"
	fstorage "influxdb.cluster/platform/query/functions/inputs/storage"
	"go.uber.org/zap"
)

type MetaClient = inputs.MetaClient
type Authorizer = inputs.Authorizer

func NewController(mc MetaClient, reader fstorage.Reader, auth Authorizer, authEnabled bool, logger *zap.Logger) *control.Controller {
	// flux
	var (
		concurrencyQuota = 10
		memoryBytesQuota = 1e6
	)

	cc := control.Config{
		ExecutorDependencies: make(execute.Dependencies),
		ConcurrencyQuota:     concurrencyQuota,
		MemoryBytesQuota:     int64(memoryBytesQuota),
		Logger:               logger,
	}

	err := inputs.InjectFromDependencies(cc.ExecutorDependencies, inputs.Dependencies{Reader: reader, MetaClient: mc, Authorizer: auth, AuthEnabled: authEnabled})
	if err != nil {
		panic(err)
	}

	err = inputs.InjectBucketDependencies(cc.ExecutorDependencies, inputs.BucketDependencies{MetaClient: mc, Authorizer: auth, AuthEnabled: authEnabled})
	if err != nil {
		panic(err)
	}

	return control.New(cc)
}
