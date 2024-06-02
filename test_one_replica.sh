echo "Compiling test..."
go build ./test.go

echo "Run Replica 0"
./test.exe 0 5000 > /dev/null

echo "OK: Test passed"
