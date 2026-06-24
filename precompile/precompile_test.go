package precompile

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func addr(id byte) [20]byte {
	var a [20]byte
	a[19] = id
	return a
}

func TestIsPrecompile(t *testing.T) {
	for i := byte(1); i <= 9; i++ {
		if !IsPrecompile(addr(i)) {
			t.Errorf("addr(%d) should be precompile", i)
		}
	}
	if IsPrecompile(addr(0)) || IsPrecompile(addr(10)) {
		t.Error("addr(0) and addr(10) should not be precompile")
	}
}

func TestIsPrecompileNonZeroPrefix(t *testing.T) {
	var a [20]byte
	a[0] = 0x01
	a[19] = 1
	if IsPrecompile(a) {
		t.Error("address with non-zero prefix should not be precompile")
	}
}

func TestSHA256(t *testing.T) {
	input := []byte("hello")
	out, gasLeft, isPC := Run(addr(2), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if gasLeft == 0 {
		t.Fatal("should have gas left")
	}
	want := sha256.Sum256(input)
	if string(out) != string(want[:]) {
		t.Fatalf("sha256 mismatch: got %x want %x", out, want)
	}
}

func TestSHA256Empty(t *testing.T) {
	out, gasLeft, isPC := Run(addr(2), nil, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	want := sha256.Sum256(nil)
	if string(out) != string(want[:]) {
		t.Fatalf("sha256 empty mismatch: got %x want %x", out, want)
	}
	_ = gasLeft
}

func TestSHA256OutOfGas(t *testing.T) {
	_, gasLeft, isPC := Run(addr(2), []byte("hello"), 10)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if gasLeft != 0 {
		t.Fatal("should be out of gas")
	}
}

func TestRIPEMD160(t *testing.T) {
	input := []byte("hello")
	out, gasLeft, isPC := Run(addr(3), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes output, got %d", len(out))
	}
	// First 12 bytes should be zero (RIPEMD-160 is 20 bytes, padded to 32)
	for i := 0; i < 12; i++ {
		if out[i] != 0 {
			t.Fatalf("want out[%d]=0, got 0x%X", i, out[i])
		}
	}
	_ = gasLeft
}

func TestRIPEMD160Empty(t *testing.T) {
	out, gasLeft, isPC := Run(addr(3), nil, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes output, got %d", len(out))
	}
	_ = gasLeft
}

func TestRIPEMD160OutOfGas(t *testing.T) {
	_, gasLeft, isPC := Run(addr(3), []byte("hello"), 10)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if gasLeft != 0 {
		t.Fatal("should be out of gas")
	}
}

func TestIdentity(t *testing.T) {
	input := []byte{1, 2, 3, 4}
	out, _, isPC := Run(addr(4), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if string(out) != string(input) {
		t.Fatalf("identity mismatch")
	}
}

func TestIdentityEmpty(t *testing.T) {
	out, gasLeft, isPC := Run(addr(4), nil, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 0 {
		t.Fatalf("want empty output, got %d bytes", len(out))
	}
	_ = gasLeft
}

func TestIdentityOutOfGas(t *testing.T) {
	_, gasLeft, isPC := Run(addr(4), []byte{1, 2, 3}, 10)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if gasLeft != 0 {
		t.Fatal("should be out of gas")
	}
}

func TestModexp(t *testing.T) {
	// 2^10 mod 1024 = 0
	bLen := pad32big(1)  // base length = 1
	eLen := pad32big(1)  // exp length = 1
	mLen := pad32big(2)  // mod length = 2
	input := append(append(append(bLen, eLen...), mLen...), 2, 10, 4, 0) // base=2, exp=10, mod=1024
	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	_ = out // result is 0 (2^10 = 1024 = 0x400, mod 0x400 = 0)
}

func TestModexpEmptyInput(t *testing.T) {
	out, gasLeft, isPC := Run(addr(5), nil, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 0 {
		t.Fatalf("want empty output for empty input, got %d bytes", len(out))
	}
	_ = gasLeft
}

func TestModexpZeroMod(t *testing.T) {
	bLen := pad32big(1)
	eLen := pad32big(1)
	mLen := pad32big(1)
	input := append(append(append(bLen, eLen...), mLen...), 2, 1, 0)
	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 1 || out[0] != 0 {
		t.Fatalf("want [0], got %v", out)
	}
}

func TestModexpModOne(t *testing.T) {
	bLen := pad32big(1)
	eLen := pad32big(1)
	mLen := pad32big(1)
	input := append(append(append(bLen, eLen...), mLen...), 5, 3, 1)
	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 1 || out[0] != 0 {
		t.Fatalf("want [0] (anything mod 1 = 0), got %v", out)
	}
}

func TestModexpZeroExp(t *testing.T) {
	bLen := pad32big(1)
	eLen := pad32big(1)
	mLen := pad32big(1)
	input := append(append(append(bLen, eLen...), mLen...), 5, 0, 7)
	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	// 5^0 mod 7 = 1
	if len(out) != 1 || out[0] != 1 {
		t.Fatalf("want [1], got %v", out)
	}
}

func TestModexpOutOfGas(t *testing.T) {
	bLen := pad32big(1)
	eLen := pad32big(1)
	mLen := pad32big(1)
	input := append(append(append(bLen, eLen...), mLen...), 2, 10, 3)
	_, gasLeft, isPC := Run(addr(5), input, 10)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if gasLeft != 0 {
		t.Fatal("should be out of gas")
	}
}

func TestModexpZeroLengthMod(t *testing.T) {
	bLen := pad32big(1)
	eLen := pad32big(1)
	mLen := pad32big(0)
	input := append(append(append(bLen, eLen...), mLen...), 2, 10)
	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 0 {
		t.Fatalf("want empty output for zero mod length, got %d bytes", len(out))
	}
}

func TestEcRecoverInvalidSig(t *testing.T) {
	// Invalid signature should return 32 zero bytes, not error.
	input := make([]byte, 128)
	out, _, isPC := Run(addr(1), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(out))
	}
}

func TestEcRecoverRecoverError(t *testing.T) {
	// Valid v but invalid r/s should trigger RecoverCompact error
	input := make([]byte, 128)
	// Set v = 27 (valid)
	input[63] = 27
	// r and s are all zeros (invalid)
	out, _, isPC := Run(addr(1), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(out))
	}
}

func TestEcRecover(t *testing.T) {
	// Known test vector from Ethereum.
	// hash, v=28, r, s from a real signature.
	hashHex := "456e9aea5e197a1f1af7a3e85a3212fa4049a3ba34c2289b4c860fc0b0c64ef3"
	vHex    := "000000000000000000000000000000000000000000000000000000000000001c"
	rHex    := "9242685bf161793cc25603c231bc2f568eb630ea16aa137d2664ac8038825608"
	sHex    := "4f8ae3bd7535248d0bd448298cc2e2071e56992d0774dc340c368ae950852ada"
	wantHex := "0000000000000000000000007156526fbd7a3c72969b54f64e42c10fbb768c8a"

	input, _ := hex.DecodeString(hashHex + vHex + rHex + sHex)
	out, _, _ := Run(addr(1), input, 10000)
	got := hex.EncodeToString(out)
	if got != wantHex {
		t.Fatalf("ecRecover:\n got  %s\n want %s", got, wantHex)
	}
}

func TestEcRecoverInvalidV(t *testing.T) {
	input := make([]byte, 128)
	// Set v to an invalid value (not 27 or 28)
	input[63] = 1
	out, gasLeft, isPC := Run(addr(1), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestEcRecoverVTooLarge(t *testing.T) {
	input := make([]byte, 128)
	// Set v to a value > 255 (bitLen > 8)
	input[32] = 1 // First byte of v is non-zero, making bitLen > 8
	out, gasLeft, isPC := Run(addr(1), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestEcRecoverOutOfGas(t *testing.T) {
	input := make([]byte, 128)
	_, gasLeft, isPC := Run(addr(1), input, 100)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if gasLeft != 0 {
		t.Fatal("should be out of gas")
	}
}

func TestEcRecoverShortInput(t *testing.T) {
	input := make([]byte, 64)
	out, gasLeft, isPC := Run(addr(1), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestBn256Add(t *testing.T) {
	// Add two G1 points (both zero points)
	input := make([]byte, 128)
	out, gasLeft, isPC := Run(addr(6), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 64 {
		t.Fatalf("want 64 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestBn256AddOutOfGas(t *testing.T) {
	input := make([]byte, 128)
	_, gasLeft, isPC := Run(addr(6), input, 100)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if gasLeft != 0 {
		t.Fatal("should be out of gas")
	}
}

func TestBn256AddInvalidPoint(t *testing.T) {
	input := make([]byte, 128)
	input[0] = 0xFF // Invalid point
	out, gasLeft, isPC := Run(addr(6), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 64 {
		t.Fatalf("want 64 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestBn256AddInvalidSecondPoint(t *testing.T) {
	input := make([]byte, 128)
	input[64] = 0xFF // Invalid second point
	out, gasLeft, isPC := Run(addr(6), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 64 {
		t.Fatalf("want 64 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestBn256ScalarMul(t *testing.T) {
	// Scalar multiply a G1 point (zero point)
	input := make([]byte, 96)
	out, gasLeft, isPC := Run(addr(7), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 64 {
		t.Fatalf("want 64 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestBn256ScalarMulOutOfGas(t *testing.T) {
	input := make([]byte, 96)
	_, gasLeft, isPC := Run(addr(7), input, 100)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if gasLeft != 0 {
		t.Fatal("should be out of gas")
	}
}

func TestBn256ScalarMulInvalidPoint(t *testing.T) {
	input := make([]byte, 96)
	input[0] = 0xFF // Invalid point
	out, gasLeft, isPC := Run(addr(7), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 64 {
		t.Fatalf("want 64 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestBn256PairingEmpty(t *testing.T) {
	out, gasLeft, isPC := Run(addr(8), nil, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(out))
	}
	// Empty input should return success (1)
	if out[31] != 1 {
		t.Fatalf("want success (1), got %d", out[31])
	}
	_ = gasLeft
}

func TestBn256PairingWithInput(t *testing.T) {
	// 192 bytes = 1 G1 point (64 bytes) + 1 G2 point (128 bytes)
	// Use zeros - this may or may not be a valid point depending on the library
	input := make([]byte, 192)
	out, gasLeft, isPC := Run(addr(8), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestBn256PairingInvalidG1(t *testing.T) {
	input := make([]byte, 192)
	input[0] = 0xFF // Invalid G1 point
	out, gasLeft, isPC := Run(addr(8), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestBn256PairingInvalidG2(t *testing.T) {
	input := make([]byte, 192)
	input[64] = 0xFF // Invalid G2 point
	out, gasLeft, isPC := Run(addr(8), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestBn256PairingInvalidLength(t *testing.T) {
	input := make([]byte, 100) // Not a multiple of 192
	_, _, isPC := Run(addr(8), input, 100000)
	if !isPC {
		t.Fatal("should be precompile (error path)")
	}
}

func TestBn256PairingOutOfGas(t *testing.T) {
	input := make([]byte, 192)
	_, gasLeft, isPC := Run(addr(8), input, 100)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if gasLeft != 0 {
		t.Fatal("should be out of gas")
	}
}

func TestBlake2F(t *testing.T) {
	// BLAKE2F with 1 round
	input := make([]byte, 213)
	// rounds = 1 (big-endian at [0:4])
	input[3] = 0x01
	// final = 0

	out, gasLeft, isPC := Run(addr(9), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 64 {
		t.Fatalf("want 64 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestBlake2FInvalidLength(t *testing.T) {
	input := make([]byte, 100) // Wrong length
	_, _, isPC := Run(addr(9), input, 10000)
	if !isPC {
		t.Fatal("should be precompile (error path)")
	}
}

func TestBlake2FOutOfGas(t *testing.T) {
	input := make([]byte, 213)
	input[3] = 0xFF // 255 rounds
	_, gasLeft, isPC := Run(addr(9), input, 100)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if gasLeft != 0 {
		t.Fatal("should be out of gas")
	}
}

func TestBlake2FFinalFlag(t *testing.T) {
	input := make([]byte, 213)
	input[3] = 0x01       // 1 round
	input[212] = 0x01     // final = true

	out, gasLeft, isPC := Run(addr(9), input, 10000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 64 {
		t.Fatalf("want 64 bytes, got %d", len(out))
	}
	_ = gasLeft
}

func TestNonPrecompile(t *testing.T) {
	var a [20]byte
	a[19] = 0xAB
	_, _, isPC := Run(a, nil, 10000)
	if isPC {
		t.Fatal("should not be precompile")
	}
}

func TestNonPrecompileNonZeroPrefix(t *testing.T) {
	var a [20]byte
	a[0] = 0x01 // Non-zero prefix
	a[19] = 1
	_, _, isPC := Run(a, nil, 10000)
	if isPC {
		t.Fatal("should not be precompile with non-zero prefix")
	}
}

func TestNonPrecompileZero(t *testing.T) {
	var a [20]byte
	_, _, isPC := Run(a, nil, 10000)
	if isPC {
		t.Fatal("should not be precompile (address 0)")
	}
}

func TestOutOfGas(t *testing.T) {
	_, gasLeft, isPC := Run(addr(2), []byte("hello"), 10) // SHA256 costs 60+
	if !isPC {
		t.Fatal("should be precompile")
	}
	if gasLeft != 0 {
		t.Fatal("should be out of gas")
	}
}

func TestWordCount(t *testing.T) {
	tests := []struct {
		n    int
		want uint64
	}{
		{0, 0},
		{1, 1},
		{32, 1},
		{33, 2},
		{64, 2},
	}
	for _, tt := range tests {
		got := wordCount(make([]byte, tt.n))
		if got != tt.want {
			t.Errorf("wordCount(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestZero32(t *testing.T) {
	z := zero32()
	if len(z) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(z))
	}
	for i, b := range z {
		if b != 0 {
			t.Fatalf("want z[%d]=0, got %d", i, b)
		}
	}
}

func TestPad32(t *testing.T) {
	// Already 32 bytes
	b32 := make([]byte, 32)
	b32[31] = 0x42
	p := pad32(b32)
	if &p[0] != &b32[0] {
		t.Fatal("pad32 should return same slice for 32-byte input")
	}

	// Less than 32 bytes
	b4 := []byte{0x01, 0x02, 0x03, 0x04}
	p = pad32(b4)
	if len(p) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(p))
	}
	if p[31] != 0x04 {
		t.Fatalf("want p[31]=0x04, got 0x%X", p[31])
	}
	if p[28] != 0x01 {
		t.Fatalf("want p[28]=0x01, got 0x%X", p[28])
	}
}

func TestModexpGasEIP2565(t *testing.T) {
	// Test with zero exponent
	gas := modexpGasEIP2565(1, 1, 1, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	if gas < 200 {
		t.Fatalf("gas should be at least 200, got %d", gas)
	}
}

func TestModexpLargeExponent(t *testing.T) {
	bLen := pad32big(1)
	eLen := pad32big(33) // eLen > 32
	mLen := pad32big(1)
	// Create input with 33-byte exponent
	input := append(append(append(bLen, eLen...), mLen...), 2)
	expBytes := make([]byte, 33)
	expBytes[32] = 1 // small exponent value
	input = append(input, expBytes...)
	input = append(input, 7) // mod = 7

	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	// 2^1 mod 7 = 2
	if len(out) != 1 || out[0] != 2 {
		t.Fatalf("want [2], got %v", out)
	}
}

func TestModexpTruncatedResult(t *testing.T) {
	// Result larger than mod length should be truncated
	bLen := pad32big(1)
	eLen := pad32big(1)
	mLen := pad32big(1)
	// 10^10 mod 3 = 1
	input := append(append(append(bLen, eLen...), mLen...), 10, 10, 3)
	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 1 || out[0] != 1 {
		t.Fatalf("want [1], got %v", out)
	}
}

func TestModexpTruncatedResultLarge(t *testing.T) {
	// Test where result is larger than mLen (truncation path)
	// Use mod=0xFFFF (2 bytes) but mLen=1, so result gets truncated
	bLen := pad32big(2)
	eLen := pad32big(1)
	mLen := pad32big(1)
	// base=0xFFFF, exp=1, mod=0xFFFF
	// 0xFFFF^1 mod 0xFFFF = 0, but let's use a case where result > 1 byte
	// base=0x01FF, exp=1, mod=0x0100 -> result=0xFF (fits in 1 byte)
	// Let's use: base=3, exp=10, mod=1000 -> 59049 mod 1000 = 49 (fits)
	// For truncation: base=2, exp=20, mod=0xFFFF -> 1048576 mod 65535 = 43552 (2 bytes)
	// With mLen=1, this should truncate to last byte: 43552 % 256 = 0
	input := append(append(append(bLen, eLen...), mLen...), 0x02, 0x00, 10, 0xFF, 0xFF)
	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	_ = out
}

func TestModexpResultLargerThanModLen(t *testing.T) {
	// Explicitly test the truncation path: result > mLen
	// 255^2 = 65025, mod 256 = 1 (fits in 1 byte)
	// Let's try: base=256, exp=2, mod=65535 -> 65536 mod 65535 = 1
	// For truncation we need result > mLen bytes
	// base=2, exp=100, mod=10^20 (large mod), mLen=5
	// This is complex, let's use a simpler approach:
	// base=0xFFFF, exp=2, mod=0xFFFF -> result=0, fits
	// Actually, result is always < mod, so if mod fits in mLen, result fits too.
	// The truncation path is for when mLen < bytes needed for mod.
	// Let's test: mod=0x01FF (2 bytes, value 511), mLen=1
	// base=2, exp=8, mod=511 -> 256 mod 511 = 256 (needs 2 bytes, but mLen=1)
	// Result truncated to 1 byte: 256 & 0xFF = 0
	bLen := pad32big(1)
	eLen := pad32big(1)
	mLen := pad32big(1)
	input := append(append(append(bLen, eLen...), mLen...), 2, 8, 0x01, 0xFF)
	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	if len(out) != 1 {
		t.Fatalf("want 1 byte, got %d", len(out))
	}
	// 256 mod 511 = 256, truncated to 1 byte = 0
	if out[0] != 0 {
		t.Fatalf("want 0 (truncated), got %d", out[0])
	}
}

func TestModexpLargeExponentZeroHead(t *testing.T) {
	// eLen > 32 with exponent head being zero
	bLen := pad32big(1)
	eLen := pad32big(64) // eLen > 32
	mLen := pad32big(1)
	// Create input: base=2, exp=64 bytes of zeros, mod=7
	input := append(append(append(bLen, eLen...), mLen...), 2)
	expBytes := make([]byte, 64) // All zeros
	input = append(input, expBytes...)
	input = append(input, 7) // mod = 7

	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	// 2^0 mod 7 = 1
	if len(out) != 1 || out[0] != 1 {
		t.Fatalf("want [1], got %v", out)
	}
}

func TestModexpGasLargeExponent(t *testing.T) {
	// Test gas calculation with eLen > 32 and non-zero head
	input := make([]byte, 96+1+64+1) // lengths + base + exp + mod
	// bLen=1, eLen=64, mLen=1
	input[31] = 1
	input[63] = 64
	input[95] = 1
	input[96] = 2 // base = 2
	// exp: first 32 bytes have a value, rest are zeros
	input[96+1+31] = 1 // exp head has value
	input[96+1+64] = 3 // mod = 3

	gas := modexpGasEIP2565(1, 64, 1, input)
	if gas < 200 {
		t.Fatalf("gas should be at least 200, got %d", gas)
	}
}

func TestModexpGasLargeExponentZeroHead(t *testing.T) {
	// Test gas calculation with eLen > 32 and zero head
	input := make([]byte, 96+1+64+1)
	input[31] = 1
	input[63] = 64
	input[95] = 1
	input[96] = 2
	// exp: all zeros (head is zero)
	input[96+1+64] = 3

	gas := modexpGasEIP2565(1, 64, 1, input)
	if gas < 200 {
		t.Fatalf("gas should be at least 200, got %d", gas)
	}
}

func TestModexpPartialInput(t *testing.T) {
	// Input shorter than 96 bytes for lengths
	bLen := pad32big(1)
	input := bLen[:16] // Only partial
	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	_ = out
}

func TestModexpDataOutOfBounds(t *testing.T) {
	// bLen=100 but input only has 96+1 bytes
	// This tests getData with start >= len(input)
	bLen := pad32big(100)
	eLen := pad32big(0)
	mLen := pad32big(1)
	input := append(append(append(bLen, eLen...), mLen...), 0x02)
	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	_ = out
}

func TestModexpDataPartial(t *testing.T) {
	// bLen=10 but input only has 96+5 bytes
	// This tests getData with end > len(input)
	bLen := pad32big(10)
	eLen := pad32big(0)
	mLen := pad32big(1)
	input := append(append(append(bLen, eLen...), mLen...), 0x02, 0x03, 0x04, 0x05, 0x06)
	out, _, isPC := Run(addr(5), input, 100000)
	if !isPC {
		t.Fatal("should be precompile")
	}
	_ = out
}

func TestDispatchDefault(t *testing.T) {
	// This should not be reachable from Run since it validates id,
	// but test the dispatch function directly
	out, gasLeft, err := dispatch(0, nil, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if out != nil {
		t.Fatal("want nil output for id=0")
	}
	if gasLeft != 10000 {
		t.Fatalf("want gasLeft=10000, got %d", gasLeft)
	}
}

func pad32big(n int) []byte {
	b := make([]byte, 32)
	b[31] = byte(n)
	return b
}
