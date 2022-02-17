#!/bin/bash

echo `date` nohup freetsd -config ./freetsd.conf > ./logs/freetsd.log &
nohup freetsd -config ./freetsd.conf > ./logs/freetsd.log &

