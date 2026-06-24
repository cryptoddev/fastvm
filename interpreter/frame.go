package interpreter

import (
	"github.com/cryptoddev/fastvm/memory"
	"github.com/cryptoddev/fastvm/stack"
	"github.com/cryptoddev/fastvm/state"
	"github.com/holiman/uint256"
)

type Stack = stack.Stack

type Address = state.Address
type Hash = state.Hash

type BlockContext struct {
	Coinbase    Address
	BlockNumber uint64
	Time        uint64
	Difficulty  uint256.Int
	GasLimit    uint64
	BaseFee     uint256.Int
	ChainID     uint256.Int
}

type TxContext struct {
	Origin   Address
	GasPrice uint256.Int
}

type CallType uint8

const (
	CallTypeCall CallType = iota
	CallTypeCallCode
	CallTypeDelegateCall
	CallTypeStaticCall
	CallTypeCreate
	CallTypeCreate2
)

type StateDB interface {
	GetBalance(Address) *uint256.Int
	GetNonce(Address) uint64
	GetCode(Address) []byte
	GetCodeHash(Address) Hash
	GetStorage(Address, Hash) Hash
	GetCommittedStorage(Address, Hash) Hash
	SetStorage(Address, Hash, Hash)
	AddBalance(Address, *uint256.Int)
	SubBalance(Address, *uint256.Int)
	Exists(Address) bool
	Empty(Address) bool
	Suicide(Address)
	AddLog(*state.Log)
	AddRefund(uint64)
	SubRefund(uint64)
	AddressWarm(Address) bool
	WarmAddress(Address)
	SlotWarm(Address, Hash) bool
	WarmSlot(Address, Hash)
	CreatedInTx(Address) bool
}

type TracerIface interface {
	Enabled() bool
	Capture(pc uint64, op byte, gasBefore, gasAfter uint64, depth int, stack []uint256.Int, stackLen int, err error)
}

type CallTracerIface interface {
	Enabled() bool
	OnEnter(ct CallType, from, to Address, input []byte, gas uint64, value *uint256.Int)
	OnExit(output []byte, gasUsed uint64, err error)
}

type Frame struct {
	Code       []byte
	PC         uint64
	Gas        uint64
	Stack      Stack
	Memory     *memory.Memory

	CallType   CallType
	Caller     Address
	Contract   Address
	Value      uint256.Int
	Input      []byte

	ReturnData []byte

	Static     bool

	Depth      int

	State      StateDB

	Block      *BlockContext
	Tx         *TxContext

	jumpdests  bitvec

	Tracer interface {
		Enabled() bool
		Capture(pc uint64, op byte, gasBefore, gasAfter uint64, depth int, stack []uint256.Int, stackLen int, err error)
	}
}

type bitvec []byte

func (b bitvec) set(i uint64) {
	b[i/8] |= 1 << (i % 8)
}

func (b bitvec) has(i uint64) bool {
	if i/8 >= uint64(len(b)) {
		return false
	}
	return b[i/8]&(1<<(i%8)) != 0
}

func buildJumpdests(code []byte) bitvec {
	bv := make(bitvec, (len(code)+7)/8)
	for i := 0; i < len(code); {
		op := code[i]
		if op == JUMPDEST {
			bv.set(uint64(i))
		}
		if op >= PUSH1 && op <= PUSH32 {
			i += int(op-PUSH1) + 2
		} else {
			i++
		}
	}
	return bv
}

func (f *Frame) Reset() {
	f.PC = 0
	f.Stack.Reset()
	f.Memory.Reset()
	f.ReturnData = f.ReturnData[:0]
	f.Static = false
	f.jumpdests = nil
}
