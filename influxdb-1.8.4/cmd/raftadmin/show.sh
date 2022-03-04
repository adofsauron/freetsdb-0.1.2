#!/bin/bash



echo `date` ./raftadmin --leader multi:///localhost:50051,localhost:50052,localhost:50053 get_configuration
./raftadmin --leader multi:///localhost:50051,localhost:50052,localhost:50053 get_configuration

echo -e "\n"

echo `date` ./raftadmin --leader multi:///localhost:50051,localhost:50052,localhost:50053 leader
./raftadmin --leader multi:///localhost:50051,localhost:50052,localhost:50053 leader

