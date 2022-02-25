#!/bin/bash

influxd-ctl add-meta 192.168.58.128:8091
influxd-ctl add-meta 192.168.58.131:8091
influxd-ctl add-meta 192.168.58.132:8091


influxd-ctl add-data 192.168.58.128:8088
influxd-ctl add-data 192.168.58.131:8088
influxd-ctl add-data 192.168.58.132:8088

