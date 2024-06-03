#!/bin/sh
OPS=4000
echo "STRESS TEST - 3 REPLICAS"
echo "Operations per second: $OPS"
echo "Compiling test..."
go build ./test_stress.go

echo "Run Replica 0"
./test_stress.exe "$OPS" 0 5000 'localhost:5001' 'localhost:5002' > log0.txt &
echo "Run Replica 1"
./test_stress.exe "$OPS" 1 5001 'localhost:5000' 'localhost:5002' > log1.txt &
echo "Run Replica 2"
./test_stress.exe "$OPS" 2 5002 'localhost:5000' 'localhost:5001' > log2.txt &

echo "Waiting 1 minute..."
wait $(jobs -p)

SUM0=$(awk '{ sum += $1 } END { print sum }' log0.txt)
CNT0=$(wc -l < log0.txt)
SUM1=$(awk '{ sum += $1 } END { print sum }' log1.txt)
CNT1=$(wc -l < log1.txt)
SUM2=$(awk '{ sum += $1 } END { print sum }' log2.txt)
CNT2=$(wc -l < log2.txt)
# echo $SUM0 $CNT0 $SUM1 $CNT1 $SUM2 $CNT2
PROM=$(((SUM0 + SUM1 + SUM2) / (CNT0 + CNT1 + CNT2)))

echo "Wanted operations per second: $OPS"
echo "Real operations per second:   $PROM"
