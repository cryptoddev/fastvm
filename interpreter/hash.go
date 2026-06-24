package interpreter

import (
	"hash"

	"golang.org/x/crypto/sha3"
)

func init() {
	newKeccak = func() interface {
		Write([]byte) (int, error)
		Sum([]byte) []byte
	} {
		return sha3.NewLegacyKeccak256().(hash.Hash)
	}
}
