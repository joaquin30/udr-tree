#!/bin/sh
echo "CONSISTENCY TEST - 3 REPLICAS"
echo "Compiling test..."
go build ./test_consistency.go

rm -f log0.txt log1.txt log2.txt
# MQTT=mqtt://140.238.237.68:1883
MQTT=mqtt://broker.hivemq.com:1883

echo "Run Replica 0"
./test_consistency.exe 0 "$MQTT" > log0.txt &
echo "Run Replica 1"
./test_consistency.exe 1 "$MQTT" > log1.txt &
echo "Run Replica 2"
./test_consistency.exe 2 "$MQTT" > log2.txt &

echo "Waiting for CRDTs eventual consistency..."
wait $(jobs -p)

if ! diff log0.txt log1.txt > /dev/null; then
	echo "ERROR: Inconsistent CRDTs 0 and 1"
	exit 1
fi

if ! diff log1.txt log2.txt > /dev/null; then
	echo "ERROR: Inconsistent CRDT 1 and 2"
	exit 1
fi

if ! diff log0.txt log2.txt > /dev/null; then
	echo "ERROR: Inconsistent CRDT 0 and 2"
	exit 1
fi

echo "OK: Test passed"
