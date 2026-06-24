package precompile

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	dcrecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/bn256"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
)

var ErrOutOfGas = errors.New("precompile: out of gas")

func Run(addr [20]byte, input []byte, gas uint64) ([]byte, uint64, bool) {
	if addr[0] != 0 || addr[1] != 0 || addr[2] != 0 || addr[3] != 0 ||
		addr[4] != 0 || addr[5] != 0 || addr[6] != 0 || addr[7] != 0 ||
		addr[8] != 0 || addr[9] != 0 || addr[10] != 0 || addr[11] != 0 ||
		addr[12] != 0 || addr[13] != 0 || addr[14] != 0 || addr[15] != 0 ||
		addr[16] != 0 || addr[17] != 0 || addr[18] != 0 {
		return nil, gas, false
	}
	id := addr[19]
	if id == 0 || id > 9 {
		return nil, gas, false
	}

	out, gasLeft, err := dispatch(id, input, gas)
	if err != nil {
		return nil, 0, true
	}
	return out, gasLeft, true
}

func dispatch(id byte, input []byte, gas uint64) ([]byte, uint64, error) {
	switch id {
	case 1:
		return ecRecover(input, gas)
	case 2:
		return sha256hash(input, gas)
	case 3:
		return ripemd160hash(input, gas)
	case 4:
		return identity(input, gas)
	case 5:
		return modexp(input, gas)
	case 6:
		return bn256Add(input, gas)
	case 7:
		return bn256ScalarMul(input, gas)
	case 8:
		return bn256Pairing(input, gas)
	case 9:
		return blake2F(input, gas)
	}
	return nil, gas, nil
}

func ecRecover(input []byte, gas uint64) ([]byte, uint64, error) {
	const cost = 3000
	if gas < cost {
		return nil, 0, ErrOutOfGas
	}
	gas -= cost

	in := make([]byte, 128)
	copy(in, input)

	hash := in[:32]
	v := new(big.Int).SetBytes(in[32:64])
	r := new(big.Int).SetBytes(in[64:96])
	s := new(big.Int).SetBytes(in[96:128])

	// v must be 27 or 28 per EIP-2.
	if v.BitLen() > 8 {
		return zero32(), gas, nil
	}
	vByte := byte(v.Uint64())
	if vByte != 27 && vByte != 28 {
		return zero32(), gas, nil
	}

	sig := make([]byte, 65)
	sig[0] = vByte
	copy(sig[1:33], pad32(r.Bytes()))
	copy(sig[33:65], pad32(s.Bytes()))

	pubKey, _, err := dcrecdsa.RecoverCompact(sig, hash)
	if err != nil {
		return zero32(), gas, nil
	}

	// Address = keccak256(pubkey uncompressed sans 0x04 prefix)[12:]
	uncompressed := pubKey.SerializeUncompressed()
	h := keccak256(uncompressed[1:])
	out := make([]byte, 32)
	copy(out[12:], h[12:])
	return out, gas, nil
}

func sha256hash(input []byte, gas uint64) ([]byte, uint64, error) {
	cost := uint64(60) + 12*wordCount(input)
	if gas < cost {
		return nil, 0, ErrOutOfGas
	}
	gas -= cost
	h := sha256.Sum256(input)
	return h[:], gas, nil
}

func ripemd160hash(input []byte, gas uint64) ([]byte, uint64, error) {
	cost := uint64(600) + 120*wordCount(input)
	if gas < cost {
		return nil, 0, ErrOutOfGas
	}
	gas -= cost
	h := ripemd160.New()
	h.Write(input)
	out := make([]byte, 32)
	copy(out[12:], h.Sum(nil))
	return out, gas, nil
}

func identity(input []byte, gas uint64) ([]byte, uint64, error) {
	cost := uint64(15) + 3*wordCount(input)
	if gas < cost {
		return nil, 0, ErrOutOfGas
	}
	gas -= cost
	out := make([]byte, len(input))
	copy(out, input)
	return out, gas, nil
}

func modexp(input []byte, gas uint64) ([]byte, uint64, error) {
	// Read baseLen, expLen, modLen from first 96 bytes (EIP-198).
	readLen := func(off int) uint64 {
		if off+32 > len(input) {
			tmp := make([]byte, 32)
			if off < len(input) {
				copy(tmp[32-(len(input)-off):], input[off:])
			}
			return new(big.Int).SetBytes(tmp).Uint64()
		}
		return new(big.Int).SetBytes(input[off : off+32]).Uint64()
	}

	bLen := readLen(0)
	eLen := readLen(32)
	mLen := readLen(64)

	cost := modexpGasEIP2565(bLen, eLen, mLen, input)
	if gas < cost {
		return nil, 0, ErrOutOfGas
	}
	gas -= cost

	if mLen == 0 {
		return []byte{}, gas, nil
	}

	getData := func(off, length uint64) []byte {
		if length == 0 {
			return nil
		}
		start := uint64(96) + off
		if start >= uint64(len(input)) {
			return make([]byte, length)
		}
		end := start + length
		if end > uint64(len(input)) {
			tmp := make([]byte, length)
			copy(tmp, input[start:])
			return tmp
		}
		return input[start:end]
	}

	base := new(big.Int).SetBytes(getData(0, bLen))
	exp := new(big.Int).SetBytes(getData(bLen, eLen))
	mod := new(big.Int).SetBytes(getData(bLen+eLen, mLen))

	var result *big.Int
	if mod.Sign() == 0 {
		result = new(big.Int)
	} else if mod.Cmp(big.NewInt(1)) == 0 {
		result = new(big.Int)
	} else if exp.Sign() == 0 {
		result = big.NewInt(1)
		result.Mod(result, mod)
	} else {
		result = new(big.Int).Exp(base, exp, mod)
	}

	out := make([]byte, mLen)
	rb := result.Bytes()
	if uint64(len(rb)) <= mLen {
		copy(out[mLen-uint64(len(rb)):], rb)
	} else {
		copy(out, rb[uint64(len(rb))-mLen:])
	}
	return out, gas, nil
}

func modexpGasEIP2565(bLen, eLen, mLen uint64, input []byte) uint64 {
	maxLen := bLen
	if mLen > maxLen {
		maxLen = mLen
	}
	words := (maxLen + 7) / 8
	multComplexity := words * words

	var iterCount uint64
	if eLen <= 32 {
		if eLen == 0 {
			iterCount = 0
		} else {
			start := 96 + bLen
			expBytes := make([]byte, eLen)
			if int(start) < len(input) {
				end := start + eLen
				if int(end) > len(input) {
					end = uint64(len(input))
				}
				copy(expBytes[eLen-(end-start):], input[start:end])
			}
			e := new(big.Int).SetBytes(expBytes)
			if e.Sign() == 0 {
				iterCount = 0
			} else {
				iterCount = uint64(e.BitLen() - 1)
			}
		}
	} else {
		// eLen > 32: use only the first 32 bytes of exponent for gas estimation
		start := 96 + bLen
		expHead := make([]byte, 32)
		if int(start) < len(input) {
			end := start + 32
			if int(end) > len(input) {
				end = uint64(len(input))
			}
			copy(expHead[32-(end-start):], input[start:end])
		}
		e := new(big.Int).SetBytes(expHead)
		if e.Sign() == 0 {
			iterCount = 8 * (eLen - 32)
		} else {
			iterCount = uint64(e.BitLen()-1) + 8*(eLen-32)
		}
	}
	if iterCount < 1 {
		iterCount = 1
	}

	gas := multComplexity * iterCount / 3
	if gas < 200 {
		gas = 200
	}
	return gas
}

func bn256Add(input []byte, gas uint64) ([]byte, uint64, error) {
	const cost = 150
	if gas < cost {
		return nil, 0, ErrOutOfGas
	}
	gas -= cost

	in := make([]byte, 128)
	copy(in, input)

	p1, ok := new(bn256.G1).Unmarshal(in[:64])
	if !ok {
		return make([]byte, 64), gas, nil
	}
	p2, ok := new(bn256.G1).Unmarshal(in[64:])
	if !ok {
		return make([]byte, 64), gas, nil
	}
	return new(bn256.G1).Add(p1, p2).Marshal(), gas, nil
}

func bn256ScalarMul(input []byte, gas uint64) ([]byte, uint64, error) {
	const cost = 6000
	if gas < cost {
		return nil, 0, ErrOutOfGas
	}
	gas -= cost

	in := make([]byte, 96)
	copy(in, input)

	p, ok := new(bn256.G1).Unmarshal(in[:64])
	if !ok {
		return make([]byte, 64), gas, nil
	}
	scalar := new(big.Int).SetBytes(in[64:96])
	return new(bn256.G1).ScalarMult(p, scalar).Marshal(), gas, nil
}

func bn256Pairing(input []byte, gas uint64) ([]byte, uint64, error) {
	if len(input)%192 != 0 {
		return nil, gas, errors.New("bn256Pairing: invalid input length")
	}
	k := uint64(len(input) / 192)
	cost := 45000 + 34000*k
	if gas < cost {
		return nil, 0, ErrOutOfGas
	}
	gas -= cost

	out := make([]byte, 32)
	if len(input) == 0 {
		out[31] = 1
		return out, gas, nil
	}

	// acc = product of all pairings; result is 1 iff acc == GT identity
	var acc *bn256.GT
	for i := 0; i < len(input); i += 192 {
		g1, ok := new(bn256.G1).Unmarshal(input[i : i+64])
		if !ok {
			return out, gas, nil
		}
		g2, ok := new(bn256.G2).Unmarshal(input[i+64 : i+192])
		if !ok {
			return out, gas, nil
		}
		p := bn256.Pair(g1, g2)
		if acc == nil {
			acc = p
		}
	}
	if acc != nil {
		m := acc.Marshal()
		isOne := true
		for _, b := range m[1:] {
			if b != 0 {
				isOne = false
				break
			}
		}
		if isOne && m[0] == 1 {
			out[31] = 1
		}
	}
	return out, gas, nil
}

func newG1Point(data []byte) (*bn256.G1, error) {
	p, ok := new(bn256.G1).Unmarshal(data)
	if !ok {
		return nil, errors.New("invalid G1 point")
	}
	return p, nil
}

func newG2Point(data []byte) (*bn256.G2, error) {
	p, ok := new(bn256.G2).Unmarshal(data)
	if !ok {
		return nil, errors.New("invalid G2 point")
	}
	return p, nil
}

func blake2F(input []byte, gas uint64) ([]byte, uint64, error) {
	if len(input) != 213 {
		return nil, gas, errors.New("blake2f: invalid input length")
	}
	rounds := binary.BigEndian.Uint32(input[:4])
	cost := uint64(rounds)
	if gas < cost {
		return nil, 0, ErrOutOfGas
	}
	gas -= cost

	// Parse state h (8 x uint64 LE), message m (16 x uint64 LE), and counters t[2] from input
	var h [8]uint64
	for i := 0; i < 8; i++ {
		h[i] = binary.LittleEndian.Uint64(input[4+i*8 : 4+i*8+8])
	}
	var m [16]uint64
	for i := 0; i < 16; i++ {
		m[i] = binary.LittleEndian.Uint64(input[68+i*8 : 68+i*8+8])
	}
	t0 := binary.LittleEndian.Uint64(input[196:204])
	t1 := binary.LittleEndian.Uint64(input[204:212])
	final := input[212] != 0

	blake2bF(&h, m, [2]uint64{t0, t1}, final, rounds)

	out := make([]byte, 64)
	for i := 0; i < 8; i++ {
		binary.LittleEndian.PutUint64(out[i*8:], h[i])
	}
	return out, gas, nil
}

func blake2bF(h *[8]uint64, m [16]uint64, t [2]uint64, final bool, rounds uint32) {
	iv := [8]uint64{
		0x6a09e667f3bcc908, 0xbb67ae8584caa73b,
		0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1,
		0x510e527fade682d1, 0x9b05688c2b3e6c1f,
		0x1f83d9abfb41bd6b, 0x5be0cd19137e2179,
	}
	sigma := [12][16]byte{
		{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		{14, 10, 4, 8, 9, 15, 13, 6, 1, 12, 0, 2, 11, 7, 5, 3},
		{11, 8, 12, 0, 5, 2, 15, 13, 10, 14, 3, 6, 7, 1, 9, 4},
		{7, 9, 3, 1, 13, 12, 11, 14, 2, 6, 5, 10, 4, 0, 15, 8},
		{9, 0, 5, 7, 2, 4, 10, 15, 14, 1, 11, 12, 6, 8, 3, 13},
		{2, 12, 6, 10, 0, 11, 8, 3, 4, 13, 7, 5, 15, 14, 1, 9},
		{12, 5, 1, 15, 14, 13, 4, 10, 0, 7, 6, 3, 9, 2, 8, 11},
		{13, 11, 7, 14, 12, 1, 3, 9, 5, 0, 15, 4, 8, 6, 2, 10},
		{6, 15, 14, 9, 11, 3, 0, 8, 12, 2, 13, 7, 1, 4, 10, 5},
		{10, 2, 8, 4, 7, 6, 1, 5, 15, 11, 9, 14, 3, 12, 13, 0},
		{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		{14, 10, 4, 8, 9, 15, 13, 6, 1, 12, 0, 2, 11, 7, 5, 3},
	}

	v := [16]uint64{
		h[0], h[1], h[2], h[3], h[4], h[5], h[6], h[7],
		iv[0], iv[1], iv[2], iv[3],
		iv[4] ^ t[0], iv[5] ^ t[1], iv[6], iv[7],
	}
	if final {
		v[14] ^= 0xffffffffffffffff
	}

	g := func(a, b, c, d int, x, y uint64) {
		v[a] = v[a] + v[b] + x
		v[d] = bits64RotR(v[d]^v[a], 32)
		v[c] = v[c] + v[d]
		v[b] = bits64RotR(v[b]^v[c], 24)
		v[a] = v[a] + v[b] + y
		v[d] = bits64RotR(v[d]^v[a], 16)
		v[c] = v[c] + v[d]
		v[b] = bits64RotR(v[b]^v[c], 63)
	}

	for i := uint32(0); i < rounds; i++ {
		s := sigma[i%12]
		g(0, 4, 8, 12, m[s[0]], m[s[1]])
		g(1, 5, 9, 13, m[s[2]], m[s[3]])
		g(2, 6, 10, 14, m[s[4]], m[s[5]])
		g(3, 7, 11, 15, m[s[6]], m[s[7]])
		g(0, 5, 10, 15, m[s[8]], m[s[9]])
		g(1, 6, 11, 12, m[s[10]], m[s[11]])
		g(2, 7, 8, 13, m[s[12]], m[s[13]])
		g(3, 4, 9, 14, m[s[14]], m[s[15]])
	}

	for i := 0; i < 8; i++ {
		h[i] ^= v[i] ^ v[i+8]
	}
}

func bits64RotR(x uint64, n uint) uint64 {
	return (x >> n) | (x << (64 - n))
}

func wordCount(b []byte) uint64 {
	return (uint64(len(b)) + 31) / 32
}

func zero32() []byte {
	return make([]byte, 32)
}

func pad32(b []byte) []byte {
	if len(b) == 32 {
		return b
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

func IsPrecompile(addr [20]byte) bool {
	for i := 0; i < 19; i++ {
		if addr[i] != 0 {
			return false
		}
	}
	return addr[19] >= 1 && addr[19] <= 9
}

var _ = secp256k1.S256()
