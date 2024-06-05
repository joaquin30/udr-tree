#!/bin/sh
ID=0
OPS=2000
echo "STRESS TEST - 3 REPLICAS"
echo "Operations per second: $OPS"
echo "Compiling test..."
go build ./test_stress.go

# MQTT=mqtt://mqtt.eclipseprojects.io:1883
MQTT=mqtt://140.238.237.68:1883
# MQTT=mqtt://broker.hivemq.com:1883

echo "Run Replica 0"
./test_stress.exe 0 "$MQTT" "$OPS" > /dev/null &
echo "Run Replica 1"
./test_stress.exe 1 "$MQTT" "$OPS" > /dev/null &
echo "Run Replica 2"
./test_stress.exe 2 "$MQTT" "$OPS" > /dev/null &

echo "Waiting 1 minute..."
wait $(jobs -p)
echo "Finished"
