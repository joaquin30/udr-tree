echo "Compiling test..."
go build ./test.go

rm -f log0.txt log1.txt log2.txt

echo "Run Replica 0"
./test.exe 0 5000 'localhost:5001' 'localhost:5002' > log0.txt &
echo "Run Replica 1"
./test.exe 1 5001 'localhost:5000' 'localhost:5002' > log1.txt &
echo "Run Replica 2"
./test.exe 2 5002 'localhost:5000' 'localhost:5001' > log2.txt &

echo "Waiting for CRDTs eventual consistency..."
wait $(jobs -p)

diff log0.txt log1.txt > /dev/null
if [ $? -ne 0 ]; then
	echo "ERROR: Inconsistent CRDTs 0 and 1"
	exit 1
fi

diff log1.txt log2.txt > /dev/null
if [ $? -ne 0 ]; then
	echo "ERROR: Inconsistent CRDT 1 and 2"
	exit 1
fi

diff log0.txt log2.txt > /dev/null
if [ $? -ne 0 ]; then
	echo "ERROR: Inconsistent CRDT 0 and 2"
	exit 1
fi

echo "OK: Test passed"
