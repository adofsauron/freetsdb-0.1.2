#!/bin/bash

echo `date` nohup influxd -config ./conf/influxd.conf > ./logs/influxd.log &
nohup influxd -config ./conf/influxd.conf > ./logs/influxd.log &

