package p2p

type MessageType byte

const (
	MsgTx        MessageType = 0x01
	MsgBlock     MessageType = 0x02
	MsgGetBlocks MessageType = 0x03
)

type Message struct {
	Type MessageType
	Data []byte
}
