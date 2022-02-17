#!/bin/bash

HERE=`pwd`

cd $HERE
cd cmd/freets
echo `date` go build -o freets ./main.go
go build -o freets ./main.go
cp ./freets /usr/bin -f

cd $HERE
cd cmd/freetsd
echo `date` go build -o freetsd ./main.go
go build -o freetsd ./main.go
cp ./freetsd /usr/bin -f

cd $HERE
cd cmd/freetsd-ctl
echo `date` go build -o freetsd-ctl ./main.go
go build -o freetsd-ctl ./main.go
cp ./freetsd-ctl /usr/bin -f

cd $HERE
cd cmd/freetsd-meta
echo `date` go build -o freetsd-meta ./main.go
go build -o freetsd-meta ./main.go
cp ./freetsd-meta /usr/bin -f

cd $HERE
cd cmd/freets_inspect
echo `date` go build -o freets_inspect ./main.go
go build -o freets_inspect ./main.go
cp ./freets_inspect /usr/bin -f

cd $HERE
cd cmd/freets_tools
echo `date` go build -o freets_tools ./main.go
go build -o freets_tools ./main.go
cp ./freets_tools /usr/bin -f

# cd $HERE
# cd cmd/freets_tsm
# echo `date` go build -o freets_tsm ./main.go
# go build -o freets_tsm ./main.go
# cp ./freets_tsm /usr/bin -f

cd $HERE
cd cmd/store
echo `date` go build -o store ./main.go
go build -o store ./main.go
cp ./store /usr/bin -f

cd $HERE

