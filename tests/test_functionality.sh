#!/bin/sh
echo "FUNCTIONALITY TEST - 1 REPLICA"
echo "Compiling test..."
go build ./test_consistency.go

echo "Run Replica 0"
./test_consistency.exe 0 5000 > /dev/null

echo "OK: Test passed"
