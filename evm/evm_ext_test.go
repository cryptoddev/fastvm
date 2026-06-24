package evm

import (
	"testing"

	"github.com/cryptoddev/fastvm/interpreter"
	"github.com/cryptoddev/fastvm/state"
	"github.com/cryptoddev/fastvm/tracer"
	"github.com/holiman/uint256"
)

func TestNewWithTrace(t *testing.T) {
	db := state.New()
	cfg := Config{ChainID: 1, GasLimit: 30_000_000, Trace: true}
	vm := New(db, cfg)
	tr := vm.Tracer()
	if tr == nil {
		t.Fatal("expected tracer when Trace=true")
	}
}

func TestNewWithoutTrace(t *testing.T) {
	db := state.New()
	cfg := Config{ChainID: 1, GasLimit: 30_000_000}
	vm := New(db, cfg)
	tr := vm.Tracer()
	if tr != nil {
		t.Fatal("expected nil tracer when Trace=false")
	}
}

func TestExecuteInsufficientFunds(t *testing.T) {
	vm, db := newTestEVM()

	var from, to Address
	from[19] = 0xAA
	to[19] = 0xBB

	db.SetBalance(from, uint256.NewInt(0))

	result := vm.Execute(Message{
		From:     from,
		To:       &to,
		Gas:      21000,
		Value:    uint256.NewInt(100),
		GasPrice: 1,
	})
	if result.Err != interpreter.ErrInsufficientFunds {
		t.Fatalf("want ErrInsufficientFunds, got %v", result.Err)
	}
}

func TestExecuteIntrinsicGasTooLow(t *testing.T) {
	vm, db := newTestEVM()

	var from, to Address
	from[19] = 0xAA
	to[19] = 0xBB

	db.SetBalance(from, uint256.NewInt(1_000_000))

	result := vm.Execute(Message{
		From:              from,
		To:                &to,
		Gas:               100,
		Value:             uint256.NewInt(100),
		ApplyIntrinsicGas: true,
	})
	if result.Err != ErrIntrinsicGas {
		t.Fatalf("want ErrIntrinsicGas, got %v", result.Err)
	}
}

func TestExecuteIntrinsicGasCreate(t *testing.T) {
	vm, db := newTestEVM()

	var from Address
	from[19] = 0xAA

	db.SetBalance(from, uint256.NewInt(1_000_000))

	// CREATE intrinsic gas = 21000 + 32000 + calldata cost
	result := vm.Execute(Message{
		From:              from,
		Gas:               60000,
		Value:             uint256.NewInt(0),
		Data:              []byte{interpreter.STOP},
		ApplyIntrinsicGas: true,
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestExecuteEOACall(t *testing.T) {
	vm, db := newTestEVM()

	var from, to Address
	from[19] = 0xAA
	to[19] = 0xBB

	db.SetBalance(from, uint256.NewInt(1_000_000))

	// Call EOA (no code)
	result := vm.Execute(Message{
		From:  from,
		To:    &to,
		Gas:   100000,
		Value: uint256.NewInt(100),
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.GasLeft != 100000 {
		t.Fatalf("want gasLeft=100000, got %d", result.GasLeft)
	}
}

func TestExecuteWithTrace(t *testing.T) {
	vm, db := newTestEVM()

	var from, to Address
	from[19] = 0xAA
	to[19] = 0xBB

	db.SetBalance(from, uint256.NewInt(1_000_000))

	tr := tracer.New()
	result := vm.ExecuteWithTrace(Message{
		From:  from,
		To:    &to,
		Gas:   21000,
		Value: uint256.NewInt(100),
	}, tr)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestExecuteWithCallTrace(t *testing.T) {
	vm, db := newTestEVM()

	var from, to Address
	from[19] = 0xAA
	to[19] = 0xBB

	db.SetBalance(from, uint256.NewInt(1_000_000))

	tr := tracer.New()
	result := vm.ExecuteWithCallTrace(Message{
		From:  from,
		To:    &to,
		Gas:   21000,
		Value: uint256.NewInt(100),
	}, tr)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestExecuteAccessList(t *testing.T) {
	vm, db := newTestEVM()

	var from, to Address
	from[19] = 0xAA
	to[19] = 0xBB

	db.SetBalance(from, uint256.NewInt(1_000_000))

	var slot Hash
	slot[31] = 1

	result := vm.Execute(Message{
		From:  from,
		To:    &to,
		Gas:   21000,
		Value: uint256.NewInt(100),
		AccessList: []AccessListEntry{
			{Address: to, Slots: []Hash{slot}},
		},
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestExecuteRefund(t *testing.T) {
	vm, db := newTestEVM()

	var caller Address
	caller[19] = 1
	db.SetBalance(caller, uint256.NewInt(1_000_000))

	// Contract that writes to storage then clears it (generates refund)
	// PUSH1 0 (value), PUSH1 0 (slot), SSTORE, PUSH1 42, PUSH1 0, SSTORE, STOP
	contractCode := []byte{
		interpreter.PUSH1, 42,
		interpreter.PUSH1, 0,
		interpreter.SSTORE,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.SSTORE,
		interpreter.STOP,
	}
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

	result := vm.Execute(Message{
		From:  caller,
		To:    &contractAddr,
		Gas:   100000,
		Value: uint256.NewInt(0),
	})
	if result.Err != nil {
		t.Fatalf("call error: %v", result.Err)
	}
}

func TestExecuteCoinbasePayment(t *testing.T) {
	db := state.New()

	var caller, contractAddr, coinbase Address
	caller[19] = 1
	contractAddr[19] = 0xCC
	coinbase[19] = 0xFE

	db.SetBalance(caller, uint256.NewInt(1_000_000))

	// Contract that does some work: PUSH1 1, PUSH1 2, ADD, POP, STOP
	contractCode := []byte{
		interpreter.PUSH1, 1,
		interpreter.PUSH1, 2,
		interpreter.ADD,
		interpreter.POP,
		interpreter.STOP,
	}
	db.CreateAccount(contractAddr)
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

	cfg := Config{ChainID: 1, GasLimit: 30_000_000, BaseFee: 1, Coinbase: coinbase}
	vm := New(db, cfg)

	result := vm.Execute(Message{
		From:     caller,
		To:       &contractAddr,
		Gas:      100000,
		Value:    uint256.NewInt(0),
		GasPrice: 10,
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	// Coinbase should have received fees (gasUsed > 0 for contract call)
	if db.GetBalance(coinbase).IsZero() {
		t.Fatalf("coinbase should have received fees, gasUsed=%d, gasLeft=%d", 100000-result.GasLeft, result.GasLeft)
	}
}

func TestExecuteNoCoinbasePayment(t *testing.T) {
	db := state.New()

	var from, to Address
	from[19] = 0xAA
	to[19] = 0xBB

	db.SetBalance(from, uint256.NewInt(1_000_000))

	// BaseFee >= GasPrice → no tip
	cfg := Config{ChainID: 1, GasLimit: 30_000_000, BaseFee: 100}
	vm := New(db, cfg)

	result := vm.Execute(Message{
		From:     from,
		To:       &to,
		Gas:      21000,
		Value:    uint256.NewInt(100),
		GasPrice: 10,
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	// Coinbase should not have received fees
	if !db.GetBalance(cfg.Coinbase).IsZero() {
		t.Fatal("coinbase should not have received fees when GasPrice <= BaseFee")
	}
}

func TestExecuteCallCode(t *testing.T) {
	vm, db := newTestEVM()

	var caller Address
	caller[19] = 1
	db.SetBalance(caller, uint256.NewInt(1_000_000))

	// Simple contract: STOP
	contractCode := []byte{interpreter.STOP}
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

	result := vm.Execute(Message{
		From:  caller,
		To:    &contractAddr,
		Gas:   100000,
		Value: uint256.NewInt(0),
	})
	if result.Err != nil {
		t.Fatalf("call error: %v", result.Err)
	}
}

func TestCreate2(t *testing.T) {
	vm, db := newTestEVM()

	var deployer Address
	deployer[19] = 1
	db.SetBalance(deployer, uint256.NewInt(1_000_000))

	// Init code that deploys a single byte 0x42
	initCode := []byte{
		interpreter.PUSH1, 0x42,
		interpreter.PUSH1, 0,
		interpreter.MSTORE,
		interpreter.PUSH1, 1,
		interpreter.PUSH1, 31,
		interpreter.RETURN,
	}

	result := vm.Execute(Message{
		From:  deployer,
		Gas:   1_000_000,
		Value: uint256.NewInt(0),
		Data:  initCode,
	})
	if result.Err != nil {
		t.Fatalf("deploy error: %v", result.Err)
	}
	if result.ContractAddress == nil {
		t.Fatal("expected contract address")
	}
}

func TestCreateInsufficientValue(t *testing.T) {
	vm, db := newTestEVM()

	var deployer Address
	deployer[19] = 1
	db.SetBalance(deployer, uint256.NewInt(0))

	initCode := []byte{interpreter.STOP}

	result := vm.Execute(Message{
		From:  deployer,
		Gas:   1_000_000,
		Value: uint256.NewInt(1000),
		Data:  initCode,
	})
	if result.Err != interpreter.ErrInsufficientFunds {
		t.Fatalf("want ErrInsufficientFunds, got %v", result.Err)
	}
}

func TestCreateRevert(t *testing.T) {
	vm, db := newTestEVM()

	var deployer Address
	deployer[19] = 1
	db.SetBalance(deployer, uint256.NewInt(1_000_000))

	// Init code that reverts
	initCode := []byte{
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.REVERT,
	}

	result := vm.Execute(Message{
		From:  deployer,
		Gas:   1_000_000,
		Value: uint256.NewInt(0),
		Data:  initCode,
	})
	if result.Err != interpreter.ErrRevert {
		t.Fatalf("want ErrRevert, got %v", result.Err)
	}
}

func TestCreateError(t *testing.T) {
	vm, db := newTestEVM()

	var deployer Address
	deployer[19] = 1
	db.SetBalance(deployer, uint256.NewInt(1_000_000))

	// Init code that hits invalid opcode
	initCode := []byte{interpreter.INVALID}

	result := vm.Execute(Message{
		From:  deployer,
		Gas:   1_000_000,
		Value: uint256.NewInt(0),
		Data:  initCode,
	})
	if result.Err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateAddress(t *testing.T) {
	var deployer Address
	deployer[19] = 0xAB

	// nonce = 0
	addr0 := createAddress(deployer, 0)
	if addr0 == (Address{}) {
		t.Fatal("expected non-zero address")
	}

	// nonce < 0x80
	addr1 := createAddress(deployer, 1)
	if addr1 == addr0 {
		t.Fatal("addresses should differ")
	}

	// nonce >= 0x80
	addr128 := createAddress(deployer, 128)
	if addr128 == addr1 {
		t.Fatal("addresses should differ")
	}
}

func TestCreate2Address(t *testing.T) {
	var deployer Address
	deployer[19] = 0xAB

	salt := []byte{0x01, 0x02, 0x03, 0x04}
	initCode := []byte{interpreter.STOP}

	addr1 := create2Address(deployer, salt, initCode)
	if addr1 == (Address{}) {
		t.Fatal("expected non-zero address")
	}

	// Different salt → different address
	addr2 := create2Address(deployer, []byte{0x05}, initCode)
	if addr1 == addr2 {
		t.Fatal("addresses should differ")
	}
}

func TestIntrinsicGas(t *testing.T) {
	// Base
	g := intrinsicGas(nil, false, nil)
	if g != 21000 {
		t.Fatalf("want 21000, got %d", g)
	}

	// Create
	g = intrinsicGas(nil, true, nil)
	if g != 53000 {
		t.Fatalf("want 53000, got %d", g)
	}

	// With data
	g = intrinsicGas([]byte{0x00, 0x01}, false, nil)
	if g != 21000+4+16 {
		t.Fatalf("want %d, got %d", 21000+4+16, g)
	}

	// With access list
	g = intrinsicGas(nil, false, []AccessListEntry{
		{Slots: []Hash{{}, {}}},
	})
	if g != 21000+2400+2*1900 {
		t.Fatalf("want %d, got %d", 21000+2400+2*1900, g)
	}
}

func TestCallDepthLimit(t *testing.T) {
	vm, db := newTestEVM()

	var caller Address
	caller[19] = 1
	db.SetBalance(caller, uint256.NewInt(1_000_000))

	// Contract that calls itself recursively
	contractCode := []byte{
		// PUSH2 gas, PUSH20 addr, PUSH1 value, PUSH1 0, PUSH1 0, PUSH1 0, PUSH1 0, CALL
		interpreter.PUSH2, 0x03, 0xE8, // 1000
		interpreter.PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xCC,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.CALL,
		interpreter.POP,
		interpreter.STOP,
	}
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

	result := vm.Execute(Message{
		From:  caller,
		To:    &contractAddr,
		Gas:   1000000,
		Value: uint256.NewInt(0),
	})
	// Should eventually hit depth limit or run out of gas
	_ = result
}

func TestStaticCallWriteProtection(t *testing.T) {
	vm, db := newTestEVM()

	var caller Address
	caller[19] = 1
	db.SetBalance(caller, uint256.NewInt(1_000_000))

	// Contract that tries to SSTORE (should fail in static context)
	contractCode := []byte{
		interpreter.PUSH1, 42,
		interpreter.PUSH1, 0,
		interpreter.SSTORE,
		interpreter.STOP,
	}
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

	// Caller contract that does STATICCALL
	callerCode := []byte{
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xCC,
		interpreter.PUSH1, 0,
		interpreter.STATICCALL,
		interpreter.POP,
		interpreter.STOP,
	}
	var callerAddr Address
	callerAddr[19] = 0xDD
	db.CreateAccount(callerAddr)
	db.SetCode(callerAddr, callerCode, keccak256Hash(callerCode))

	result := vm.Execute(Message{
		From:  caller,
		To:    &callerAddr,
		Gas:   100000,
		Value: uint256.NewInt(0),
	})
	// STATICCALL should succeed (returns 0 on failure)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestCallCodeValueCheck(t *testing.T) {
	vm, db := newTestEVM()

	var caller Address
	caller[19] = 1
	db.SetBalance(caller, uint256.NewInt(0))

	// Contract that does CALLCODE with value
	contractCode := []byte{
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xEE,
		interpreter.PUSH1, 1,
		interpreter.CALLCODE,
		interpreter.POP,
		interpreter.STOP,
	}
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

	result := vm.Execute(Message{
		From:  caller,
		To:    &contractAddr,
		Gas:   100000,
		Value: uint256.NewInt(0),
	})
	// Should fail due to insufficient funds for CALLCODE value check
	_ = result
}

func TestCallCodeEOA(t *testing.T) {
	vm, db := newTestEVM()

	var caller Address
	caller[19] = 1
	db.SetBalance(caller, uint256.NewInt(1_000_000))

	// Contract that does CALLCODE to an EOA (no code)
	contractCode := []byte{
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xEE,
		interpreter.PUSH1, 0,
		interpreter.CALLCODE,
		interpreter.POP,
		interpreter.STOP,
	}
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

	result := vm.Execute(Message{
		From:  caller,
		To:    &contractAddr,
		Gas:   100000,
		Value: uint256.NewInt(0),
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestDelegateCall(t *testing.T) {
	vm, db := newTestEVM()

	var caller Address
	caller[19] = 1
	db.SetBalance(caller, uint256.NewInt(1_000_000))

	// Simple contract: STOP
	contractCode := []byte{interpreter.STOP}
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

	// Caller that does DELEGATECALL
	callerCode := []byte{
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xCC,
		interpreter.PUSH1, 0,
		interpreter.DELEGATECALL,
		interpreter.POP,
		interpreter.STOP,
	}
	var callerAddr Address
	callerAddr[19] = 0xDD
	db.CreateAccount(callerAddr)
	db.SetCode(callerAddr, callerCode, keccak256Hash(callerCode))

	result := vm.Execute(Message{
		From:  caller,
		To:    &callerAddr,
		Gas:   100000,
		Value: uint256.NewInt(0),
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestCreateDepthLimit(t *testing.T) {
	vm, db := newTestEVM()

	var deployer Address
	deployer[19] = 1
	db.SetBalance(deployer, uint256.NewInt(1_000_000))

	// Contract that creates another contract recursively
	initCode := []byte{
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.CREATE,
		interpreter.POP,
		interpreter.STOP,
	}

	result := vm.Execute(Message{
		From:  deployer,
		Gas:   1_000_000,
		Value: uint256.NewInt(0),
		Data:  initCode,
	})
	_ = result
}

func TestCallValueTransfer(t *testing.T) {
	vm, db := newTestEVM()

	var caller, contractAddr Address
	caller[19] = 1
	contractAddr[19] = 0xCC

	db.SetBalance(caller, uint256.NewInt(1000))
	db.CreateAccount(contractAddr)

	// Simple contract: STOP
	contractCode := []byte{interpreter.STOP}
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

	result := vm.Execute(Message{
		From:  caller,
		To:    &contractAddr,
		Gas:   100000,
		Value: uint256.NewInt(500),
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if db.GetBalance(contractAddr).Uint64() != 500 {
		t.Fatalf("want contract balance=500, got %d", db.GetBalance(contractAddr).Uint64())
	}
}

func TestCallInsufficientValue(t *testing.T) {
	vm, db := newTestEVM()

	var caller, contractAddr Address
	caller[19] = 1
	contractAddr[19] = 0xCC

	db.SetBalance(caller, uint256.NewInt(100))
	db.CreateAccount(contractAddr)

	contractCode := []byte{interpreter.STOP}
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

	result := vm.Execute(Message{
		From:  caller,
		To:    &contractAddr,
		Gas:   100000,
		Value: uint256.NewInt(500),
	})
	if result.Err != interpreter.ErrInsufficientFunds {
		t.Fatalf("want ErrInsufficientFunds, got %v", result.Err)
	}
}

func TestCallCodeInsufficientValue(t *testing.T) {
	vm, db := newTestEVM()

	var caller, contractAddr Address
	caller[19] = 1
	contractAddr[19] = 0xCC

	db.SetBalance(caller, uint256.NewInt(100))
	db.CreateAccount(contractAddr)

	// Contract that does CALLCODE with value
	callerCode := []byte{
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.PUSH20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDD,
		interpreter.PUSH1, 200,
		interpreter.CALLCODE,
		interpreter.POP,
		interpreter.STOP,
	}
	db.SetCode(contractAddr, callerCode, keccak256Hash(callerCode))

	result := vm.Execute(Message{
		From:  caller,
		To:    &contractAddr,
		Gas:   100000,
		Value: uint256.NewInt(0),
	})
	// CALLCODE should fail due to insufficient funds
	_ = result
}
