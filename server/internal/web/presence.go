package web

import (
	"crypto/sha256"
	"encoding/binary"
)

func tokenConnectionID(token string) uint64 {
	sum := sha256.Sum256([]byte(token))
	id := binary.BigEndian.Uint64(sum[:8])
	if id == 0 {
		return 1
	}
	return id
}
