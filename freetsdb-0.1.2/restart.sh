#!/bin/bash

echo pkill freetsd
pkill freetsd

echo `date` sleep 2s
sleep 2s

echo `date` rm /root/.freetsdb -rf
rm /root/.freetsdb -rf

echo `date` sleep 2s
sleep 2s

echo `date` bash ./run-meta.sh
bash ./run-meta.sh

echo `date` sleep 2s
sleep 2s

echo `date` freetsd-ctl add-meta localhost:8091
freetsd-ctl add-meta localhost:8091

echo `date` bash ./run-data.sh
bash ./run-data.sh

echo `date` sleep 2s
sleep 2s

echo `date` freetsd-ctl add-data localhost:8088
freetsd-ctl add-data localhost:8088

echo `date` freetsd-ctl show
freetsd-ctl show

