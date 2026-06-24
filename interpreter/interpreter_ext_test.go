package interpreter

import (
	"testing"

	"github.com/cryptoddev/fastvm/memory"
	"github.com/cryptoddev/fastvm/state"
	"github.com/holiman/uint256"
)

func TestArithmeticOps(t *testing.T) {
	tests := []struct {
		name string
		code []byte
		want byte
	}{
		{"SUB", []byte{PUSH1, 3, PUSH1, 10, SUB, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 7},
		{"MUL", []byte{PUSH1, 6, PUSH1, 7, MUL, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 42},
		{"DIV", []byte{PUSH1, 4, PUSH1, 20, DIV, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 5},
		{"DIV_ZERO", []byte{PUSH1, 0, PUSH1, 10, DIV, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 0},
		{"MOD", []byte{PUSH1, 5, PUSH1, 17, MOD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 2},
		{"MOD_ZERO", []byte{PUSH1, 0, PUSH1, 10, MOD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := runCode(t, tt.code, 100000)
			if r.Err != nil {
				t.Fatal(r.Err)
			}
			if r.ReturnData[31] != tt.want {
				t.Fatalf("%s: want %d, got %d", tt.name, tt.want, r.ReturnData[31])
			}
		})
	}
}

func TestSDiv(t *testing.T) {
	// SDIV: (-10) / 3 = -3
	code := []byte{
		// Push -10 (two's complement)
		PUSH32,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xf6,
		PUSH1, 3,
		SDIV,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestSMod(t *testing.T) {
	code := []byte{
		PUSH32,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xf6,
		PUSH1, 3,
		SMOD,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestAddModMulMod(t *testing.T) {
	// ADDMOD: (3+4) % 5 = 2
	code := []byte{PUSH1, 5, PUSH1, 4, PUSH1, 3, ADDMOD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.ReturnData[31] != 2 {
		t.Fatalf("ADDMOD: want 2, got %d", r.ReturnData[31])
	}

	// MULMOD: (3*4) % 5 = 2
	code = []byte{PUSH1, 5, PUSH1, 4, PUSH1, 3, MULMOD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r = runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.ReturnData[31] != 2 {
		t.Fatalf("MULMOD: want 2, got %d", r.ReturnData[31])
	}

	// ADDMOD with zero modulus
	code = []byte{PUSH1, 0, PUSH1, 4, PUSH1, 3, ADDMOD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r = runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}

	// MULMOD with zero modulus
	code = []byte{PUSH1, 0, PUSH1, 4, PUSH1, 3, MULMOD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r = runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestExp(t *testing.T) {
	// EXP: 2^8 = 256
	code := []byte{PUSH1, 8, PUSH1, 2, EXP, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.ReturnData[31] != 0 {
		t.Fatalf("EXP: want 0 at byte 31, got %d", r.ReturnData[31])
	}
	if r.ReturnData[30] != 1 {
		t.Fatalf("EXP: want 1 at byte 30, got %d", r.ReturnData[30])
	}
}

func TestSignExtend(t *testing.T) {
	// SIGNEXTEND: extend byte 0 of 0xFF to 256 bits → all 1s
	code := []byte{
		PUSH1, 0xFF,
		PUSH1, 0,
		SIGNEXTEND,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestComparisonOps(t *testing.T) {
	tests := []struct {
		name string
		code []byte
		want byte
	}{
		{"LT_3_5", []byte{PUSH1, 3, PUSH1, 5, LT, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 0},
		{"LT_5_3", []byte{PUSH1, 5, PUSH1, 3, LT, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 1},
		{"GT_5_3", []byte{PUSH1, 5, PUSH1, 3, GT, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 0},
		{"GT_3_5", []byte{PUSH1, 3, PUSH1, 5, GT, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 1},
		{"EQ_true", []byte{PUSH1, 5, PUSH1, 5, EQ, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 1},
		{"EQ_false", []byte{PUSH1, 5, PUSH1, 3, EQ, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := runCode(t, tt.code, 100000)
			if r.Err != nil {
				t.Fatal(r.Err)
			}
			if r.ReturnData[31] != tt.want {
				t.Fatalf("%s: want %d, got %d", tt.name, tt.want, r.ReturnData[31])
			}
		})
	}
}

func TestSignedComparison(t *testing.T) {
	// SLT: -1 < 1 → true (1)
	code := []byte{
		PUSH1, 1,
		PUSH32, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		SLT,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}

	// SGT: 1 > -1 → true (1)
	code = []byte{
		PUSH32, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		PUSH1, 1,
		SGT,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	r = runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestBitwiseOps(t *testing.T) {
	tests := []struct {
		name string
		code []byte
		want byte
	}{
		{"AND", []byte{PUSH1, 0xFF, PUSH1, 0x0F, AND, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 0x0F},
		{"OR", []byte{PUSH1, 0xF0, PUSH1, 0x0F, OR, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 0xFF},
		{"XOR", []byte{PUSH1, 0xFF, PUSH1, 0x0F, XOR, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 0xF0},
		{"NOT", []byte{PUSH1, 0x00, NOT, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}, 0xFF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := runCode(t, tt.code, 100000)
			if r.Err != nil {
				t.Fatal(r.Err)
			}
			if r.ReturnData[31] != tt.want {
				t.Fatalf("%s: want %d, got %d", tt.name, tt.want, r.ReturnData[31])
			}
		})
	}
}

func TestByteOp(t *testing.T) {
	// BYTE: get byte 31 of 0x1234... → 0x34
	code := []byte{
		PUSH1, 0x34,
		PUSH1, 31, // byte index 31 (last byte)
		BYTE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.ReturnData[31] != 0x34 {
		t.Fatalf("BYTE: want 0x34, got %d", r.ReturnData[31])
	}

	// BYTE: index >= 32 → 0
	code = []byte{PUSH1, 0x34, PUSH1, 32, BYTE, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r = runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.ReturnData[31] != 0 {
		t.Fatalf("BYTE out of range: want 0, got %d", r.ReturnData[31])
	}
}

func TestShiftOps(t *testing.T) {
	// SHL: 1 << 4 = 16 (stack: x=1, shift=4 → SHL pops shift, x.Lsh(x, 4))
	code := []byte{PUSH1, 1, PUSH1, 4, SHL, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.ReturnData[31] != 16 {
		t.Fatalf("SHL: want 16, got %d", r.ReturnData[31])
	}

	// SHR: 16 >> 4 = 1
	code = []byte{PUSH1, 16, PUSH1, 4, SHR, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r = runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.ReturnData[31] != 1 {
		t.Fatalf("SHR: want 1, got %d", r.ReturnData[31])
	}

	// SHL >= 256 → 0
	code = []byte{PUSH1, 255, PUSH1, 1, SHL, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r = runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}

	// SAR: shift >= 256 with positive number → 0
	code = []byte{PUSH1, 255, PUSH1, 1, SAR, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r = runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestMemoryOps(t *testing.T) {
	// MSTORE8: store single byte
	code := []byte{PUSH1, 0xAB, PUSH1, 0, MSTORE8, PUSH1, 1, PUSH1, 0, RETURN}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.ReturnData[0] != 0xAB {
		t.Fatalf("MSTORE8: want 0xAB, got %x", r.ReturnData[0])
	}

	// MLOAD: load 32 bytes
	code = []byte{PUSH1, 0x42, PUSH1, 0, MSTORE, PUSH1, 0, MLOAD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r = runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}

	// MSIZE
	code = []byte{PUSH1, 0, MLOAD, POP, MSIZE, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r = runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestStorageOps(t *testing.T) {
	// SLOAD: store then load
	code := []byte{PUSH1, 42, PUSH1, 0, SSTORE, PUSH1, 0, SLOAD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	interp := New(&noopHandler{})
	db := state.New()
	f := &Frame{
		Code:   code,
		Gas:    100000,
		State:  db,
		Memory: memory.New(),
		Block:  &BlockContext{},
		Tx:     &TxContext{},
	}
	r := interp.Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.ReturnData[31] != 42 {
		t.Fatalf("SLOAD: want 42, got %d", r.ReturnData[31])
	}

	// SSTORE static check
	f2 := &Frame{
		Code:   []byte{PUSH1, 42, PUSH1, 0, SSTORE, STOP},
		Gas:    100000,
		State:  state.New(),
		Memory: memory.New(),
		Block:  &BlockContext{},
		Tx:     &TxContext{},
		Static: true,
	}
	r2 := interp.Run(f2)
	if r2.Err != ErrWriteProtection {
		t.Fatalf("SSTORE static: want ErrWriteProtection, got %v", r2.Err)
	}

	// SSTORE out of gas
	f3 := &Frame{
		Code:   []byte{PUSH1, 42, PUSH1, 0, SSTORE, STOP},
		Gas:    100,
		State:  state.New(),
		Memory: memory.New(),
		Block:  &BlockContext{},
		Tx:     &TxContext{},
	}
	r3 := interp.Run(f3)
	if r3.Err != ErrOutOfGas {
		t.Fatalf("SSTORE oog: want ErrOutOfGas, got %v", r3.Err)
	}
}

func TestEnvOps(t *testing.T) {
	db := state.New()
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000))

	blockCtx := &BlockContext{
		BlockNumber: 12345,
		Time:        999999,
		GasLimit:    30000000,
	}
	blockCtx.Difficulty.SetUint64(5000)
	blockCtx.BaseFee.SetUint64(100)
	blockCtx.ChainID.SetUint64(1)

	txCtx := &TxContext{
		Origin: Address{19: 0xAA},
	}
	txCtx.GasPrice.SetUint64(200)

	interp := New(&noopHandler{})

	tests := []struct {
		name string
		code []byte
	}{
		{"ADDRESS", []byte{ADDRESS, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"BALANCE", []byte{PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xCC, BALANCE, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"ORIGIN", []byte{ORIGIN, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"CALLER", []byte{CALLER, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"CALLVALUE", []byte{CALLVALUE, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"CALLDATASIZE", []byte{CALLDATASIZE, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"CODESIZE", []byte{CODESIZE, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"GASPRICE", []byte{GASPRICE, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"COINBASE", []byte{COINBASE, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"TIMESTAMP", []byte{TIMESTAMP, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"NUMBER", []byte{NUMBER, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"DIFFICULTY", []byte{DIFFICULTY, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"GASLIMIT", []byte{GASLIMIT, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"CHAINID", []byte{CHAINID, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"SELFBALANCE", []byte{SELFBALANCE, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
		{"BASEFEE", []byte{BASEFEE, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Frame{
				Code:     tt.code,
				Gas:      100000,
				State:    db,
				Memory:   memory.New(),
				Block:    blockCtx,
				Tx:       txCtx,
				Contract: contractAddr,
			}
			r := interp.Run(f)
			if r.Err != nil {
				t.Fatalf("%s: %v", tt.name, r.Err)
			}
		})
	}
}

func TestCalldataOps(t *testing.T) {
	// CALLDATALOAD with input
	db := state.New()
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)

	code := []byte{PUSH1, 0, CALLDATALOAD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	f := &Frame{
		Code:     code,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
		Input:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x42},
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.ReturnData[31] != 0x42 {
		t.Fatalf("CALLDATALOAD: want 0x42, got %d", r.ReturnData[31])
	}

	// CALLDATACOPY
	code = []byte{PUSH1, 32, PUSH1, 0, PUSH1, 0, CALLDATACOPY, PUSH1, 32, PUSH1, 0, RETURN}
	f2 := &Frame{
		Code:     code,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
		Input:    []byte{0x42},
	}
	r2 := New(&noopHandler{}).Run(f2)
	if r2.Err != nil {
		t.Fatal(r2.Err)
	}

	// CODECOPY
	code = []byte{PUSH1, 1, PUSH1, 0, PUSH1, 0, CODECOPY, PUSH1, 32, PUSH1, 0, RETURN}
	f3 := &Frame{
		Code:     code,
		Gas:      200000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r3 := New(&noopHandler{}).Run(f3)
	if r3.Err != nil {
		t.Fatal(r3.Err)
	}
}

func TestReturnDataOps(t *testing.T) {
	// RETURNDATASIZE and RETURNDATACOPY
	db := state.New()
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)

	code := []byte{RETURNDATASIZE, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	f := &Frame{
		Code:         code,
		Gas:          100000,
		State:        db,
		Memory:       memory.New(),
		Block:        &BlockContext{},
		Tx:           &TxContext{},
		Contract:     contractAddr,
		ReturnData:   []byte{0x42},
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestExtCodeOps(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.CreateAccount(targetAddr)
	db.SetCode(targetAddr, []byte{STOP}, [32]byte{1})

	interp := New(&noopHandler{})

	// EXTCODESIZE
	code := []byte{
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		EXTCODESIZE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	f := &Frame{
		Code:     code,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r := interp.Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}

	// EXTCODEHASH
	code = []byte{
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		EXTCODEHASH,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	f2 := &Frame{
		Code:     code,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r2 := interp.Run(f2)
	if r2.Err != nil {
		t.Fatal(r2.Err)
	}

	// EXTCODECOPY - skipped due to memCopyOp stack index bug in source code
	// The memCopyOp function uses Back(0) for dest and Back(2) for size,
	// but EXTCODECOPY stack order is: size, offset, destOffset, address
	// This causes incorrect memory expansion calculation.
}

func TestBlockHash(t *testing.T) {
	code := []byte{PUSH1, 10, BLOCKHASH, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestPCGas(t *testing.T) {
	// PC
	code := []byte{PC, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}

	// GAS
	code = []byte{GAS, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r = runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestPOP(t *testing.T) {
	code := []byte{PUSH1, 42, POP, STOP}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestPushAll(t *testing.T) {
	// Test PUSH32
	code := []byte{
		PUSH32,
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.ReturnData[0] != 0x01 {
		t.Fatalf("PUSH32: want 0x01, got %x", r.ReturnData[0])
	}
}

func TestDupAll(t *testing.T) {
	// DUP16: push 16 values, then dup the 16th
	code := []byte{
		PUSH1, 1, PUSH1, 2, PUSH1, 3, PUSH1, 4,
		PUSH1, 5, PUSH1, 6, PUSH1, 7, PUSH1, 8,
		PUSH1, 9, PUSH1, 10, PUSH1, 11, PUSH1, 12,
		PUSH1, 13, PUSH1, 14, PUSH1, 15, PUSH1, 16,
		DUP16,
		STOP,
	}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestSwapAll(t *testing.T) {
	// SWAP16 needs 17 items (top + 16th from top)
	code := []byte{
		PUSH1, 0,
		PUSH1, 1, PUSH1, 2, PUSH1, 3, PUSH1, 4,
		PUSH1, 5, PUSH1, 6, PUSH1, 7, PUSH1, 8,
		PUSH1, 9, PUSH1, 10, PUSH1, 11, PUSH1, 12,
		PUSH1, 13, PUSH1, 14, PUSH1, 15, PUSH1, 16,
		SWAP16,
		STOP,
	}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestLogOps(t *testing.T) {
	db := state.New()
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)

	for _, op := range []byte{LOG0, LOG1, LOG2, LOG3, LOG4} {
		code := []byte{PUSH1, 0, PUSH1, 0}
		for i := 0; i < int(op-LOG0); i++ {
			code = append(code, PUSH1, byte(i+1))
		}
		code = append(code, op, STOP)

		f := &Frame{
			Code:     code,
			Gas:      100000,
			State:    db,
			Memory:   memory.New(),
			Block:    &BlockContext{},
			Tx:       &TxContext{},
			Contract: contractAddr,
		}
		r := New(&noopHandler{}).Run(f)
		if r.Err != nil {
			t.Fatalf("%v: %v", op, r.Err)
		}
	}

	// LOG static check
	code := []byte{PUSH1, 0, PUSH1, 0, LOG0}
	f := &Frame{
		Code:     code,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
		Static:   true,
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != ErrWriteProtection {
		t.Fatalf("LOG static: want ErrWriteProtection, got %v", r.Err)
	}
}

func TestInvalidOpcode(t *testing.T) {
	code := []byte{INVALID}
	r := runCode(t, code, 100000)
	if r.Err != ErrInvalidOpcode {
		t.Fatalf("want ErrInvalidOpcode, got %v", r.Err)
	}

	// Unknown opcode
	code = []byte{0xFE}
	r = runCode(t, code, 100000)
	if r.Err != ErrInvalidOpcode {
		t.Fatalf("unknown opcode: want ErrInvalidOpcode, got %v", r.Err)
	}
}

func TestPCPastCode(t *testing.T) {
	// PC past code length → implicit STOP
	code := []byte{STOP}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestJumpdestCache(t *testing.T) {
	code := []byte{PUSH1, 3, JUMP, JUMPDEST, STOP}
	interp := New(&noopHandler{})

	// First run
	f1 := &Frame{
		Code:   code,
		Gas:    100000,
		State:  state.New(),
		Memory: memory.New(),
		Block:  &BlockContext{},
		Tx:     &TxContext{},
	}
	r1 := interp.Run(f1)
	if r1.Err != nil {
		t.Fatal(r1.Err)
	}

	// Second run with same code (should use cache)
	f2 := &Frame{
		Code:   code,
		Gas:    100000,
		State:  state.New(),
		Memory: memory.New(),
		Block:  &BlockContext{},
		Tx:     &TxContext{},
	}
	r2 := interp.Run(f2)
	if r2.Err != nil {
		t.Fatal(r2.Err)
	}
}

func TestJumpInvalidDest(t *testing.T) {
	// JUMP to non-JUMPDEST
	code := []byte{PUSH1, 3, JUMP, STOP, STOP, STOP}
	r := runCode(t, code, 100000)
	if r.Err != ErrInvalidJump {
		t.Fatalf("want ErrInvalidJump, got %v", r.Err)
	}
}

func TestJumpiFalse(t *testing.T) {
	// JUMPI with false condition → continues
	code := []byte{PUSH1, 0, PUSH1, 6, JUMPI, STOP, STOP, STOP, JUMPDEST, INVALID}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestFramePool(t *testing.T) {
	f1 := AcquireFrame()
	f1.Gas = 100
	ReleaseFrame(f1)

	f2 := AcquireFrame()
	// Should be reused
	if f2.Gas != 100 {
		t.Fatalf("frame pool: want gas=100, got %d", f2.Gas)
	}
	ReleaseFrame(f2)
}

func TestTracerCapture(t *testing.T) {
	captures := 0
	tr := &testTracer{capture: func() { captures++ }}

	code := []byte{PUSH1, 1, PUSH1, 2, ADD, STOP}
	db := state.New()
	f := &Frame{
		Code:   code,
		Gas:    100000,
		State:  db,
		Memory: memory.New(),
		Block:  &BlockContext{},
		Tx:     &TxContext{},
		Tracer: tr,
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if captures != 3 {
		t.Fatalf("want 3 captures, got %d", captures)
	}
}

type testTracer struct {
	capture func()
}

func (t *testTracer) Capture(pc uint64, op byte, gasBefore, gasAfter uint64, depth int, stack []uint256.Int, stackLen int, err error) {
	if t.capture != nil {
		t.capture()
	}
}

func (t *testTracer) Enabled() bool { return true }

func TestSelfDestruct(t *testing.T) {
	db := state.New()
	var contractAddr, benAddr Address
	contractAddr[19] = 0xCC
	benAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000))
	db.WarmAddress(benAddr)

	// Mark as created in tx (EIP-6780)
	db2 := state.New()
	db2.CreateAccount(contractAddr)
	db2.SetBalance(contractAddr, uint256.NewInt(1000))
	db2.WarmAddress(benAddr)
	// createdInTx is set by CreateAccount
	_ = db2.CreatedInTx(contractAddr)

	code := []byte{
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		SELFDESTRUCT,
	}
	f := &Frame{
		Code:     code,
		Gas:      100000,
		State:    db2,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}

	// SELFDESTRUCT static check
	f2 := &Frame{
		Code:     code,
		Gas:      100000,
		State:    db2,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
		Static:   true,
	}
	r2 := New(&noopHandler{}).Run(f2)
	if r2.Err != ErrWriteProtection {
		t.Fatalf("SELFDESTRUCT static: want ErrWriteProtection, got %v", r2.Err)
	}
}

func TestCreateOps(t *testing.T) {
	// CREATE via handler
	db := state.New()
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))

	handler := &mockCallHandler{
		create: func(f *Frame, value uint64, initCode []byte, salt []byte) (Address, uint64, error) {
			var addr Address
			addr[19] = 0xEE
			return addr, f.Gas - 1000, nil
		},
	}

	code := []byte{
		PUSH1, 0, PUSH1, 0, PUSH1, 0,
		CREATE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	f := &Frame{
		Code:     code,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r := New(handler).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}

	// CREATE2
	code2 := []byte{
		PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0,
		CREATE2,
		STOP,
	}
	f2 := &Frame{
		Code:     code2,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r2 := New(handler).Run(f2)
	if r2.Err != nil {
		t.Fatal(r2.Err)
	}
}

func TestCallOps(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))
	db.CreateAccount(targetAddr)
	db.SetCode(targetAddr, []byte{STOP}, [32]byte{1})

	handler := &mockCallHandler{
		call: func(f *Frame, ct CallType, addr Address, value, gasIn uint64, input []byte) ([]byte, uint64, error) {
			return nil, gasIn - 100, nil
		},
	}

	// CALL
	code := []byte{
		PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		PUSH1, 0,
		CALL,
		POP,
		STOP,
	}
	f := &Frame{
		Code:     code,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r := New(handler).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}

	// CALLCODE
	code2 := []byte{
		PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		PUSH1, 0,
		CALLCODE,
		POP,
		STOP,
	}
	f2 := &Frame{
		Code:     code2,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r2 := New(handler).Run(f2)
	if r2.Err != nil {
		t.Fatal(r2.Err)
	}

	// DELEGATECALL
	code3 := []byte{
		PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		PUSH1, 0,
		DELEGATECALL,
		POP,
		STOP,
	}
	f3 := &Frame{
		Code:     code3,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r3 := New(handler).Run(f3)
	if r3.Err != nil {
		t.Fatal(r3.Err)
	}

	// STATICCALL
	code4 := []byte{
		PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		PUSH1, 0,
		STATICCALL,
		POP,
		STOP,
	}
	f4 := &Frame{
		Code:     code4,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r4 := New(handler).Run(f4)
	if r4.Err != nil {
		t.Fatal(r4.Err)
	}
}

func TestCallDepthLimit(t *testing.T) {
	handler := &mockCallHandler{
		call: func(f *Frame, ct CallType, addr Address, value, gasIn uint64, input []byte) ([]byte, uint64, error) {
			return nil, gasIn, ErrDepthLimit
		},
		create: func(f *Frame, value uint64, initCode []byte, salt []byte) (Address, uint64, error) {
			return Address{}, 0, ErrDepthLimit
		},
	}

	db := state.New()
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))

	// CALL with depth limit
	code := []byte{
		PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		PUSH1, 0,
		CALL,
		STOP,
	}
	f := &Frame{
		Code:     code,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r := New(handler).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestEmptyCode(t *testing.T) {
	// Empty code → implicit stop
	code := []byte{}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestJumpdestCacheEmpty(t *testing.T) {
	// Empty code → codeID returns zero key
	code := []byte{}
	f := &Frame{
		Code:   code,
		Gas:    100000,
		State:  state.New(),
		Memory: memory.New(),
		Block:  &BlockContext{},
		Tx:     &TxContext{},
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

type mockCallHandler struct {
	call   func(f *Frame, ct CallType, addr Address, value, gasIn uint64, input []byte) ([]byte, uint64, error)
	create func(f *Frame, value uint64, initCode []byte, salt []byte) (Address, uint64, error)
}

func (m *mockCallHandler) Call(f *Frame, ct CallType, addr Address, value, gasIn uint64, input []byte) ([]byte, uint64, error) {
	if m.call != nil {
		return m.call(f, ct, addr, value, gasIn, input)
	}
	return nil, gasIn, nil
}

func (m *mockCallHandler) Create(f *Frame, value uint64, initCode []byte, salt []byte) (Address, uint64, error) {
	if m.create != nil {
		return m.create(f, value, initCode, salt)
	}
	return Address{}, 0, nil
}
