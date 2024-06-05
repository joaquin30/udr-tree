#!/bin/sh
echo "FUNCTIONALITY TEST - 1 REPLICA"
echo "Compiling test..."
go build ./test_consistency.go

MQTT=mqtt://test.mosquitto.org:1883

echo "Run Replica 0"
./test_consistency.exe 0 "$MQTT" > /dev/null

echo "OK: Test passed"
