#!/bin/bash

echo pkill influxd
pkill influxd

echo `date` sleep 2s
sleep 2s

echo -e "\n"

# echo `date` rm /data/influxdb -rf
# rm /data/influxdb -rf

# echo `date` sleep 2s
# sleep 2s

# echo -e "\n"

echo `date` bash ./run-meta.sh
bash ./run-meta.sh

echo `date` sleep 2s
sleep 2s

echo -e "\n"

echo `date` bash ./run-data.sh
bash ./run-data.sh

echo `date` sleep 2s
sleep 2s

echo -e "\n"


echo `date` influxd-ctl show
influxd-ctl show

echo -e "\n"

echo `date` "ps -ef | grep influxd | grep -v grep"
ps -ef | grep influxd | grep -v grep