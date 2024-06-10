# UDR-Tree

Implementation of "A highly-available move operation for replicated trees"

Link: <https://martin.kleppmann.com/2021/10/07/crdt-tree-move-operation.html>

## Dependencies

- <https://github.com/google/uuid>
- <https://github.com/vmihailenco/msgpack>

## App

The app is a simple command line program demostrating operations of the CRDT. Build the app with `go build` and run it with `./udr-tree [id] [server_ip]`. Type `help` to learn the commands.

## Tests

For both tests a MQTT server or the server in the file `network/causal-server.go` must be running, locally or in a remote server. The IP of the server must be specified on the scripts

- `tests/test_consistency.sh` does random operations and connects/disconnects to test the consistency between replicas.
- `tests/test_stress.sh` does random operation at a certain rate per second, testing the performance of the replicas and server. The operations per second must be specified in the script. For testing in diferent machines you must run the commands in the script manually.
