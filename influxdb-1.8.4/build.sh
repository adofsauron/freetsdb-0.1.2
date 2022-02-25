#!/bin/bash

HERE=`pwd`

cd $HERE
cd cmd/freets
echo `date` go build -gcflags=all="-N -l" -o influx ./main.go
go build -gcflags=all="-N -l" -o influx ./main.go
cp ./influx /usr/bin -f

cd $HERE
cd cmd/freetsd
echo `date` go build -gcflags=all="-N -l" -o influxd ./main.go
go build -gcflags=all="-N -l" -o influxd ./main.go
cp ./influxd /usr/bin -f

cd $HERE
cd cmd/freetsd-ctl
echo `date` go build -gcflags=all="-N -l" -o influxd-ctl ./main.go
go build -gcflags=all="-N -l" -o influxd-ctl ./main.go
cp ./influxd-ctl /usr/bin -f

cd $HERE
cd cmd/freetsd-meta
echo `date` go build -gcflags=all="-N -l" -o influxd-meta ./main.go
go build -gcflags=all="-N -l" -o influxd-meta ./main.go
cp ./influxd-meta /usr/bin -f

cd $HERE
cd cmd/freets_inspect
echo `date` go build -gcflags=all="-N -l" -o influx_inspect ./main.go
go build -gcflags=all="-N -l" -o influx_inspect ./main.go
cp ./influx_inspect /usr/bin -f

cd $HERE
cd cmd/freets_tools
echo `date` go build -gcflags=all="-N -l" -o influx_tools ./main.go
go build -gcflags=all="-N -l" -o influx_tools ./main.go
cp ./influx_tools /usr/bin -f

# cd $HERE
# cd cmd/freets_tsm
# echo `date` go build -gcflags=all="-N -l" -o influx_tsm ./main.go
# go build -gcflags=all="-N -l" -o influx_tsm ./main.go
# cp ./influx_tsm /usr/bin -f

cd $HERE
cd cmd/store
echo `date` go build -gcflags=all="-N -l" -o store ./main.go
go build -gcflags=all="-N -l" -o store ./main.go
cp ./store /usr/bin -f

cd $HERE

