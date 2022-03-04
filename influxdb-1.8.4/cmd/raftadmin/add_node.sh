#!/bin/bash

echo `date` ./raftadmin localhost:50051 add_voter localhost:50052 localhost:50052 0
./raftadmin localhost:50051 add_voter localhost:50052 localhost:50052 0


echo `date` ./raftadmin --leader multi:///localhost:50051,localhost:50052 add_voter localhost:50053 localhost:50053 0
./raftadmin --leader multi:///localhost:50051,localhost:50052 add_voter localhost:50053 localhost:50053 0

