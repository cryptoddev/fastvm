package interpreter

import (
	"testing"

	"github.com/cryptoddev/fastvm/memory"
	"github.com/cryptoddev/fastvm/state"
	"github.com/holiman/uint256"
)

// noopHandler is a CallHandler that does nothing (for unit tests).
type noopHandler struct{}

func (n *noopHandler) Call(_ *Frame, _ CallType, _ Address, _, _ uint64, _ []byte) ([]byte, uint64, error) {
	return nil, 0, nil
}
func (n *noopHandler) Create(_ *Frame, _ uint64, _ []byte, _ []byte) (Address, uint64, error) {
	return Address{}, 0, nil
}

func newTestFrame(code []byte, gas uint64) *Frame {
	db := state.New()
	return &Frame{
		Code:   code,
		Gas:    gas,
		State:  db,
		Memory: memory.New(),
		Block:  &BlockContext{},
		Tx:     &TxContext{},
	}
}

func runCode(t *testing.T, code []byte, gas uint64) ExecResult {
	t.Helper()
	interp := New(&noopHandler{})
	f := newTestFrame(code, gas)
	return interp.Run(f)
}

func TestAdd(t *testing.T) {
	// PUSH1 3, PUSH1 4, ADD, RETURN(0,32) — result in memory
	// Simpler: just check via MSTORE + RETURN
	code := []byte{
		PUSH1, 3,
		PUSH1, 4,
		ADD,        // stack: [7]
		PUSH1, 0,   // offset
		MSTORE,     // mem[0:32] = 7
		PUSH1, 32,
		PUSH1, 0,
		RETURN,
	}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if len(r.ReturnData) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(r.ReturnData))
	}
	if r.ReturnData[31] != 7 {
		t.Fatalf("want 7, got %d", r.ReturnData[31])
	}
}

func TestPushAndStop(t *testing.T) {
	code := []byte{PUSH1, 0x42, STOP}
	r := runCode(t, code, 10000)
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
}

func TestReturn(t *testing.T) {
	// PUSH1 0x42, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
	code := []byte{
		PUSH1, 0x42,
		PUSH1, 0,
		MSTORE,
		PUSH1, 32,
		PUSH1, 0,
		RETURN,
	}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if len(r.ReturnData) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(r.ReturnData))
	}
	if r.ReturnData[31] != 0x42 {
		t.Fatalf("want 0x42 at byte 31, got %x", r.ReturnData[31])
	}
}

func TestRevert(t *testing.T) {
	code := []byte{PUSH1, 0, PUSH1, 0, REVERT}
	r := runCode(t, code, 100000)
	if r.Err != ErrRevert {
		t.Fatalf("want ErrRevert, got %v", r.Err)
	}
}

func TestJump(t *testing.T) {
	// PUSH1 4, JUMP, INVALID, JUMPDEST, PUSH1 1, STOP
	code := []byte{PUSH1, 4, JUMP, INVALID, JUMPDEST, PUSH1, 1, STOP}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
}

func TestJumpi(t *testing.T) {
	// PUSH1 1 (cond), PUSH1 6 (dest), JUMPI, INVALID, INVALID, INVALID, JUMPDEST, STOP
	code := []byte{PUSH1, 1, PUSH1, 6, JUMPI, INVALID, JUMPDEST, STOP}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
}

func TestSha3(t *testing.T) {
	// Store 0xdeadbeef at mem[0], hash 4 bytes.
	// PUSH4 0xdeadbeef, PUSH1 0, MSTORE, PUSH1 4, PUSH1 28, SHA3, STOP
	code := []byte{
		PUSH4, 0xde, 0xad, 0xbe, 0xef,
		PUSH1, 0,
		MSTORE,
		PUSH1, 4,
		PUSH1, 28,
		SHA3,
		STOP,
	}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
}

func TestOutOfGas(t *testing.T) {
	code := []byte{PUSH1, 1, PUSH1, 2, ADD, STOP}
	r := runCode(t, code, 1) // only 1 gas — not enough
	if r.Err != ErrOutOfGas {
		t.Fatalf("want ErrOutOfGas, got %v", r.Err)
	}
}

func TestSstore(t *testing.T) {
	// PUSH1 42 (value), PUSH1 1 (slot), SSTORE, STOP
	code := []byte{PUSH1, 42, PUSH1, 1, SSTORE, STOP}
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
		t.Fatalf("unexpected error: %v", r.Err)
	}
	var slot [32]byte
	slot[31] = 1
	val := db.GetStorage(f.Contract, slot)
	if val[31] != 42 {
		t.Fatalf("want 42, got %d", val[31])
	}
}

func TestDupSwap(t *testing.T) {
	// PUSH1 1, PUSH1 2, DUP1 → stack: [1,2,2]; SWAP1 → [1,2,2] swapped top two → [1,2,2]
	code := []byte{PUSH1, 1, PUSH1, 2, DUP1, SWAP1, STOP}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
}

func TestIsZero(t *testing.T) {
	f := newTestFrame(nil, 0)
	f.Stack.Push(uint256.NewInt(0))
	opIsZero(f)
	top, _ := f.Stack.Peek()
	if top.Uint64() != 1 {
		t.Fatalf("ISZERO(0) should be 1, got %d", top.Uint64())
	}

	f.Stack.Push(uint256.NewInt(5))
	opIsZero(f)
	top, _ = f.Stack.Peek()
	if top.Uint64() != 0 {
		t.Fatalf("ISZERO(5) should be 0, got %d", top.Uint64())
	}
}


