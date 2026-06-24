package types

import (
	"github.com/holiman/uint256"
)

type Word = uint256.Int

type Gas = uint64

type OpCode = byte

type StorageKey = [32]byte

type StorageValue = [32]byte

type Address = [20]byte

type Hash = [32]byte

var ZeroAddress Address

var ZeroHash Hash

func WordFromBytes32(b [32]byte) Word {
	var w Word
	w.SetBytes32(b[:])
	return w
}

func WordToBytes32(w *Word) [32]byte {
	return w.Bytes32()
}

func AddressToWord(a Address) Word {
	var w Word
	var b [32]byte
	copy(b[12:], a[:])
	w.SetBytes32(b[:])
	return w
}

func WordToAddress(w *Word) Address {
	b := w.Bytes32()
	var a Address
	copy(a[:], b[12:])
	return a
}

func HashToWord(h Hash) Word {
	var w Word
	w.SetBytes32(h[:])
	return w
}

func WordToHash(w *Word) Hash {
	return w.Bytes32()
}
