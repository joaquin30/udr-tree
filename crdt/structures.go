package crdt

import (
	"sync"
	"github.com/google/uuid"
)

type LamportClock struct {
	sync.Mutex
	ID   uuid.UUID
	time uint64
}

func (this *LamportClock) Sync(time uint64) {
	this.Lock()
	defer this.Unlock()
	
	if time >= this.time {
		this.time = time + 1
	}
}

func (this *LamportClock) Now() uint64 {
	this.Lock()
	defer this.Unlock()
	
	now := this.time
	this.time += 1
	return now
}

type Move struct {
	ReplicaID uuid.UUID
	Timestamp uint64
	NewParent uuid.UUID
	NewName   string
	Node      uuid.UUID
}

// definir estructura de los bytes

func MoveFromBytes(data []byte) Move {
	return Move{}
}

func MoveToBytes(move Move) []byte {
	return make([]byte, 0)
}

type LogMove struct {
	ReplicaID uuid.UUID
	Timestamp uint64
	OldParent uuid.UUID
	OldName   string
	NewParent uuid.UUID
	NewName   string
	Node      uuid.UUID
}

func (this LogMove) Before(log LogMove) bool {
	if this.Timestamp == this.Timestamp {
		return this.ReplicaID.String() < log.ReplicaID.String()
	}
	
	return this.Timestamp < log.Timestamp
}
