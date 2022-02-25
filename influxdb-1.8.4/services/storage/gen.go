package storage

//go:generate protoc -I$GOPATH/src/influxdb.cluster/vendor -I. --gogofaster_out=. source.proto
