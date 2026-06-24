package forkdb

import (
	"hash"

	"golang.org/x/crypto/sha3"
)

func newKeccak256() hash.Hash {
	return sha3.NewLegacyKeccak256()
}
