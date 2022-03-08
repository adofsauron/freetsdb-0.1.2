#!/bin/bash

HERE=`pwd`

cd $HERE
cd cmd/freets
rm influx -rf

echo `date` go build -gcflags=all="-N -l" -o influx ./main.go
go build -gcflags=all="-N -l" -o influx ./main.go

echo `date` cp ./influx /usr/bin -f
rm /usr/bin/influx -rf
cp ./influx /usr/bin -f

echo -e "\n"

cd $HERE
cd cmd/freetsd
rm influxd -rf

echo `date` go build -gcflags=all="-N -l" -o influxd ./main.go
go build -gcflags=all="-N -l" -o influxd ./main.go

echo `date` cp ./influxd /usr/bin -f
rm /usr/bin/influxd -rf
cp ./influxd /usr/bin -f

echo -e "\n"

cd $HERE
cd cmd/freetsd-ctl
rm influxd-ctl -rf

echo `date` go build -gcflags=all="-N -l" -o influxd-ctl ./main.go
go build -gcflags=all="-N -l" -o influxd-ctl ./main.go

echo `date` cp ./influxd-ctl /usr/bin -f
rm /usr/bin/influxd-ctl -rf
cp ./influxd-ctl /usr/bin -f

echo -e "\n"

cd $HERE
cd cmd/freetsd-meta
rm influxd-meta -rf

echo `date` go build -gcflags=all="-N -l" -o influxd-meta ./main.go
go build -gcflags=all="-N -l" -o influxd-meta ./main.go

echo `date` cp ./influxd-meta /usr/bin -f
rm /usr/bin/influxd-meta -rf
cp ./influxd-meta /usr/bin -f

echo -e "\n"

cd $HERE
cd cmd/freets_inspect
rm influx_inspect -rf

echo `date` go build -gcflags=all="-N -l" -o influx_inspect ./main.go
go build -gcflags=all="-N -l" -o influx_inspect ./main.go

echo `date` cp ./influx_inspect /usr/bin -f
rm /usr/bin/influx_inspect -rf
cp ./influx_inspect /usr/bin -f

echo -e "\n"

cd $HERE
cd cmd/freets_tools
rm influx_tools -rf

echo `date` go build -gcflags=all="-N -l" -o influx_tools ./main.go
go build -gcflags=all="-N -l" -o influx_tools ./main.go

echo `date` cp ./influx_tools /usr/bin -f
rm /usr/bin/influx_tools -rf
cp ./influx_tools /usr/bin -f

echo -e "\n"

# cd $HERE
# cd cmd/freets_tsm
# echo `date` go build -gcflags=all="-N -l" -o influx_tsm ./main.go
# go build -gcflags=all="-N -l" -o influx_tsm ./main.go
# cp ./influx_tsm /usr/bin -f

cd $HERE
cd cmd/store
echo `date` go build -gcflags=all="-N -l" -o store ./main.go
go build -gcflags=all="-N -l" -o store ./main.go

echo `date` cp ./store /usr/bin -f
rm /usr/bin/store -rf
cp ./store /usr/bin -f

echo -e "\n"

cd $HERE


cd cmd/raftadmin
rm raftadmin -rf

echo `date` go build -gcflags=all="-N -l" -o raftadmin ./cmd/raftadmin/raftadmin.go
go build -gcflags=all="-N -l" -o raftadmin ./cmd/raftadmin/raftadmin.go

echo `date` cp ./raftadmin /usr/bin -f
rm /usr/bin/raftadmin -rf
cp ./raftadmin /usr/bin -f

echo -e "\n"

cd $HERE

