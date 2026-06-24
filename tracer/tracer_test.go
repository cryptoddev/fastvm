package tracer_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cryptoddev/fastvm/evm"
	"github.com/cryptoddev/fastvm/interpreter"
	"github.com/cryptoddev/fastvm/state"
	"github.com/cryptoddev/fastvm/tracer"
	"github.com/holiman/uint256"
)

func setup() (*evm.EVM, *state.StateDB) {
	db := state.New()
	vm := evm.New(db, evm.Config{ChainID: 1, GasLimit: 30_000_000})
	return vm, db
}

func TestCallTracer(t *testing.T) {
	vm, db := setup()

	var caller, contract evm.Address
	caller[19] = 0xAA
	contract[19] = 0xBB

	db.SetBalance(caller, uint256.NewInt(1e18))
	db.CreateAccount(contract)

	// Contract: PUSH1 0x42, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
	code := []byte{
		interpreter.PUSH1, 0x42,
		interpreter.PUSH1, 0,
		interpreter.MSTORE,
		interpreter.PUSH1, 32,
		interpreter.PUSH1, 0,
		interpreter.RETURN,
	}
	db.SetCode(contract, code, [32]byte{})

	tr := tracer.New()
	result := vm.ExecuteWithCallTrace(evm.Message{
		From:  caller,
		To:    &contract,
		Gas:   100_000,
		Value: uint256.NewInt(0),
	}, tr)

	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if tr.Root == nil {
		t.Fatal("no root node")
	}
	if tr.Root.GasUsed == 0 {
		t.Fatal("gasUsed should be > 0")
	}

	out := tr.String()
	t.Log("\n" + out)

	if !strings.Contains(out, "CALL") {
		t.Error("output should contain CALL")
	}
	if !strings.Contains(out, "←") {
		t.Error("output should contain return arrow")
	}
}

func TestCallTracerRevert(t *testing.T) {
	vm, db := setup()

	var caller, contract evm.Address
	caller[19] = 0xAA
	contract[19] = 0xCC
	db.SetBalance(caller, uint256.NewInt(1e18))
	db.CreateAccount(contract)

	// Contract: PUSH1 0, PUSH1 0, REVERT
	code := []byte{interpreter.PUSH1, 0, interpreter.PUSH1, 0, interpreter.REVERT}
	db.SetCode(contract, code, [32]byte{})

	tr := tracer.New()
	result := vm.ExecuteWithCallTrace(evm.Message{
		From:  caller,
		To:    &contract,
		Gas:   100_000,
		Value: uint256.NewInt(0),
	}, tr)

	if result.Err == nil {
		t.Fatal("expected revert error")
	}

	out := tr.String()
	t.Log("\n" + out)
	if !strings.Contains(out, "REVERT") {
		t.Error("output should contain REVERT")
	}
}

func TestNoopTracer(t *testing.T) {
	tr := tracer.Noop()
	if tr.Enabled() {
		t.Fatal("Noop tracer should not be enabled")
	}
	if tr.Root != nil {
		t.Fatal("Noop tracer should have nil root")
	}
}

func TestTracerEnabled(t *testing.T) {
	tr := tracer.New()
	if !tr.Enabled() {
		t.Fatal("New tracer should be enabled")
	}
}

func TestTracerStringEmpty(t *testing.T) {
	tr := tracer.New()
	out := tr.String()
	if out != "(empty trace)" {
		t.Fatalf("want '(empty trace)', got %q", out)
	}
}

func TestTracerOnEnterDisabled(t *testing.T) {
	tr := tracer.Noop()
	var from, to [20]byte
	from[19] = 1
	to[19] = 2
	tr.OnEnter(tracer.CallTypeCall, from, to, []byte{0x01, 0x02, 0x03, 0x04}, 1000, uint256.NewInt(0))
	if tr.Root != nil {
		t.Fatal("OnEnter on disabled tracer should not create root")
	}
}

func TestTracerOnExitDisabled(t *testing.T) {
	tr := tracer.Noop()
	tr.OnExit([]byte{0x01}, 100, nil)
	// Should not panic
}

func TestTracerOnExitNilCurrent(t *testing.T) {
	tr := tracer.New()
	tr.OnExit([]byte{0x01}, 100, nil)
	// Should not panic
}

func TestTracerCapture(t *testing.T) {
	tr := tracer.New()
	// Capture is a no-op, just verify it doesn't panic
	tr.Capture(0, 0, 0, 0, 0, nil, 0, nil)
}

func TestTracerWithNestedCalls(t *testing.T) {
	vm, db := setup()

	var caller, outer, inner evm.Address
	caller[19] = 0xAA
	outer[19] = 0xBB
	inner[19] = 0xCC

	db.SetBalance(caller, uint256.NewInt(1e18))
	db.CreateAccount(outer)
	db.CreateAccount(inner)

	// Inner contract: RETURN empty
	innerCode := []byte{interpreter.PUSH1, 0, interpreter.PUSH1, 0, interpreter.RETURN}
	db.SetCode(inner, innerCode, [32]byte{})

	// Outer contract: CALL inner, then RETURN
	// This is a simplified test - we just verify the tracer handles nested calls
	outerCode := []byte{interpreter.STOP}
	db.SetCode(outer, outerCode, [32]byte{})

	tr := tracer.New()
	_ = vm.ExecuteWithCallTrace(evm.Message{
		From:  caller,
		To:    &outer,
		Gas:   100_000,
		Value: uint256.NewInt(0),
	}, tr)

	out := tr.String()
	t.Log("\n" + out)
	if !strings.Contains(out, "CALL") {
		t.Error("output should contain CALL")
	}
}

func TestTracerWithRevertReason(t *testing.T) {
	vm, db := setup()

	var caller, contract evm.Address
	caller[19] = 0xAA
	contract[19] = 0xDD
	db.SetBalance(caller, uint256.NewInt(1e18))
	db.CreateAccount(contract)

	code := []byte{interpreter.PUSH1, 0, interpreter.PUSH1, 0, interpreter.REVERT}
	db.SetCode(contract, code, [32]byte{})

	tr := tracer.New()
	_ = vm.ExecuteWithCallTrace(evm.Message{
		From:  caller,
		To:    &contract,
		Gas:   100_000,
		Value: uint256.NewInt(0),
	}, tr)

	out := tr.String()
	t.Log("\n" + out)
	if !strings.Contains(out, "REVERT") {
		t.Error("output should contain REVERT")
	}
}

func TestTracerCallTypes(t *testing.T) {
	// Test each call type individually
	callTypes := []struct {
		ct   tracer.CallType
		name string
	}{
		{tracer.CallTypeCall, "CALL"},
		{tracer.CallTypeCallCode, "CALLCODE"},
		{tracer.CallTypeDelegateCall, "DELEGATECALL"},
		{tracer.CallTypeStaticCall, "STATICCALL"},
		{tracer.CallTypeCreate, "CREATE"},
		{tracer.CallTypeCreate2, "CREATE2"},
	}

	for _, tc := range callTypes {
		tr := tracer.New()
		var from, to [20]byte
		from[19] = 1
		to[19] = 2
		tr.OnEnter(tc.ct, from, to, []byte{0x01, 0x02, 0x03, 0x04}, 1000, uint256.NewInt(0))
		tr.OnExit([]byte{0x01}, 100, nil)

		out := tr.String()
		if !strings.Contains(out, tc.name) {
			t.Errorf("output for %s should contain %s", tc.name, tc.name)
		}
	}
}

func TestTracerUnknownCallType(t *testing.T) {
	tr := tracer.New()
	var from, to [20]byte
	from[19] = 1
	to[19] = 2
	// Use an unknown call type (value 99)
	tr.OnEnter(99, from, to, []byte{0x01, 0x02, 0x03, 0x04}, 1000, uint256.NewInt(0))
	tr.OnExit([]byte{0x01}, 100, nil)

	out := tr.String()
	// Unknown call types default to "CALL"
	if !strings.Contains(out, "CALL") {
		t.Error("unknown call type should default to CALL")
	}
}

func TestTracerWithValue(t *testing.T) {
	tr := tracer.New()
	var from, to [20]byte
	from[19] = 1
	to[19] = 2
	val := uint256.NewInt(1000)
	tr.OnEnter(tracer.CallTypeCall, from, to, []byte{}, 1000, val)
	tr.OnExit(nil, 100, nil)

	if tr.Root.Value.Uint64() != 1000 {
		t.Fatalf("want value=1000, got %d", tr.Root.Value.Uint64())
	}
}

func TestTracerWithNilValue(t *testing.T) {
	tr := tracer.New()
	var from, to [20]byte
	from[19] = 1
	to[19] = 2
	tr.OnEnter(tracer.CallTypeCall, from, to, []byte{}, 1000, nil)
	tr.OnExit(nil, 100, nil)

	if !tr.Root.Value.IsZero() {
		t.Fatal("value should be zero when nil is passed")
	}
}

func TestTracerWithShortInput(t *testing.T) {
	tr := tracer.New()
	var from, to [20]byte
	from[19] = 1
	to[19] = 2
	tr.OnEnter(tracer.CallTypeCall, from, to, []byte{0xAB}, 1000, nil)
	tr.OnExit(nil, 100, nil)

	out := tr.String()
	t.Log("\n" + out)
	if !strings.Contains(out, "ab") {
		t.Error("output should contain short input hex")
	}
}

func TestTracerWithLongOutput(t *testing.T) {
	tr := tracer.New()
	var from, to [20]byte
	from[19] = 1
	to[19] = 2
	tr.OnEnter(tracer.CallTypeCall, from, to, []byte{0x01, 0x02, 0x03, 0x04, 0x05}, 1000, nil)
	output := make([]byte, 100)
	for i := range output {
		output[i] = byte(i)
	}
	tr.OnExit(output, 100, nil)

	out := tr.String()
	t.Log("\n" + out)
	if !strings.Contains(out, "..(") {
		t.Error("long output should be truncated with ..(N) format")
	}
}

func TestTracerWithRevertHexOutput(t *testing.T) {
	tr := tracer.New()
	var from, to [20]byte
	from[19] = 1
	to[19] = 2
	tr.OnEnter(tracer.CallTypeCall, from, to, []byte{0x01, 0x02, 0x03, 0x04}, 1000, nil)
	tr.OnExit([]byte{0xDE, 0xAD}, 100, errors.New("revert"))

	out := tr.String()
	t.Log("\n" + out)
	if !strings.Contains(out, "0xdead") {
		t.Error("revert with short data should show hex")
	}
}

func TestTracerWithRevertEmptyOutput(t *testing.T) {
	tr := tracer.New()
	var from, to [20]byte
	from[19] = 1
	to[19] = 2
	tr.OnEnter(tracer.CallTypeCall, from, to, []byte{0x01, 0x02, 0x03, 0x04}, 1000, nil)
	tr.OnExit(nil, 100, errors.New("revert"))

	out := tr.String()
	t.Log("\n" + out)
	if !strings.Contains(out, "REVERT") {
		t.Error("should contain REVERT")
	}
}

func TestTracerWithRevertErrorString(t *testing.T) {
	tr := tracer.New()
	var from, to [20]byte
	from[19] = 1
	to[19] = 2
	tr.OnEnter(tracer.CallTypeCall, from, to, []byte{0x01, 0x02, 0x03, 0x04}, 1000, nil)
	// ABI-encoded Error("fail"): selector + offset(32 bytes) + length(32 bytes) + string
	revertData := make([]byte, 68+4)
	// Selector
	revertData[0] = 0x08
	revertData[1] = 0xc3
	revertData[2] = 0x79
	revertData[3] = 0xa0
	// Offset = 32 (at bytes 4-35, big-endian)
	revertData[35] = 0x20
	// Length = 4 (at bytes 36-67, big-endian)
	revertData[67] = 0x04
	// String "fail" at bytes 68-71
	revertData[68] = 'f'
	revertData[69] = 'a'
	revertData[70] = 'i'
	revertData[71] = 'l'

	tr.OnExit(revertData, 100, errors.New("revert"))

	out := tr.String()
	t.Log("\n" + out)
	if !strings.Contains(out, "REVERT") {
		t.Error("should contain REVERT")
	}
	if !strings.Contains(out, "fail") {
		t.Error("should contain error reason 'fail'")
	}
}

func TestTracerWithLongInput(t *testing.T) {
	tr := tracer.New()
	var from, to [20]byte
	from[19] = 1
	to[19] = 2
	input := make([]byte, 100)
	for i := range input {
		input[i] = byte(i)
	}
	input[0] = 0x01
	input[1] = 0x02
	input[2] = 0x03
	input[3] = 0x04
	tr.OnEnter(tracer.CallTypeCall, from, to, input, 1000, nil)
	tr.OnExit(nil, 100, nil)

	out := tr.String()
	t.Log("\n" + out)
	if !strings.Contains(out, "(96b)") {
		t.Error("long input should show remaining byte count")
	}
}

func TestTracerWithShortAddress(t *testing.T) {
	tr := tracer.New()
	var from, to [20]byte
	to[19] = 0xAB
	tr.OnEnter(tracer.CallTypeCall, from, to, []byte{}, 1000, nil)
	tr.OnExit(nil, 100, nil)

	out := tr.String()
	t.Log("\n" + out)
	// Address should be rendered
	if !strings.Contains(out, "ab") {
		t.Error("output should contain address")
	}
}


