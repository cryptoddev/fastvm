package interpreter

import (
	"github.com/cryptoddev/fastvm/gas"
	"github.com/holiman/uint256"
)

func (interp *Interpreter) execCall(f *Frame, op byte) ExecResult {
	if f.Depth >= MaxCallDepth {
		f.Stack.PushRaw(0)
		return ExecResult{GasLeft: f.Gas}
	}

	var (
		gasArg    *uint256.Int
		addrWord  *uint256.Int
		valueWord *uint256.Int
		inOffset  *uint256.Int
		inSize    *uint256.Int
		outOffset *uint256.Int
		outSize   *uint256.Int
		err       error
	)

	gasArg, err = f.Stack.Pop()
	if err != nil {
		return ExecResult{Err: err}
	}
	addrWord, err = f.Stack.Pop()
	if err != nil {
		return ExecResult{Err: err}
	}

	hasValue := op == CALL || op == CALLCODE
	if hasValue {
		valueWord, err = f.Stack.Pop()
		if err != nil {
			return ExecResult{Err: err}
		}
	}

	inOffset, err = f.Stack.Pop()
	if err != nil {
		return ExecResult{Err: err}
	}
	inSize, err = f.Stack.Pop()
	if err != nil {
		return ExecResult{Err: err}
	}
	outOffset, err = f.Stack.Pop()
	if err != nil {
		return ExecResult{Err: err}
	}
	outSize, err = f.Stack.Pop()
	if err != nil {
		return ExecResult{Err: err}
	}

	if inSize.Uint64() > 0 {
		if merr := expandMem(f, inOffset.Uint64(), inSize.Uint64()); merr != nil {
			return ExecResult{Err: merr}
		}
	}
	if outSize.Uint64() > 0 {
		if merr := expandMem(f, outOffset.Uint64(), outSize.Uint64()); merr != nil {
			return ExecResult{Err: merr}
		}
	}

	addr := wordToAddr(addrWord)

	if !f.State.AddressWarm(addr) {
		if f.Gas < gas.ColdAccountAccess {
			return ExecResult{Err: ErrOutOfGas}
		}
		f.Gas -= gas.ColdAccountAccess
		f.State.WarmAddress(addr)
	} else {
		if f.Gas < gas.WarmStorageRead {
			return ExecResult{Err: ErrOutOfGas}
		}
		f.Gas -= gas.WarmStorageRead
	}

	var value uint64
	if hasValue && valueWord != nil {
		value = valueWord.Uint64()
		if value > 0 {
			if f.Gas < gas.CallValue {
				return ExecResult{Err: ErrOutOfGas}
			}
			f.Gas -= gas.CallValue
			isPrecompile := addr[0] == 0 && addr[1] == 0 && addr[2] == 0 && addr[3] == 0 &&
				addr[4] == 0 && addr[5] == 0 && addr[6] == 0 && addr[7] == 0 &&
				addr[8] == 0 && addr[9] == 0 && addr[10] == 0 && addr[11] == 0 &&
				addr[12] == 0 && addr[13] == 0 && addr[14] == 0 && addr[15] == 0 &&
				addr[16] == 0 && addr[17] == 0 && addr[18] == 0 &&
				addr[19] >= 1 && addr[19] <= 9
			if !isPrecompile && !f.State.Exists(addr) {
				if f.Gas < gas.NewAccount {
					return ExecResult{Err: ErrOutOfGas}
				}
				f.Gas -= gas.NewAccount
			}
		}
	}

	forwardGas := gas.CallGas(f.Gas)
	if gasArg.IsUint64() && gasArg.Uint64() < forwardGas {
		forwardGas = gasArg.Uint64()
	}
	stipend := uint64(0)
	if hasValue && value > 0 {
		stipend = gas.CallStipend
	}
	f.Gas -= forwardGas
	forwardGas += stipend

	callStatic := f.Static || op == STATICCALL

	var input []byte
	if inSize.Uint64() > 0 {
		input = f.Memory.GetCopy(inOffset.Uint64(), inSize.Uint64())
	}

	var ct CallType
	switch op {
	case CALL:
		ct = CallTypeCall
	case CALLCODE:
		ct = CallTypeCallCode
	case DELEGATECALL:
		ct = CallTypeDelegateCall
	case STATICCALL:
		ct = CallTypeStaticCall
	}
	_ = callStatic

	ret, gasLeft, callErr := interp.handler.Call(f, ct, addr, value, forwardGas, input)
	if gasLeft > forwardGas-stipend {
		gasLeft = forwardGas - stipend
	}
	f.Gas += gasLeft
	f.ReturnData = ret

	if outSize.Uint64() > 0 && len(ret) > 0 {
		n := outSize.Uint64()
		if uint64(len(ret)) < n {
			n = uint64(len(ret))
		}
		dst := f.Memory.GetPtr(outOffset.Uint64(), outSize.Uint64())
		copy(dst, ret[:n])
	}

	if callErr != nil {
		f.Stack.PushRaw(0)
	} else {
		f.Stack.PushRaw(1)
	}

	return ExecResult{GasLeft: f.Gas}
}

func (interp *Interpreter) execCreate(f *Frame, op byte) ExecResult {
	if f.Static {
		return ExecResult{Err: ErrWriteProtection}
	}
	if f.Depth >= MaxCallDepth {
		f.Stack.PushRaw(0)
		return ExecResult{GasLeft: f.Gas}
	}

	valueWord, err := f.Stack.Pop()
	if err != nil {
		return ExecResult{Err: err}
	}
	offset, err := f.Stack.Pop()
	if err != nil {
		return ExecResult{Err: err}
	}
	size, err := f.Stack.Pop()
	if err != nil {
		return ExecResult{Err: err}
	}

	var salt []byte
	if op == CREATE2 {
		saltWord, serr := f.Stack.Pop()
		if serr != nil {
			return ExecResult{Err: serr}
		}
		b := saltWord.Bytes32()
		salt = b[:]
	}

	off := offset.Uint64()
	sz := size.Uint64()
	if sz > 0 {
		if merr := expandMem(f, off, sz); merr != nil {
			return ExecResult{Err: merr}
		}
	}

	initCode := f.Memory.GetCopy(off, sz)
	value := valueWord.Uint64()

	forwardGas := gas.CallGas(f.Gas)
	f.Gas -= forwardGas

	addr, gasLeft, createErr := interp.handler.Create(f, value, initCode, salt)
	f.Gas += gasLeft

	if createErr != nil {
		f.Stack.PushRaw(0)
	} else {
		var w uint256.Int
		addrToWord(&addr, &w)
		f.Stack.Push(&w)
	}

	return ExecResult{GasLeft: f.Gas}
}
