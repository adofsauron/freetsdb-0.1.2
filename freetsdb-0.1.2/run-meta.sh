#!/bin/bash

echo `date` freetsd-meta -config ./freetsd-meta.conf > ./logs/freetsd-meta.log &
nohup freetsd-meta -config ./freetsd-meta.conf > ./logs/freetsd-meta.log &

