package types

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
)

type NodeID struct {
	Provider string `json:"provider"`
	ID       uint64 `json:"id"`
}

func (n *NodeID) ToBytes() []byte {
	var buf = make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(n.ID))

	return append(common.Hex2Bytes(n.Provider), buf...)
}

type Proof struct {
	NodeID
	Nonce int64 `json:"nonce"`
}

func (p *Proof) ToBytes() []byte {
	var nonceBuf = make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceBuf, uint64(p.Nonce))
	buf := p.NodeID.ToBytes()

	return append(buf, nonceBuf...)
}

type Result struct {
	NodeID
	Success bool
}
