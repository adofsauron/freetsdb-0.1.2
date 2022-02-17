#!/bin/bash

echo `date` freetsd-meta -config ./conf/freetsd-meta.conf > ./logs/freetsd-meta.log &
nohup freetsd-meta -config ./conf/freetsd-meta.conf > ./logs/freetsd-meta.log &

