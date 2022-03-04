#!/bin/bash

influxd-ctl add-meta 192.168.58.128:8091
influxd-ctl add-meta 192.168.58.131:8091
influxd-ctl add-meta 192.168.58.132:8091


influxd-ctl add-data 192.168.58.128:8088
influxd-ctl add-data 192.168.58.131:8088
influxd-ctl add-data 192.168.58.132:8088

# influxd-ctl remove-data 192.168.58.128:8088





raftadmin 192.168.58.128:50051 add_voter 192.168.58.132:50051 192.168.58.132:50051 0

raftadmin --leader multi:///192.168.58.128:50051,192.168.58.132:50051 add_voter  192.168.58.131:50051  192.168.58.131:50051 0

raftadmin 192.168.58.128:50051 leader

raftadmin 192.168.58.128:50051 get_configuration
