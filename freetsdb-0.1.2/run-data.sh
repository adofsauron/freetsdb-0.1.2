#!/bin/bash

echo `date` nohup freetsd -config ./conf/freetsd.conf > ./logs/freetsd.log &
nohup freetsd -config ./conf/freetsd.conf > ./logs/freetsd.log &

