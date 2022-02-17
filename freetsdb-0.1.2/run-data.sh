#!/bin/bash

echo `date` nohup freetsd -config ./freetsd.conf > freetsd.log &
nohup freetsd -config ./freetsd.conf > freetsd.log &

