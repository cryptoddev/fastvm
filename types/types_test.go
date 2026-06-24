package types

import (
	"testing"

	"github.com/holiman/uint256"
)

func TestWordFromBytes32(t *testing.T) {
	b := [32]byte{31: 0x42}
	w := WordFromBytes32(b)
	if w.Uint64() != 0x42 {
		t.Fatalf("want 0x42, got %x", w.Uint64())
	}
}

func TestWordFromBytes32Zero(t *testing.T) {
	var b [32]byte
	w := WordFromBytes32(b)
	if !w.IsZero() {
		t.Fatal("want zero word")
	}
}

func TestWordFromBytes32Max(t *testing.T) {
	b := [32]byte{
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	}
	w := WordFromBytes32(b)
	if w.Uint64() != ^uint64(0) {
		t.Fatalf("low 64 bits should be all 1s, got %x", w.Uint64())
	}
}

func TestWordToBytes32(t *testing.T) {
	w := uint256.NewInt(0x42)
	b := WordToBytes32(w)
	if b[31] != 0x42 {
		t.Fatalf("want b[31]=0x42, got 0x%X", b[31])
	}
	for i := 0; i < 31; i++ {
		if b[i] != 0 {
			t.Fatalf("want b[%d]=0, got 0x%X", i, b[i])
		}
	}
}

func TestWordToBytes32Zero(t *testing.T) {
	w := new(uint256.Int)
	b := WordToBytes32(w)
	if b != ([32]byte{}) {
		t.Fatal("zero word should produce zero bytes32")
	}
}

func TestAddressToWord(t *testing.T) {
	a := Address{19: 0x42}
	w := AddressToWord(a)
	if w.Uint64() != 0x42 {
		t.Fatalf("want 0x42, got %x", w.Uint64())
	}
}

func TestAddressToWordNonZero(t *testing.T) {
	a := Address{0: 0xAB, 19: 0xCD}
	w := AddressToWord(a)
	b := w.Bytes32()
	// Address should be at bytes 12-31
	if b[12] != 0xAB {
		t.Fatalf("want b[12]=0xAB, got 0x%X", b[12])
	}
	if b[31] != 0xCD {
		t.Fatalf("want b[31]=0xCD, got 0x%X", b[31])
	}
}

func TestAddressToWordZero(t *testing.T) {
	var a Address
	w := AddressToWord(a)
	if !w.IsZero() {
		t.Fatal("zero address should produce zero word")
	}
}

func TestWordToAddress(t *testing.T) {
	w := uint256.NewInt(0x42)
	a := WordToAddress(w)
	if a[19] != 0x42 {
		t.Fatalf("want a[19]=0x42, got 0x%X", a[19])
	}
	for i := 0; i < 19; i++ {
		if a[i] != 0 {
			t.Fatalf("want a[%d]=0, got 0x%X", i, a[i])
		}
	}
}

func TestWordToAddressNonZero(t *testing.T) {
	var b [32]byte
	b[12] = 0xAB
	b[31] = 0xCD
	var w uint256.Int
	w.SetBytes32(b[:])
	a := WordToAddress(&w)
	if a[0] != 0xAB {
		t.Fatalf("want a[0]=0xAB, got 0x%X", a[0])
	}
	if a[19] != 0xCD {
		t.Fatalf("want a[19]=0xCD, got 0x%X", a[19])
	}
}

func TestWordToAddressZero(t *testing.T) {
	w := new(uint256.Int)
	a := WordToAddress(w)
	if a != (Address{}) {
		t.Fatal("zero word should produce zero address")
	}
}

func TestHashToWord(t *testing.T) {
	h := Hash{31: 0x42}
	w := HashToWord(h)
	if w.Uint64() != 0x42 {
		t.Fatalf("want 0x42, got %x", w.Uint64())
	}
}

func TestHashToWordZero(t *testing.T) {
	var h Hash
	w := HashToWord(h)
	if !w.IsZero() {
		t.Fatal("zero hash should produce zero word")
	}
}

func TestWordToHash(t *testing.T) {
	w := uint256.NewInt(0x42)
	h := WordToHash(w)
	if h[31] != 0x42 {
		t.Fatalf("want h[31]=0x42, got 0x%X", h[31])
	}
	for i := 0; i < 31; i++ {
		if h[i] != 0 {
			t.Fatalf("want h[%d]=0, got 0x%X", i, h[i])
		}
	}
}

func TestWordToHashZero(t *testing.T) {
	w := new(uint256.Int)
	h := WordToHash(w)
	if h != (Hash{}) {
		t.Fatal("zero word should produce zero hash")
	}
}

func TestZeroAddress(t *testing.T) {
	if ZeroAddress != (Address{}) {
		t.Fatal("ZeroAddress should be all zeros")
	}
}

func TestZeroHash(t *testing.T) {
	if ZeroHash != (Hash{}) {
		t.Fatal("ZeroHash should be all zeros")
	}
}

func TestAddressWordRoundtrip(t *testing.T) {
	a := Address{0: 0x01, 10: 0xAB, 19: 0xFF}
	w := AddressToWord(a)
	a2 := WordToAddress(&w)
	if a != a2 {
		t.Fatalf("roundtrip failed: want %x, got %x", a, a2)
	}
}

func TestHashWordRoundtrip(t *testing.T) {
	h := Hash{0: 0x01, 16: 0xAB, 31: 0xFF}
	w := HashToWord(h)
	h2 := WordToHash(&w)
	if h != h2 {
		t.Fatalf("roundtrip failed: want %x, got %x", h, h2)
	}
}

func TestBytes32WordRoundtrip(t *testing.T) {
	b := [32]byte{0: 0x01, 16: 0xAB, 31: 0xFF}
	w := WordFromBytes32(b)
	b2 := WordToBytes32(&w)
	if b != b2 {
		t.Fatalf("roundtrip failed: want %x, got %x", b, b2)
	}
}
