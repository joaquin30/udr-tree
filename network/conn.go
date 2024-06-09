package network

type CRDTTree interface {
	GetID() int
	ApplyRemoteOperation([]byte)
}

type ReplicaConn interface {
	Send([]byte)
	Connect()
	Disconnect()
	Close()
}
