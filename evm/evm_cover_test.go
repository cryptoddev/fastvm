package evm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cryptoddev/fastvm/interpreter"
	"github.com/cryptoddev/fastvm/tracer"
	"github.com/holiman/uint256"
)

func mockRPC(t *testing.T, handler func(method string, params []interface{}) interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqs []struct {
			Method string        `json:"method"`
			Params []interface{} `json:"params"`
			ID     int           `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		type resp struct {
			JSONRPC string      `json:"jsonrpc"`
			ID      int         `json:"id"`
			Result  interface{} `json:"result"`
		}
		var responses []resp
		for _, req := range reqs {
			responses = append(responses, resp{JSONRPC: "2.0", ID: req.ID, Result: handler(req.Method, req.Params)})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responses)
	}))
}

func TestNewFromFork(t *testing.T) {
	srv := mockRPC(t, func(method string, params []interface{}) interface{} {
		switch method {
		case "eth_getBalance":
			return "0x64"
		case "eth_getTransactionCount":
			return "0x0"
		case "eth_getCode":
			return "0x"
		}
		return "0x0"
	})
	defer srv.Close()

	cfg := Config{ChainID: 1, GasLimit: 30_000_000}
	vm, _ := NewFromFork(srv.URL, 0, cfg)

	var from, to Address
	from[19] = 0xAA
	to[19] = 0xBB

	result := vm.Execute(Message{
		From:  from,
		To:    &to,
		Gas:   21000,
		Value: uint256.NewInt(100),
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestExecCallPrecompile(t *testing.T) {
	vm, db := newTestEVM()

	var from Address
	from[19] = 0xAA
	db.SetBalance(from, uint256.NewInt(1_000_000))

	var precompileAddr Address
	precompileAddr[19] = 1

	result := vm.Execute(Message{
		From:  from,
		To:    &precompileAddr,
		Gas:   100000,
		Value: uint256.NewInt(0),
		Data:  make([]byte, 128),
	})
	if result.Err != nil {
		t.Fatalf("precompile call failed: %v", result.Err)
	}
}

func TestExecCallWithCallTracer(t *testing.T) {
	vm, db := newTestEVM()

	var caller, contractAddr Address
	caller[19] = 1
	contractAddr[19] = 0xCC
	db.SetBalance(caller, uint256.NewInt(1_000_000))
	db.CreateAccount(contractAddr)
	db.SetCode(contractAddr, []byte{interpreter.STOP}, keccak256Hash([]byte{interpreter.STOP}))

	tr := tracer.New()
	result := vm.ExecuteWithCallTrace(Message{
		From:  caller,
		To:    &contractAddr,
		Gas:   100000,
		Value: uint256.NewInt(0),
	}, tr)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestCreateWithValue(t *testing.T) {
	vm, db := newTestEVM()

	var deployer Address
	deployer[19] = 1
	db.SetBalance(deployer, uint256.NewInt(1_000_000))

	f := interpreter.AcquireFrame()
	f.Contract = deployer
	f.Caller = deployer
	f.State = db
	f.Gas = 1_000_000
	f.Depth = 0

	initCode := []byte{
		interpreter.PUSH1, 0x42,
		interpreter.PUSH1, 0,
		interpreter.MSTORE,
		interpreter.PUSH1, 1,
		interpreter.PUSH1, 31,
		interpreter.RETURN,
	}

	addr, gasLeft, err := vm.Create(f, 500, initCode, nil)
	if err != nil {
		t.Fatalf("create with value failed: %v", err)
	}
	if addr == (Address{}) {
		t.Fatal("expected non-zero address")
	}
	if gasLeft == 0 {
		t.Fatal("expected gas left after create")
	}
	code := db.GetCode(addr)
	if len(code) != 1 || code[0] != 0x42 {
		t.Fatalf("unexpected deployed code: %x", code)
	}
	interpreter.ReleaseFrame(f)
}

func TestCreateWithSalt(t *testing.T) {
	vm, db := newTestEVM()

	var deployer Address
	deployer[19] = 1
	db.SetBalance(deployer, uint256.NewInt(1_000_000))

	f := interpreter.AcquireFrame()
	f.Contract = deployer
	f.Caller = deployer
	f.State = db
	f.Gas = 1_000_000
	f.Depth = 0

	initCode := []byte{
		interpreter.PUSH1, 0x42,
		interpreter.PUSH1, 0,
		interpreter.MSTORE,
		interpreter.PUSH1, 1,
		interpreter.PUSH1, 31,
		interpreter.RETURN,
	}

	var salt [32]byte
	salt[31] = 1

	addr, _, err := vm.Create(f, 0, initCode, salt[:])
	if err != nil {
		t.Fatalf("create2 failed: %v", err)
	}
	if addr == (Address{}) {
		t.Fatal("expected non-zero address")
	}
	code := db.GetCode(addr)
	if len(code) != 1 || code[0] != 0x42 {
		t.Fatalf("unexpected deployed code: %x", code)
	}
	interpreter.ReleaseFrame(f)
}

func TestNewFrameNilValue(t *testing.T) {
	vm, _ := newTestEVM()

	var contract, caller Address
	contract[19] = 0xAA
	caller[19] = 0xBB

	msg := Message{From: caller, To: &contract, Gas: 1000}

	f := vm.newFrame([]byte{interpreter.STOP}, contract, caller, msg)
	if f.Value.IsZero() == false {
		t.Fatal("expected zero value when msg.Value is nil")
	}
	interpreter.ReleaseFrame(f)
}

func TestCallDepthLimitHit(t *testing.T) {
	vm, _ := newTestEVM()

	f := interpreter.AcquireFrame()
	f.Depth = interpreter.MaxCallDepth - 1
	f.State = vm.db
	f.Contract = Address{}
	f.Gas = 1000

	_, _, err := vm.Call(f, interpreter.CallTypeCall, Address{}, 0, 1000, nil)
	if err != interpreter.ErrDepthLimit {
		t.Fatalf("want ErrDepthLimit, got %v", err)
	}
	interpreter.ReleaseFrame(f)
}

func TestCreateDepthLimitHit(t *testing.T) {
	vm, _ := newTestEVM()

	f := interpreter.AcquireFrame()
	f.Depth = interpreter.MaxCallDepth - 1
	f.State = vm.db
	f.Contract = Address{}
	f.Gas = 1000

	_, _, err := vm.Create(f, 0, nil, nil)
	if err != interpreter.ErrDepthLimit {
		t.Fatalf("want ErrDepthLimit, got %v", err)
	}
	interpreter.ReleaseFrame(f)
}

func TestCallCodeValueTransfer(t *testing.T) {
	vm, db := newTestEVM()

	var caller, codeAddr Address
	caller[19] = 1
	codeAddr[19] = 0xCC
	db.SetBalance(caller, uint256.NewInt(1000))
	db.CreateAccount(codeAddr)
	db.SetCode(codeAddr, []byte{interpreter.STOP}, keccak256Hash([]byte{interpreter.STOP}))

	f := interpreter.AcquireFrame()
	f.Contract = caller
	f.Caller = caller
	f.State = db
	f.Gas = 100000
	f.Depth = 0

	_, _, err := vm.Call(f, interpreter.CallTypeCallCode, codeAddr, 100, 1000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	interpreter.ReleaseFrame(f)
}

func TestCallCodeBalanceCheck(t *testing.T) {
	vm, db := newTestEVM()

	var caller, codeAddr Address
	caller[19] = 1
	codeAddr[19] = 0xCC
	db.SetBalance(caller, uint256.NewInt(50))
	db.CreateAccount(codeAddr)
	db.SetCode(codeAddr, []byte{interpreter.STOP}, keccak256Hash([]byte{interpreter.STOP}))

	f := interpreter.AcquireFrame()
	f.Contract = caller
	f.Caller = caller
	f.State = db
	f.Gas = 100000
	f.Depth = 0

	_, _, err := vm.Call(f, interpreter.CallTypeCallCode, codeAddr, 100, 1000, nil)
	if err != interpreter.ErrInsufficientFunds {
		t.Fatalf("want ErrInsufficientFunds, got %v", err)
	}
	interpreter.ReleaseFrame(f)
}

func TestCallCodeWithError(t *testing.T) {
	vm, db := newTestEVM()

	var caller, contractAddr, codeAddr Address
	caller[19] = 1
	contractAddr[19] = 0xBB
	codeAddr[19] = 0xCC
	db.SetBalance(caller, uint256.NewInt(1000))
	db.CreateAccount(contractAddr)
	db.CreateAccount(codeAddr)
	db.SetCode(codeAddr, []byte{interpreter.INVALID}, keccak256Hash([]byte{interpreter.INVALID}))

	f := interpreter.AcquireFrame()
	f.Contract = caller
	f.Caller = caller
	f.State = db
	f.Gas = 100000
	f.Depth = 0

	_, _, err := vm.Call(f, interpreter.CallTypeCallCode, codeAddr, 0, 1000, nil)
	if err == nil {
		t.Fatal("expected error from INVALID opcode")
	}
	interpreter.ReleaseFrame(f)
}

func TestCallCodeWithTracer(t *testing.T) {
	vm, db := newTestEVM()

	var caller, codeAddr Address
	caller[19] = 1
	codeAddr[19] = 0xCC
	db.SetBalance(caller, uint256.NewInt(1000))
	db.CreateAccount(codeAddr)
	db.SetCode(codeAddr, []byte{interpreter.STOP}, keccak256Hash([]byte{interpreter.STOP}))

	vm.callTracer = tracer.New()

	f := interpreter.AcquireFrame()
	f.Contract = caller
	f.Caller = caller
	f.State = db
	f.Gas = 100000
	f.Depth = 0

	_, _, err := vm.Call(f, interpreter.CallTypeCallCode, codeAddr, 0, 1000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	interpreter.ReleaseFrame(f)
}

func TestDelegateCallWithTracer(t *testing.T) {
	vm, db := newTestEVM()

	var caller, codeAddr Address
	caller[19] = 1
	codeAddr[19] = 0xCC
	db.SetBalance(caller, uint256.NewInt(1000))
	db.CreateAccount(codeAddr)
	db.SetCode(codeAddr, []byte{interpreter.STOP}, keccak256Hash([]byte{interpreter.STOP}))

	vm.callTracer = tracer.New()

	f := interpreter.AcquireFrame()
	f.Contract = caller
	f.Caller = caller
	f.State = db
	f.Gas = 100000
	f.Depth = 0

	_, _, err := vm.Call(f, interpreter.CallTypeDelegateCall, codeAddr, 0, 1000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	interpreter.ReleaseFrame(f)
}

func TestExecCallDelegateCallTracer(t *testing.T) {
	vm, db := newTestEVM()

	var caller Address
	caller[19] = 1
	db.SetBalance(caller, uint256.NewInt(1_000_000))

	contractCode := []byte{interpreter.STOP}
	var contractAddr Address
	contractAddr[19] = 0xCC
	db.CreateAccount(contractAddr)
	db.SetCode(contractAddr, contractCode, keccak256Hash(contractCode))

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

	tr := tracer.New()
	result := vm.ExecuteWithCallTrace(Message{
		From:  caller,
		To:    &callerAddr,
		Gas:   100000,
		Value: uint256.NewInt(0),
	}, tr)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}
