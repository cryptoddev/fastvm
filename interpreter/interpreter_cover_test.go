package interpreter

import (
	"testing"

	"github.com/cryptoddev/fastvm/memory"
	"github.com/cryptoddev/fastvm/stack"
	"github.com/cryptoddev/fastvm/state"
	"github.com/holiman/uint256"
)

// frame.go: bitvec.has out-of-bounds
func TestBitvecHasOutOfBounds(t *testing.T) {
	bv := make(bitvec, 1)
	if bv.has(8) {
		t.Fatal("expected false for out-of-bounds index")
	}
}

// interpreter.go: errRevert.Error()
func TestErrRevertError(t *testing.T) {
	if ErrRevert.Error() != "revert" {
		t.Fatal("unexpected error message")
	}
}

// interpreter.go: expandMem with size==0
func TestExpandMemZeroSize(t *testing.T) {
	f := newTestFrame(nil, 100)
	if err := expandMem(f, 100, 0); err != nil {
		t.Fatal(err)
	}
}

// interpreter.go: memCopyOp with size>0, wordCost OOG
func TestMemCopyOpWordCostOOG(t *testing.T) {
	f := newTestFrame(nil, 100)
	dummy := uint64(100)
	f.Memory.Expand(0, 32, &dummy)
	f.Gas = 2
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(32)
	err := memCopyOp(f, func() error { return nil })
	if err != ErrOutOfGas {
		t.Fatalf("want ErrOutOfGas, got %v", err)
	}
}

// interpreter.go: logOp OOG
func TestLogOpOOG(t *testing.T) {
	f := newTestFrame(nil, 374)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	err := logOp(f, 0)
	if err != ErrOutOfGas {
		t.Fatalf("want ErrOutOfGas, got %v", err)
	}
}

// interpreter.go: chargeExtAccess warm address OOG
func TestChargeExtAccessWarmOOG(t *testing.T) {
	db := state.New()
	var addr Address
	addr[19] = 0x01
	db.WarmAddress(addr)
	f := newTestFrame(nil, 99)
	f.State = db
	err := chargeExtAccess(f, addr)
	if err != ErrOutOfGas {
		t.Fatalf("want ErrOutOfGas, got %v", err)
	}
}

// interpreter.go: chargeExtAccess cold address OOG
func TestChargeExtAccessColdOOG(t *testing.T) {
	db := state.New()
	var addr Address
	addr[19] = 0x01
	f := newTestFrame(nil, 2599)
	f.State = db
	err := chargeExtAccess(f, addr)
	if err != ErrOutOfGas {
		t.Fatalf("want ErrOutOfGas, got %v", err)
	}
}

// op_arithmetic.go: Pop error for all one-Pop functions
func TestArithmeticOpOnePopErrors(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*Frame) error
	}{
		{"opAdd", opAdd},
		{"opMul", opMul},
		{"opSub", opSub},
		{"opDiv", opDiv},
		{"opSDiv", opSDiv},
		{"opMod", opMod},
		{"opSMod", opSMod},
		{"opExp", opExp},
		{"opSignExtend", opSignExtend},
		{"opLt", opLt},
		{"opGt", opGt},
		{"opSlt", opSlt},
		{"opSgt", opSgt},
		{"opEq", opEq},
		{"opAnd", opAnd},
		{"opOr", opOr},
		{"opXor", opXor},
		{"opByte", opByte},
		{"opShl", opShl},
		{"opShr", opShr},
		{"opSar", opSar},
		{"opSha3", opSha3},
	}
	for _, tt := range tests {
		f := newTestFrame(nil, 0)
		if err := tt.fn(f); err != stack.ErrStackUnderflow {
			t.Errorf("%s: want stack underflow, got %v", tt.name, err)
		}
	}
}

// op_arithmetic.go: Pop error for two-Pop functions
func TestArithmeticOpTwoPopErrors(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*Frame) error
	}{
		{"opAddMod", opAddMod},
		{"opMulMod", opMulMod},
	}
	for _, tt := range tests {
		f := newTestFrame(nil, 0)
		if err := tt.fn(f); err != stack.ErrStackUnderflow {
			t.Errorf("%s (empty): want stack underflow, got %v", tt.name, err)
		}

		f2 := newTestFrame(nil, 0)
		f2.Stack.Push(uint256.NewInt(0))
		if err := tt.fn(f2); err != stack.ErrStackUnderflow {
			t.Errorf("%s (1 item): want stack underflow, got %v", tt.name, err)
		}
	}
}

// op_memory.go: opMstore Pop errors
func TestOpMstorePopError(t *testing.T) {
	f := newTestFrame(nil, 0)
	if err := opMstore(f); err != stack.ErrStackUnderflow {
		t.Fatalf("want stack underflow, got %v", err)
	}

	f2 := newTestFrame(nil, 0)
	f2.Stack.Push(uint256.NewInt(0))
	if err := opMstore(f2); err != stack.ErrStackUnderflow {
		t.Fatalf("want stack underflow, got %v", err)
	}
}

// op_memory.go: opMstore8 Pop errors
func TestOpMstore8PopError(t *testing.T) {
	f := newTestFrame(nil, 0)
	if err := opMstore8(f); err != stack.ErrStackUnderflow {
		t.Fatalf("want stack underflow, got %v", err)
	}

	f2 := newTestFrame(nil, 0)
	f2.Stack.Push(uint256.NewInt(0))
	if err := opMstore8(f2); err != stack.ErrStackUnderflow {
		t.Fatalf("want stack underflow, got %v", err)
	}
}

// op_memory.go: opSstore Pop errors
func TestOpSstorePopError(t *testing.T) {
	f := newTestFrame(nil, 0)
	if err := opSstore(f); err != stack.ErrStackUnderflow {
		t.Fatalf("want stack underflow, got %v", err)
	}

	f2 := newTestFrame(nil, 0)
	f2.Stack.Push(uint256.NewInt(0))
	if err := opSstore(f2); err != stack.ErrStackUnderflow {
		t.Fatalf("want stack underflow, got %v", err)
	}
}

// op_memory.go: opJump Pop error
func TestOpJumpPopError(t *testing.T) {
	f := newTestFrame(nil, 0)
	if err := opJump(f); err != stack.ErrStackUnderflow {
		t.Fatalf("want stack underflow, got %v", err)
	}
}

// op_memory.go: opJumpi Pop errors
func TestOpJumpiPopError(t *testing.T) {
	f := newTestFrame(nil, 0)
	if err := opJumpi(f); err != stack.ErrStackUnderflow {
		t.Fatalf("want stack underflow, got %v", err)
	}

	f2 := newTestFrame(nil, 0)
	f2.Stack.Push(uint256.NewInt(0))
	if err := opJumpi(f2); err != stack.ErrStackUnderflow {
		t.Fatalf("want stack underflow, got %v", err)
	}
}

// op_memory.go: opJumpdest (no-op)
func TestOpJumpdest(t *testing.T) {
	if err := opJumpdest(nil); err != nil {
		t.Fatal(err)
	}
}

// op_memory.go: opPush partial push past code end
func TestOpPushPastCodeEnd(t *testing.T) {
	code := []byte{PUSH2, 0x01}
	f := newTestFrame(code, 100000)
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	top, _ := f.Stack.Peek()
	if top.Uint64() != 1 {
		t.Fatalf("want 1, got %d", top.Uint64())
	}
}

// op_env.go: opCalldataCopy with sz==0
func TestOpCalldataCopyZeroSize(t *testing.T) {
	f := newTestFrame(nil, 100000)
	f.Input = []byte{1, 2, 3}
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	if err := opCalldataCopy(f); err != nil {
		t.Fatal(err)
	}
}

// op_env.go: opCodeCopy with sz==0
func TestOpCodeCopyZeroSize(t *testing.T) {
	f := newTestFrame([]byte{STOP}, 100000)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	if err := opCodeCopy(f); err != nil {
		t.Fatal(err)
	}
}

// op_env.go: opExtCodeCopy (full coverage)
func TestOpExtCodeCopy(t *testing.T) {
	db := state.New()
	var addr Address
	addr[19] = 0xCC
	db.CreateAccount(addr)
	db.SetCode(addr, []byte{STOP, STOP, STOP}, [32]byte{1})

	f := newTestFrame(nil, 100000)
	f.State = db
	f.Stack.Push(uint256.NewInt(0))
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	if err := opExtCodeCopy(f); err != nil {
		t.Fatal(err)
	}

	f2 := newTestFrame(nil, 100000)
	f2.State = db
	f2.Memory.Expand(0, 32, &f2.Gas)
	f2.Stack.Push(uint256.NewInt(0))
	f2.Stack.PushRaw(0)
	f2.Stack.PushRaw(0)
	f2.Stack.PushRaw(2)
	if err := opExtCodeCopy(f2); err != nil {
		t.Fatal(err)
	}
}

// op_env.go: opReturnDataCopy (full coverage)
func TestOpReturnDataCopy(t *testing.T) {
	f := newTestFrame(nil, 100000)
	f.ReturnData = []byte{0x42}
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	if err := opReturnDataCopy(f); err != nil {
		t.Fatal(err)
	}

	f2 := newTestFrame(nil, 100000)
	f2.ReturnData = []byte{0x42}
	f2.Stack.PushRaw(0)
	f2.Stack.PushRaw(16)
	f2.Stack.PushRaw(1)
	err := opReturnDataCopy(f2)
	if err != ErrReturnDataOOB {
		t.Fatalf("want ErrReturnDataOOB, got %v", err)
	}

	f3 := newTestFrame(nil, 100000)
	f3.ReturnData = []byte{0x42, 0x43}
	f3.Memory.Expand(0, 32, &f3.Gas)
	f3.Stack.PushRaw(0)
	f3.Stack.PushRaw(0)
	f3.Stack.PushRaw(2)
	if err := opReturnDataCopy(f3); err != nil {
		t.Fatal(err)
	}
}

// op_env.go: opExtCodeHash with empty address
func TestOpExtCodeHashEmpty(t *testing.T) {
	db := state.New()
	var addr Address
	addr[19] = 0xCC
	f := newTestFrame(nil, 100000)
	f.State = db
	var w uint256.Int
	addrToWord(&addr, &w)
	f.Stack.Push(&w)
	if err := opExtCodeHash(f); err != nil {
		t.Fatal(err)
	}
	top, _ := f.Stack.Peek()
	if !top.IsZero() {
		t.Fatal("expected zero for empty account")
	}
}

// op_log.go: opLog Pop errors
func TestOpLogPopErrors(t *testing.T) {
	f := newTestFrame(nil, 100000)
	if err := opLog(f, 0); err != stack.ErrStackUnderflow {
		t.Fatalf("want stack underflow, got %v", err)
	}

	f2 := newTestFrame(nil, 100000)
	f2.Stack.PushRaw(0)
	if err := opLog(f2, 0); err != stack.ErrStackUnderflow {
		t.Fatalf("want stack underflow, got %v", err)
	}

	f3 := newTestFrame(nil, 100000)
	f3.Stack.PushRaw(0)
	f3.Stack.PushRaw(0)
	if err := opLog(f3, 1); err != stack.ErrStackUnderflow {
		t.Fatalf("want stack underflow, got %v", err)
	}
}

// op_call.go: execCall depth limit
func TestExecCallDepthLimit(t *testing.T) {
	db := state.New()
	var addr Address
	addr[19] = 0xCC
	db.CreateAccount(addr)

	f := &Frame{
		Code:     []byte{CALL},
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: addr,
		Depth:    MaxCallDepth,
	}
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)

	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// op_call.go: execCreate depth limit
func TestExecCreateDepthLimit(t *testing.T) {
	db := state.New()
	var addr Address
	addr[19] = 0xCC
	db.CreateAccount(addr)

	f := &Frame{
		Code:     []byte{CREATE},
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: addr,
		Depth:    MaxCallDepth,
	}
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)

	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// op_call.go: execCall with inSize>0 and outSize>0
func TestExecCallInOutSize(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))
	db.CreateAccount(targetAddr)
	db.WarmAddress(targetAddr)

	handler := &mockCallHandler{
		call: func(f *Frame, ct CallType, addr Address, value, gasIn uint64, input []byte) ([]byte, uint64, error) {
			return []byte{0xAA, 0xBB, 0xCC}, gasIn - 100, nil
		},
	}

	code := []byte{
		PUSH1, 3,
		PUSH1, 0,
		PUSH1, 32,
		PUSH1, 0,
		PUSH1, 0,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		PUSH2, 0x03, 0xE8,
		CALL,
		POP,
		STOP,
	}
	f := &Frame{
		Code:     code,
		Gas:      1000000,
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

// op_call.go: execCall with value>0
func TestExecCallValueTransfer(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xEE
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))
	db.WarmAddress(targetAddr)

	handler := &mockCallHandler{
		call: func(f *Frame, ct CallType, addr Address, value, gasIn uint64, input []byte) ([]byte, uint64, error) {
			return nil, gasIn - 100, nil
		},
	}

	code := []byte{
		PUSH1, 0,
		PUSH1, 0,
		PUSH1, 0,
		PUSH1, 0,
		PUSH1, 1,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xEE,
		PUSH2, 0x03, 0xE8,
		CALL,
		POP,
		STOP,
	}
	f := &Frame{
		Code:     code,
		Gas:      1000000,
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

// op_call.go: execCall value>0 NewAccount OOG
func TestExecCallValueNewAccountOOG(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xEE
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))

	code := []byte{
		PUSH1, 0,
		PUSH1, 0,
		PUSH1, 0,
		PUSH1, 0,
		PUSH1, 1,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xEE,
		PUSH2, 0x03, 0xE8,
		CALL,
		POP,
		STOP,
	}
	f := &Frame{
		Code:     code,
		Gas:      11500,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != ErrOutOfGas {
		t.Fatalf("want ErrOutOfGas, got %v", r.Err)
	}
}

// op_call.go: execCall cold OOG
func TestExecCallColdOOG(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))
	db.CreateAccount(targetAddr)

	code := []byte{
		PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		PUSH2, 0x03, 0xE8,
		CALL,
		POP,
		STOP,
	}
	f := &Frame{
		Code:     code,
		Gas:      2599,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != ErrOutOfGas {
		t.Fatalf("want ErrOutOfGas, got %v", r.Err)
	}
}

// op_call.go: execCreate static check
func TestExecCreateStatic(t *testing.T) {
	db := state.New()
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)

	code := []byte{PUSH1, 0, PUSH1, 0, PUSH1, 0, CREATE}
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
		t.Fatalf("want ErrWriteProtection, got %v", r.Err)
	}
}

// op_call.go: execCreate with size>0 (inSize > 0 path)
func TestExecCreateWithSize(t *testing.T) {
	db := state.New()
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))

	handler := &mockCallHandler{
		create: func(f *Frame, value uint64, initCode []byte, salt []byte) (Address, uint64, error) {
			var addr Address
			addr[19] = 0xEE
			return addr, f.Gas, nil
		},
	}

	code := []byte{PUSH1, 0, PUSH1, 0, PUSH1, 32, CREATE}
	f := &Frame{
		Code:     code,
		Gas:      1000000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	dummy := uint64(1000000)
	f.Memory.Expand(0, 32, &dummy)
	for i := 0; i < 32; i++ {
		f.Memory.SetByte(uint64(i), byte(i))
	}

	r := New(handler).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// op_call.go: execCreate2
func TestExecCreate2(t *testing.T) {
	db := state.New()
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))

	handler := &mockCallHandler{
		create: func(f *Frame, value uint64, initCode []byte, salt []byte) (Address, uint64, error) {
			var addr Address
			addr[19] = 0xEE
			return addr, f.Gas, nil
		},
	}

	code := []byte{PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0, CREATE2}
	f := &Frame{
		Code:     code,
		Gas:      1000000,
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

// op_call.go: execCall gasArg < forwardGas
func TestExecCallGasArgLess(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))
	db.CreateAccount(targetAddr)
	db.WarmAddress(targetAddr)

	handler := &mockCallHandler{
		call: func(f *Frame, ct CallType, addr Address, value, gasIn uint64, input []byte) ([]byte, uint64, error) {
			return nil, gasIn, nil
		},
	}

	code := []byte{
		PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		PUSH1, 10,
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
}

// pool.go: ReleaseFrame with large code/input cap
func TestReleaseFrameLargeCap(t *testing.T) {
	f := AcquireFrame()
	f.Code = make([]byte, 0, 66000)
	f.Input = make([]byte, 0, 66000)
	ReleaseFrame(f)
	f2 := AcquireFrame()
	ReleaseFrame(f2)
}

// Run: stack underflow error propagation through Run loop
func TestRunStackUnderflowPropagation(t *testing.T) {
	r := runCode(t, []byte{ADD}, 100000)
	if r.Err == nil {
		t.Fatal("expected error from ADD with empty stack")
	}
}

// Run: chargeExtAccess warm path with EXTCODESIZE
func TestRunExtCodeSizeWarm(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.CreateAccount(targetAddr)
	db.SetCode(targetAddr, []byte{STOP}, [32]byte{1})
	db.WarmAddress(targetAddr)

	code := []byte{
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		EXTCODESIZE,
		POP,
		STOP,
	}
	f := &Frame{
		Code:     code,
		Gas:      200,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// Run: EXTCODECOPY through the interpreter
func TestRunExtCodeCopy(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.CreateAccount(targetAddr)
	db.SetCode(targetAddr, []byte{STOP, STOP}, [32]byte{1})
	db.WarmAddress(targetAddr)

	code := []byte{
		PUSH1, 2,
		PUSH1, 0,
		PUSH1, 0,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		EXTCODECOPY,
		PUSH1, 32,
		PUSH1, 0,
		RETURN,
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
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// Run: RETURNDATACOPY through the interpreter
func TestRunReturnDataCopy(t *testing.T) {
	db := state.New()
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)

	code := []byte{PUSH1, 2, PUSH1, 0, PUSH1, 0, RETURNDATACOPY, STOP}
	f := &Frame{
		Code:         code,
		Gas:          100000,
		State:        db,
		Memory:       memory.New(),
		Block:        &BlockContext{},
		Tx:           &TxContext{},
		Contract:     contractAddr,
		ReturnData:   []byte{0x42, 0x43},
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// Run: CALLDATACOPY with size=0 via Run (covers memCopyOp size==0 + opCalldataCopy sz==0)
func TestRunCalldataCopyZeroSize(t *testing.T) {
	code := []byte{PUSH1, 0, PUSH1, 0, PUSH1, 0, CALLDATACOPY, STOP}
	f := newTestFrame(code, 100000)
	f.Input = []byte{1, 2, 3}
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// Run: CODECOPY with size=0 via Run
func TestRunCodeCopyZeroSize(t *testing.T) {
	code := []byte{PUSH1, 0, PUSH1, 0, PUSH1, 0, CODECOPY, STOP}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// Run: SSTORE with cold slot (covers cold path in main loop)
func TestRunSstoreColdSlot(t *testing.T) {
	db := state.New()
	var addr Address
	addr[19] = 0xCC
	db.CreateAccount(addr)

	code := []byte{PUSH1, 42, PUSH1, 0, SSTORE, STOP}
	f := &Frame{
		Code:     code,
		Gas:      100000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: addr,
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	slot := [32]byte{}
	val := db.GetStorage(addr, slot)
	if val[31] != 42 {
		t.Fatalf("want 42, got %d", val[31])
	}
}

// Run: EXTCODEHASH with empty account (through interpreter)
func TestRunExtCodeHashEmpty(t *testing.T) {
	db := state.New()
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)

	code := []byte{
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xEE,
		EXTCODEHASH,
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
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// Run: BALANCE with warm address (warm path through chargeExtAccess)
func TestRunBalanceWarm(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.CreateAccount(targetAddr)
	db.SetBalance(targetAddr, uint256.NewInt(500))
	db.WarmAddress(targetAddr)

	code := []byte{
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		BALANCE,
		POP,
		STOP,
	}
	f := &Frame{
		Code:     code,
		Gas:      200,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// Run: MLOAD with expandMem error (insufficient gas for memory expansion)
func TestRunMloadOutOfGas(t *testing.T) {
	code := []byte{PUSH1, 0, MLOAD}
	r := runCode(t, code, 1)
	if r.Err != ErrOutOfGas {
		t.Fatalf("want ErrOutOfGas, got %v", r.Err)
	}
}

// op_arithmetic.go: opSar with shift > 255 (positive number => clear)
func TestOpSarShiftAbove255Positive(t *testing.T) {
	code := []byte{PUSH1, 1, PUSH2, 1, 0, SAR, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.ReturnData[31] != 0 {
		t.Fatalf("want 0, got %d", r.ReturnData[31])
	}
}

// op_arithmetic.go: opSar with shift > 255 (negative number => all ones)
func TestOpSarShiftAbove255Negative(t *testing.T) {
	code := []byte{
		PUSH32,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		PUSH2, 1, 0,
		SAR,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	for i := 0; i < 32; i++ {
		if r.ReturnData[i] != 0xff {
			t.Fatalf("byte %d: want 0xff, got %x", i, r.ReturnData[i])
		}
	}
}

// Run: SLOAD cold access
func TestRunSloadCold(t *testing.T) {
	db := state.New()
	var addr Address
	addr[19] = 0xCC
	db.CreateAccount(addr)

	code := []byte{PUSH1, 0, SLOAD, POP, STOP}
	f := &Frame{
		Code:     code,
		Gas:      10000,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: addr,
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// Run: EXP with non-zero BitLen
func TestRunExpWithNonZeroBitLen(t *testing.T) {
	code := []byte{PUSH1, 3, PUSH1, 2, EXP, POP, STOP}
	r := runCode(t, code, 100000)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// Run: BALANCE cold access
func TestRunBalanceCold(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.CreateAccount(targetAddr)
	db.SetBalance(targetAddr, uint256.NewInt(500))

	code := []byte{
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		BALANCE,
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
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// Run: EXTCODEHASH warm
func TestRunExtCodeHashWarm(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.CreateAccount(targetAddr)
	db.SetCode(targetAddr, []byte{STOP}, [32]byte{1})
	db.WarmAddress(targetAddr)

	code := []byte{
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		EXTCODEHASH,
		POP,
		STOP,
	}
	f := &Frame{
		Code:     code,
		Gas:      200,
		State:    db,
		Memory:   memory.New(),
		Block:    &BlockContext{},
		Tx:       &TxContext{},
		Contract: contractAddr,
	}
	r := New(&noopHandler{}).Run(f)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
}

// Run: DELEGATECALL through interpreter
func TestRunDelegateCall(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))
	db.CreateAccount(targetAddr)
	db.SetCode(targetAddr, []byte{STOP}, [32]byte{1})
	db.WarmAddress(targetAddr)

	handler := &mockCallHandler{
		call: func(f *Frame, ct CallType, addr Address, value, gasIn uint64, input []byte) ([]byte, uint64, error) {
			return nil, gasIn - 100, nil
		},
	}

	code := []byte{
		PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		PUSH1, 0,
		DELEGATECALL,
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
}

// Run: STATICCALL through interpreter
func TestRunStaticCall(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))
	db.CreateAccount(targetAddr)
	db.SetCode(targetAddr, []byte{STOP}, [32]byte{1})
	db.WarmAddress(targetAddr)

	handler := &mockCallHandler{
		call: func(f *Frame, ct CallType, addr Address, value, gasIn uint64, input []byte) ([]byte, uint64, error) {
			return nil, gasIn - 100, nil
		},
	}

	code := []byte{
		PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		PUSH1, 0,
		STATICCALL,
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
}

// op_env.go: opCalldataCopy Pop error paths
func TestOpCalldataCopyPopErrors(t *testing.T) {
	f := newTestFrame(nil, 100000)
	f.Input = []byte{1, 2, 3}
	if err := opCalldataCopy(f); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow on empty stack")
	}
	f2 := newTestFrame(nil, 100000)
	f2.Input = []byte{1, 2, 3}
	f2.Stack.PushRaw(0)
	if err := opCalldataCopy(f2); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow with 1 item")
	}
	f3 := newTestFrame(nil, 100000)
	f3.Input = []byte{1, 2, 3}
	f3.Stack.PushRaw(0)
	f3.Stack.PushRaw(0)
	if err := opCalldataCopy(f3); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow with 2 items")
	}
}

// op_env.go: opCodeCopy Pop error paths
func TestOpCodeCopyPopErrors(t *testing.T) {
	f := newTestFrame([]byte{STOP}, 100000)
	if err := opCodeCopy(f); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow on empty stack")
	}
	f2 := newTestFrame([]byte{STOP}, 100000)
	f2.Stack.PushRaw(0)
	if err := opCodeCopy(f2); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow with 1 item")
	}
	f3 := newTestFrame([]byte{STOP}, 100000)
	f3.Stack.PushRaw(0)
	f3.Stack.PushRaw(0)
	if err := opCodeCopy(f3); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow with 2 items")
	}
}

// op_env.go: opExtCodeCopy Pop error paths
func TestOpExtCodeCopyPopErrors(t *testing.T) {
	db := state.New()
	var addr Address
	addr[19] = 0xCC
	db.CreateAccount(addr)

	f := newTestFrame(nil, 100000)
	f.State = db
	if err := opExtCodeCopy(f); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow on empty stack")
	}
	f2 := newTestFrame(nil, 100000)
	f2.State = db
	f2.Stack.Push(uint256.NewInt(0))
	if err := opExtCodeCopy(f2); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow with 1 item")
	}
	f3 := newTestFrame(nil, 100000)
	f3.State = db
	f3.Stack.Push(uint256.NewInt(0))
	f3.Stack.PushRaw(0)
	if err := opExtCodeCopy(f3); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow with 2 items")
	}
	f4 := newTestFrame(nil, 100000)
	f4.State = db
	f4.Stack.Push(uint256.NewInt(0))
	f4.Stack.PushRaw(0)
	f4.Stack.PushRaw(0)
	if err := opExtCodeCopy(f4); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow with 3 items")
	}
}

// op_env.go: opReturnDataCopy Pop error paths
func TestOpReturnDataCopyPopErrors(t *testing.T) {
	f := newTestFrame(nil, 100000)
	f.ReturnData = []byte{0x42}
	if err := opReturnDataCopy(f); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow on empty stack")
	}
	f2 := newTestFrame(nil, 100000)
	f2.ReturnData = []byte{0x42}
	f2.Stack.PushRaw(0)
	if err := opReturnDataCopy(f2); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow with 1 item")
	}
	f3 := newTestFrame(nil, 100000)
	f3.ReturnData = []byte{0x42}
	f3.Stack.PushRaw(0)
	f3.Stack.PushRaw(0)
	if err := opReturnDataCopy(f3); err != stack.ErrStackUnderflow {
		t.Fatal("want stack underflow with 2 items")
	}
}

// op_memory.go: opSstore static check (direct call)
func TestOpSstoreStatic(t *testing.T) {
	f := newTestFrame(nil, 100000)
	f.Static = true
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	if err := opSstore(f); err != ErrWriteProtection {
		t.Fatalf("want ErrWriteProtection, got %v", err)
	}
}

// op_memory.go: opJumpi with cond true and invalid dest
func TestOpJumpiInvalidDest(t *testing.T) {
	code := []byte{PUSH1, 3, PUSH1, 1, JUMPI, STOP}
	f := newTestFrame(code, 100000)
	f.jumpdests = buildJumpdests(f.Code)
	r := New(&noopHandler{}).Run(f)
	if r.Err != ErrInvalidJump {
		t.Fatalf("want ErrInvalidJump, got %v", r.Err)
	}
}

// op_log.go: opLog topic Pop error for n>1
func TestOpLogMultipleTopicPopErrors(t *testing.T) {
	f := newTestFrame(nil, 100000)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	f.Stack.PushRaw(0)
	if err := opLog(f, 2); err != stack.ErrStackUnderflow {
		t.Fatalf("LOG2: want stack underflow with 3 items, got %v", err)
	}
}

// Run: CALLCODE through interpreter
func TestRunCallCode(t *testing.T) {
	db := state.New()
	var contractAddr, targetAddr Address
	contractAddr[19] = 0xCC
	targetAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.SetBalance(contractAddr, uint256.NewInt(1000000))
	db.CreateAccount(targetAddr)
	db.SetCode(targetAddr, []byte{STOP}, [32]byte{1})
	db.WarmAddress(targetAddr)

	handler := &mockCallHandler{
		call: func(f *Frame, ct CallType, addr Address, value, gasIn uint64, input []byte) ([]byte, uint64, error) {
			return nil, gasIn - 100, nil
		},
	}

	code := []byte{
		PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0, PUSH1, 0,
		PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		PUSH1, 0,
		CALLCODE,
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
}

func TestOpSDivZero(t *testing.T) {
	f := newTestFrame(nil, 0)
	f.Stack.Push(uint256.NewInt(0))
	f.Stack.Push(uint256.NewInt(5))
	if err := opSDiv(f); err != nil { t.Fatal(err) }
	top, _ := f.Stack.Peek()
	if !top.IsZero() { t.Fatal("opSDiv(5,0) should be 0") }
}

func TestOpSModZero(t *testing.T) {
	f := newTestFrame(nil, 0)
	f.Stack.Push(uint256.NewInt(0))
	f.Stack.Push(uint256.NewInt(5))
	if err := opSMod(f); err != nil { t.Fatal(err) }
	top, _ := f.Stack.Peek()
	if !top.IsZero() { t.Fatal("opSMod(5,0) should be 0") }
}

func TestOpSltFalse(t *testing.T) {
	f := newTestFrame(nil, 0)
	f.Stack.Push(uint256.NewInt(3))
	f.Stack.Push(uint256.NewInt(5))
	if err := opSlt(f); err != nil { t.Fatal(err) }
	top, _ := f.Stack.Peek()
	if !top.IsZero() { t.Fatal("opSlt(5,3) should be 0") }
}

func TestOpSgtFalse(t *testing.T) {
	f := newTestFrame(nil, 0)
	f.Stack.Push(uint256.NewInt(5))
	f.Stack.Push(uint256.NewInt(3))
	if err := opSgt(f); err != nil { t.Fatal(err) }
	top, _ := f.Stack.Peek()
	if !top.IsZero() { t.Fatal("opSgt(3,5) should be 0 (3 not > 5)") }
}

func TestOpShlLargeShift(t *testing.T) {
	f := newTestFrame(nil, 0)
	f.Stack.Push(uint256.NewInt(1))
	f.Stack.Push(uint256.NewInt(256))
	if err := opShl(f); err != nil { t.Fatal(err) }
	top, _ := f.Stack.Peek()
	if !top.IsZero() { t.Fatal("opShl(x,256) should be 0") }
}

func TestOpShrLargeShift(t *testing.T) {
	f := newTestFrame(nil, 0)
	f.Stack.Push(uint256.NewInt(1))
	f.Stack.Push(uint256.NewInt(256))
	if err := opShr(f); err != nil { t.Fatal(err) }
	top, _ := f.Stack.Peek()
	if !top.IsZero() { t.Fatal("opShr(x,256) should be 0") }
}
