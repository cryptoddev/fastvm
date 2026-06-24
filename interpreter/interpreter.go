package interpreter

import (
	"sync"
	"unsafe"

	"github.com/cryptoddev/fastvm/gas"
	"github.com/cryptoddev/fastvm/memory"
)

type CallHandler interface {
	Call(f *Frame, callType CallType, addr Address, value, gasIn uint64, input []byte) (ret []byte, gasLeft uint64, err error)
	Create(f *Frame, value uint64, initCode []byte, salt []byte) (addr Address, gasLeft uint64, err error)
}

type Interpreter struct {
	handler       CallHandler
	jumpdestCache sync.Map
}

func New(handler CallHandler) *Interpreter {
	return &Interpreter{handler: handler}
}

func (interp *Interpreter) Run(f *Frame) ExecResult {
	if f.Memory == nil {
		f.Memory = memory.New()
	}
	// Use cached jumpdest bitmap keyed by code pointer identity.
	// Same backing array + length reuses the bitmap across calls.
	codeKey := codeID(f.Code)
	if v, ok := interp.jumpdestCache.Load(codeKey); ok {
		f.jumpdests = v.(bitvec)
	} else {
		bv := buildJumpdests(f.Code)
		interp.jumpdestCache.Store(codeKey, bv)
		f.jumpdests = bv
	}

	for {
		if f.PC >= uint64(len(f.Code)) {
			return ExecResult{GasLeft: f.Gas}
		}

		op := f.Code[f.PC]
		f.PC++

		var (
			err       error
			gasBefore uint64
		)
		if f.Tracer != nil {
			gasBefore = f.Gas
		}
		switch op {
		case STOP:
			return ExecResult{GasLeft: f.Gas}
		case ADD:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opAdd(f)
		case MUL:
			if f.Gas < gas.Low { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Low; err = opMul(f)
		case SUB:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opSub(f)
		case DIV:
			if f.Gas < gas.Low { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Low; err = opDiv(f)
		case SDIV:
			if f.Gas < gas.Low { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Low; err = opSDiv(f)
		case MOD:
			if f.Gas < gas.Low { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Low; err = opMod(f)
		case SMOD:
			if f.Gas < gas.Low { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Low; err = opSMod(f)
		case ADDMOD:
			if f.Gas < gas.Mid { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Mid; err = opAddMod(f)
		case MULMOD:
			if f.Gas < gas.Mid { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Mid; err = opMulMod(f)
		case EXP:
			expBytes := uint64((f.Stack.Back(1).BitLen() + 7) / 8)
			dynGas := gas.Exp + gas.ExpByte*expBytes
			if f.Gas < dynGas { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= dynGas; err = opExp(f)
		case SIGNEXTEND:
			if f.Gas < gas.Low { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Low; err = opSignExtend(f)
		case LT:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opLt(f)
		case GT:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opGt(f)
		case SLT:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opSlt(f)
		case SGT:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opSgt(f)
		case EQ:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opEq(f)
		case ISZERO:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opIsZero(f)
		case AND:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opAnd(f)
		case OR:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opOr(f)
		case XOR:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opXor(f)
		case NOT:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opNot(f)
		case BYTE:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opByte(f)
		case SHL:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opShl(f)
		case SHR:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opShr(f)
		case SAR:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opSar(f)
		case SHA3:
			off, sz := f.Stack.Back(0).Uint64(), f.Stack.Back(1).Uint64()
			memCost, _ := f.Memory.ExpansionCost(off, sz)
			dynGas := gas.Sha3 + memCost + gas.Sha3Words(sz)
			if f.Gas < dynGas { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= dynGas
			_ = f.Memory.Expand(off, sz, &f.Gas); err = opSha3(f)
		case ADDRESS:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opAddress(f)
		case BALANCE:
			if err = chargeExtAccess(f, wordToAddr(f.Stack.Back(0))); err == nil { err = opBalance(f) }
		case ORIGIN:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opOrigin(f)
		case CALLER:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opCaller(f)
		case CALLVALUE:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opCallValue(f)
		case CALLDATALOAD:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow; err = opCalldataLoad(f)
		case CALLDATASIZE:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opCalldataSize(f)
		case CALLDATACOPY:
			err = memCopyOp(f, func() error { return opCalldataCopy(f) })
		case CODESIZE:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opCodeSize(f)
		case CODECOPY:
			err = memCopyOp(f, func() error { return opCodeCopy(f) })
		case GASPRICE:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opGasPrice(f)
		case EXTCODESIZE:
			if err = chargeExtAccess(f, wordToAddr(f.Stack.Back(0))); err == nil { err = opExtCodeSize(f) }
		case EXTCODECOPY:
			if err = chargeExtAccess(f, wordToAddr(f.Stack.Back(0))); err == nil {
				err = memCopyOp(f, func() error { return opExtCodeCopy(f) })
			}
		case RETURNDATASIZE:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opReturnDataSize(f)
		case RETURNDATACOPY:
			err = memCopyOp(f, func() error { return opReturnDataCopy(f) })
		case EXTCODEHASH:
			if err = chargeExtAccess(f, wordToAddr(f.Stack.Back(0))); err == nil { err = opExtCodeHash(f) }
		case BLOCKHASH:
			if f.Gas < gas.BlockHash { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.BlockHash; err = opBlockHash(f)
		case COINBASE:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opCoinbase(f)
		case TIMESTAMP:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opTimestamp(f)
		case NUMBER:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opNumber(f)
		case DIFFICULTY:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opDifficulty(f)
		case GASLIMIT:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opGasLimit(f)
		case CHAINID:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opChainID(f)
		case SELFBALANCE:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opSelfBalance(f)
		case BASEFEE:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opBaseFee(f)
		case POP:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opPop(f)
		case MLOAD:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow
			if err = expandMem(f, f.Stack.Back(0).Uint64(), 32); err == nil { err = opMload(f) }
		case MSTORE:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow
			if err = expandMem(f, f.Stack.Back(0).Uint64(), 32); err == nil { err = opMstore(f) }
		case MSTORE8:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow
			if err = expandMem(f, f.Stack.Back(0).Uint64(), 1); err == nil { err = opMstore8(f) }
		case SLOAD:
			slot := f.Stack.Back(0).Bytes32()
			if !f.State.SlotWarm(f.Contract, slot) {
				if f.Gas < gas.ColdSload { return ExecResult{Err: ErrOutOfGas} }
				f.Gas -= gas.ColdSload; f.State.WarmSlot(f.Contract, slot)
			} else {
				if f.Gas < gas.WarmStorageRead { return ExecResult{Err: ErrOutOfGas} }
				f.Gas -= gas.WarmStorageRead
			}
			err = opSload(f)
		case SSTORE:
			if f.Static { return ExecResult{Err: ErrWriteProtection} }
			if f.Gas <= 2300 { return ExecResult{Err: ErrOutOfGas} }
			slot, newVal := f.Stack.Back(0).Bytes32(), f.Stack.Back(1).Bytes32()
			current := f.State.GetStorage(f.Contract, slot)
			original := f.State.GetCommittedStorage(f.Contract, slot)
			if !f.State.SlotWarm(f.Contract, slot) {
				if f.Gas < gas.ColdSload { return ExecResult{Err: ErrOutOfGas} }
				f.Gas -= gas.ColdSload; f.State.WarmSlot(f.Contract, slot)
			}
			cost, refund := gas.SstoreCost(current, original, newVal, true)
			if f.Gas < cost { return ExecResult{Err: ErrOutOfGas} }
			f.Gas -= cost; f.State.AddRefund(refund); err = opSstore(f)
		case JUMP:
			if f.Gas < gas.Mid { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Mid; err = opJump(f)
		case JUMPI:
			if f.Gas < gas.High { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.High; err = opJumpi(f)
		case PC:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opPC(f)
		case MSIZE:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opMsize(f)
		case GAS:
			if f.Gas < gas.Base { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.Base; err = opGas(f)
		case JUMPDEST:
			if f.Gas < gas.JumpDest { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.JumpDest
		case PUSH1, PUSH2, PUSH3, PUSH4, PUSH5, PUSH6, PUSH7, PUSH8,
			PUSH9, PUSH10, PUSH11, PUSH12, PUSH13, PUSH14, PUSH15, PUSH16,
			PUSH17, PUSH18, PUSH19, PUSH20, PUSH21, PUSH22, PUSH23, PUSH24,
			PUSH25, PUSH26, PUSH27, PUSH28, PUSH29, PUSH30, PUSH31, PUSH32:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow
			err = opPush(f, int(op-PUSH1)+1)
		case DUP1, DUP2, DUP3, DUP4, DUP5, DUP6, DUP7, DUP8,
			DUP9, DUP10, DUP11, DUP12, DUP13, DUP14, DUP15, DUP16:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow
			err = opDup(f, int(op-DUP1)+1)
		case SWAP1, SWAP2, SWAP3, SWAP4, SWAP5, SWAP6, SWAP7, SWAP8,
			SWAP9, SWAP10, SWAP11, SWAP12, SWAP13, SWAP14, SWAP15, SWAP16:
			if f.Gas < gas.VeryLow { return ExecResult{Err: ErrOutOfGas} }; f.Gas -= gas.VeryLow
			err = opSwap(f, int(op-SWAP1)+1)
		case LOG0, LOG1, LOG2, LOG3, LOG4:
			err = logOp(f, int(op-LOG0))
		case RETURN:
			off, sz := f.Stack.Back(0).Uint64(), f.Stack.Back(1).Uint64()
			if sz > 0 { if e := expandMem(f, off, sz); e != nil { return ExecResult{Err: e} } }
			f.Stack.Pop(); f.Stack.Pop()
			return ExecResult{ReturnData: f.Memory.GetCopy(off, sz), GasLeft: f.Gas}
		case REVERT:
			off, sz := f.Stack.Back(0).Uint64(), f.Stack.Back(1).Uint64()
			if sz > 0 { if e := expandMem(f, off, sz); e != nil { return ExecResult{Err: e} } }
			f.Stack.Pop(); f.Stack.Pop()
			return ExecResult{ReturnData: f.Memory.GetCopy(off, sz), GasLeft: f.Gas, Err: ErrRevert}
		case INVALID:
			return ExecResult{Err: ErrInvalidOpcode}
		case SELFDESTRUCT:
			if f.Static { return ExecResult{Err: ErrWriteProtection} }
			if f.Gas < gas.SelfDestruct { return ExecResult{Err: ErrOutOfGas} }
			f.Gas -= gas.SelfDestruct
			ben, serr := f.Stack.Pop()
			if serr != nil { return ExecResult{Err: serr} }
			benAddr := wordToAddr(ben)
			if !f.State.AddressWarm(benAddr) {
				if f.Gas < gas.ColdAccountAccess { return ExecResult{Err: ErrOutOfGas} }
				f.Gas -= gas.ColdAccountAccess
			}
			if !f.State.Exists(benAddr) {
				if f.Gas < gas.SelfDestructNewAccount { return ExecResult{Err: ErrOutOfGas} }
				f.Gas -= gas.SelfDestructNewAccount
			}
			if bal := f.State.GetBalance(f.Contract); !bal.IsZero() {
				f.State.AddBalance(benAddr, bal); f.State.SubBalance(f.Contract, bal)
			}
			// EIP-6780: only self-destruct if contract was created in this tx.
			if f.State.CreatedInTx(f.Contract) {
				f.State.Suicide(f.Contract)
			}
			return ExecResult{GasLeft: f.Gas}
		case CALL, CALLCODE, DELEGATECALL, STATICCALL:
			r := interp.execCall(f, op)
			if r.Err != nil && r.Err != ErrRevert { return r }
			err = r.Err
		case CREATE, CREATE2:
			r := interp.execCreate(f, op)
			if r.Err != nil && r.Err != ErrRevert { return r }
			err = r.Err
		default:
			return ExecResult{Err: ErrInvalidOpcode}
		}

		if f.Tracer != nil {
			f.Tracer.Capture(f.PC-1, op, gasBefore, f.Gas, f.Depth, f.Stack.Data(), f.Stack.Len(), err)
		}
		if err != nil {
			if err == ErrRevert { return ExecResult{Err: ErrRevert, GasLeft: f.Gas} }
			return ExecResult{Err: err}
		}
	}
}

var ErrRevert = errRevert("revert")

type errRevert string

func (e errRevert) Error() string { return string(e) }

func expandMem(f *Frame, offset, size uint64) error {
	if size == 0 {
		return nil
	}
	return f.Memory.Expand(offset, size, &f.Gas)
}

func memCopyOp(f *Frame, fn func() error) error {
	// Stack: ..., destOffset, offset, size (top)
	// We peek ahead and charge expansion+copier gas before fn pops them.
	dest := f.Stack.Back(2).Uint64()
	size := f.Stack.Back(0).Uint64()
	if size > 0 {
		if err := expandMem(f, dest, size); err != nil {
			return err
		}
		wordCost := gas.CopyWords(size)
		if f.Gas < wordCost {
			return ErrOutOfGas
		}
		f.Gas -= wordCost
	}
	return fn()
}

func logOp(f *Frame, n int) error {
	if f.Static {
		return ErrWriteProtection
	}
	offset := f.Stack.Back(0).Uint64()
	size := f.Stack.Back(1).Uint64()
	if size > 0 {
		if err := expandMem(f, offset, size); err != nil {
			return err
		}
	}
	logGas := gas.Log + gas.LogData*size + uint64(n)*gas.LogTopic
	if f.Gas < logGas {
		return ErrOutOfGas
	}
	f.Gas -= logGas
	return opLog(f, n)
}

func chargeExtAccess(f *Frame, addr Address) error {
	if !f.State.AddressWarm(addr) {
		if f.Gas < gas.ColdAccountAccess {
			return ErrOutOfGas
		}
		f.Gas -= gas.ColdAccountAccess
		f.State.WarmAddress(addr)
	} else {
		if f.Gas < gas.WarmStorageRead {
			return ErrOutOfGas
		}
		f.Gas -= gas.WarmStorageRead
	}
	return nil
}

func codeID(code []byte) [2]uintptr {
	if len(code) == 0 {
		return [2]uintptr{}
	}
	return [2]uintptr{uintptr(unsafe.Pointer(&code[0])), uintptr(len(code))}
}
