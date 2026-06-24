package tracer_test

import (
	"fmt"
	"testing"

	"github.com/cryptoddev/fastvm/evm"
	"github.com/cryptoddev/fastvm/interpreter"
	"github.com/cryptoddev/fastvm/state"
	"github.com/cryptoddev/fastvm/tracer"
	"github.com/holiman/uint256"
)

// TestNestedCalls demonstrates cast-style output with nested CALL.
//
// Contract A calls Contract B, which returns 0x99.
// Output should look like:
//
//	[gasUsed] 0000..00AA::CALL()
//	  ├─ [gasUsed] 0000..00BB::CALL()
//	  │   └─ ← 0x99
//	  └─ ← 0x99
func TestNestedCalls(t *testing.T) {
	db := state.New()
	vm := evm.New(db, evm.Config{ChainID: 1, GasLimit: 30_000_000})

	var caller, contractA, contractB evm.Address
	caller[19] = 0x01
	contractA[19] = 0xAA
	contractB[19] = 0xBB

	db.SetBalance(caller, uint256.NewInt(1e18))

	// Contract B: returns 0x99 in 32 bytes
	// PUSH1 0x99, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
	codeB := []byte{
		interpreter.PUSH1, 0x99,
		interpreter.PUSH1, 0,
		interpreter.MSTORE,
		interpreter.PUSH1, 32,
		interpreter.PUSH1, 0,
		interpreter.RETURN,
	}
	db.CreateAccount(contractB)
	db.SetCode(contractB, codeB, [32]byte{1})

	// Contract A: calls Contract B, copies return data, returns it
	// We use a simpler approach: A just calls B with CALL and returns B's output.
	//
	// Bytecode:
	//   PUSH1 32   retSize
	//   PUSH1 0    retOffset
	//   PUSH1 0    argsSize
	//   PUSH1 0    argsOffset
	//   PUSH1 0    value
	//   PUSH20 contractB
	//   PUSH3 0x0F4240  gas (1000000)
	//   CALL
	//   PUSH1 32   size
	//   PUSH1 0    offset
	//   RETURN
	addrB := contractB
	codeA := []byte{
		interpreter.PUSH1, 32,   // retSize
		interpreter.PUSH1, 0,    // retOffset
		interpreter.PUSH1, 0,    // argsSize
		interpreter.PUSH1, 0,    // argsOffset
		interpreter.PUSH1, 0,    // value
		interpreter.PUSH20,
		addrB[0], addrB[1], addrB[2], addrB[3], addrB[4],
		addrB[5], addrB[6], addrB[7], addrB[8], addrB[9],
		addrB[10], addrB[11], addrB[12], addrB[13], addrB[14],
		addrB[15], addrB[16], addrB[17], addrB[18], addrB[19],
		interpreter.PUSH3, 0x0F, 0x42, 0x40, // gas = 1_000_000
		interpreter.CALL,
		interpreter.POP,          // pop success flag
		interpreter.PUSH1, 32,
		interpreter.PUSH1, 0,
		interpreter.RETURN,
	}
	db.CreateAccount(contractA)
	db.SetCode(contractA, codeA, [32]byte{2})

	tr := tracer.New()
	result := vm.ExecuteWithCallTrace(evm.Message{
		From:  caller,
		To:    &contractA,
		Gas:   2_000_000,
		Value: uint256.NewInt(0),
	}, tr)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	output := tr.String()
	fmt.Println(output)

	// Verify structure: root has 1 child (call to B)
	if tr.Root == nil {
		t.Fatal("no root")
	}
	if len(tr.Root.Children) != 1 {
		t.Fatalf("expected 1 child call, got %d", len(tr.Root.Children))
	}
	child := tr.Root.Children[0]
	if child.To != contractB {
		t.Fatalf("child should call contractB")
	}
}
