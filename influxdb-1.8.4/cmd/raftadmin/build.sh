#!/bin/bash

echo `date` rm raftadmin -rf
rm raftadmin -rf

echo `date` go build -gcflags=all="-N -l" -o raftadmin ./cmd/raftadmin/raftadmin.go
go build -gcflags=all="-N -l" -o raftadmin ./cmd/raftadmin/raftadmin.go



