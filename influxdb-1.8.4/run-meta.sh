#!/bin/bash

echo `date` influxd-meta -config ./conf/influxd-meta.conf > ./logs/influxd-meta.log &
nohup influxd-meta -config ./conf/influxd-meta.conf > ./logs/influxd-meta.log &

