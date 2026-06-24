package evm

import (
	"testing"

	"github.com/cryptoddev/fastvm/interpreter"
	"github.com/cryptoddev/fastvm/state"
	"github.com/holiman/uint256"
)

func newTestEVM() (*EVM, *state.StateDB) {
	db := state.New()
	cfg := Config{ChainID: 1, GasLimit: 30_000_000}
	vm := New(db, cfg)
	return vm, db
}

func TestSimpleTransfer(t *testing.T) {
	vm, db := newTestEVM()

	var from, to Address
	from[19] = 0xAA
	to[19] = 0xBB

	db.SetBalance(from, uint256.NewInt(1_000_000))

	result := vm.Execute(Message{
		From:  from,
		To:    &to,
		Gas:   21000,
		Value: uint256.NewInt(100),
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if db.GetBalance(to).Uint64() != 100 {
		t.Fatalf("want to.balance=100, got %d", db.GetBalance(to).Uint64())
	}
	if db.GetBalance(from).Uint64() != 999900 {
		t.Fatalf("want from.balance=999900, got %d", db.GetBalance(from).Uint64())
	}
}

func TestContractDeploy(t *testing.T) {
	vm, db := newTestEVM()

	var deployer Address
	deployer[19] = 1
	db.SetBalance(deployer, uint256.NewInt(1_000_000))

	// Init code: PUSH1 0x42, PUSH1 0, MSTORE, PUSH1 1, PUSH1 31, RETURN
	// Deploys a contract whose code is a single byte 0x42.
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
	code := db.GetCode(*result.ContractAddress)
	if len(code) != 1 || code[0] != 0x42 {
		t.Fatalf("unexpected deployed code: %x", code)
	}
}

func TestContractCall(t *testing.T) {
	vm, db := newTestEVM()

	var caller Address
	caller[19] = 1
	db.SetBalance(caller, uint256.NewInt(1_000_000))

	// Contract code: PUSH1 0x99, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
	contractCode := []byte{
		interpreter.PUSH1, 0x99,
		interpreter.PUSH1, 0,
		interpreter.MSTORE,
		interpreter.PUSH1, 32,
		interpreter.PUSH1, 0,
		interpreter.RETURN,
	}
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

	result := vm.Execute(Message{
		From:  caller,
		To:    &contractAddr,
		Gas:   100_000,
		Value: uint256.NewInt(0),
	})
	if result.Err != nil {
		t.Fatalf("call error: %v", result.Err)
	}
	if len(result.ReturnData) != 32 {
		t.Fatalf("want 32 bytes return, got %d", len(result.ReturnData))
	}
	if result.ReturnData[31] != 0x99 {
		t.Fatalf("want 0x99, got %x", result.ReturnData[31])
	}
}

func TestRevertRestoresState(t *testing.T) {
	vm, db := newTestEVM()

	var caller Address
	caller[19] = 1
	db.SetBalance(caller, uint256.NewInt(1_000_000))

	// Contract that reverts after storing a value.
	// PUSH1 42, PUSH1 0, SSTORE, PUSH1 0, PUSH1 0, REVERT
	contractCode := []byte{
		interpreter.PUSH1, 42,
		interpreter.PUSH1, 0,
		interpreter.SSTORE,
		interpreter.PUSH1, 0,
		interpreter.PUSH1, 0,
		interpreter.REVERT,
	}
	var contractAddr Address
	contractAddr[19] = 0xDD
	db.CreateAccount(contractAddr)
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

	result := vm.Execute(Message{
		From:  caller,
		To:    &contractAddr,
		Gas:   100_000,
		Value: uint256.NewInt(0),
	})
	if result.Err != interpreter.ErrRevert {
		t.Fatalf("want ErrRevert, got %v", result.Err)
	}
	// Storage should be unchanged after revert.
	var slot [32]byte
	val := db.GetStorage(contractAddr, slot)
	if val != ([32]byte{}) {
		t.Fatalf("storage should be zero after revert, got %x", val)
	}
}


