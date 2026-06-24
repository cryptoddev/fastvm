package precompile

import (
	"math/big"
	"testing"

	"golang.org/x/crypto/bn256"
)

func TestNewG1Point(t *testing.T) {
	g1 := new(bn256.G1).ScalarBaseMult(big.NewInt(1))
	data := g1.Marshal()
	p, err := newG1Point(data)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("expected non-nil point")
	}

	_, err = newG1Point([]byte{0, 1, 2})
	if err == nil {
		t.Fatal("expected error for invalid G1 point")
	}
}

func TestNewG2Point(t *testing.T) {
	g2 := new(bn256.G2).ScalarBaseMult(big.NewInt(1))
	data := g2.Marshal()
	p, err := newG2Point(data)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("expected non-nil point")
	}

	_, err = newG2Point([]byte{0, 1, 2})
	if err == nil {
		t.Fatal("expected error for invalid G2 point")
	}
}

func TestModexpGasEIP2565ELenZero(t *testing.T) {
	gas := modexpGasEIP2565(1, 0, 1, nil)
	if gas < 200 {
		t.Fatalf("gas should be at least 200, got %d", gas)
	}
}

func TestModexpGasEIP2565ExpZero(t *testing.T) {
	input := make([]byte, 98)
	input[31] = 1
	input[63] = 1
	input[95] = 1
	gas := modexpGasEIP2565(1, 1, 1, input)
	if gas < 200 {
		t.Fatalf("gas should be at least 200, got %d", gas)
	}
}

func TestModexpGasEIP2565PartialExpELenLeq32(t *testing.T) {
	input := make([]byte, 100)
	input[31] = 1
	input[63] = 20
	input[95] = 1
	copy(input[96:], []byte{2, 0, 0, 1})
	gas := modexpGasEIP2565(1, 20, 1, input)
	if gas < 200 {
		t.Fatalf("gas should be at least 200, got %d", gas)
	}
}

func TestModexpGasEIP2565PartialExpELenGt32(t *testing.T) {
	input := make([]byte, 110)
	input[31] = 1
	input[63] = 64
	input[95] = 1
	copy(input[96:], []byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})
	gas := modexpGasEIP2565(1, 64, 1, input)
	if gas < 200 {
		t.Fatalf("gas should be at least 200, got %d", gas)
	}
}

func TestBn256PairingMultiPair(t *testing.T) {
	g1 := new(bn256.G1).ScalarBaseMult(big.NewInt(1))
	g2 := new(bn256.G2).ScalarBaseMult(big.NewInt(1))
	g1Data := g1.Marshal()
	g2Data := g2.Marshal()
	var input []byte
	input = append(input, g1Data...)
	input = append(input, g2Data...)
	input = append(input, g1Data...)
	input = append(input, g2Data...)
	out, _, isPC := Run(addr(8), input, 200000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(out))
	}
}

func TestBn256PairingValidSingle(t *testing.T) {
	g1 := new(bn256.G1).ScalarBaseMult(big.NewInt(1))
	g2 := new(bn256.G2).ScalarBaseMult(big.NewInt(1))
	input := append(g1.Marshal(), g2.Marshal()...)
	out, _, isPC := Run(addr(8), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(out))
	}
}
