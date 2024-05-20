package crdt

import (
	"github.com/google/uuid"
)

type LamportClock struct {
	id   uuid.UUID
	time uint64
}

func (this *LamportClock) Sync(clock LamportClock) {
	if clock.time >= this.time {
		this.time = clock.time + 1
	}
}

func (this *LamportClock) Now(clock LamportClock) LamportClock {
	ret := *this
	this.time += 1
	return ret
}

type Move struct {
	Timestamp LamportClock
	NewParent uuid.UUID
	NewName   string
	Node      uuid.UUID
}

// definir estructura de los bytes

func MoveFromBytes(data []byte) Move {

}

func MoveToBytes(move Move) []byte {

}

type LogMove struct {
	Timestamp LamportClock
	OldParent *uuid.UUID
	OldName   *string
	NewParent uuid.UUID
	NewName   string
	Node      UUID
}
