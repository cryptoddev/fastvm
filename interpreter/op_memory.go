package interpreter

import (
	"github.com/holiman/uint256"
)

func opPop(f *Frame) error {
	_, err := f.Stack.Pop()
	return err
}

func opMload(f *Frame) error {
	x := f.Stack.Back(0)
	off := x.Uint64()
	data := f.Memory.Get(off, 32)
	x.SetBytes32(data)
	return nil
}

func opMstore(f *Frame) error {
	offset, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	val, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b32 := val.Bytes32()
	f.Memory.Set32(offset.Uint64(), b32[:])
	return nil
}

func opMstore8(f *Frame) error {
	offset, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	val, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	f.Memory.SetByte(offset.Uint64(), byte(val.Uint64()))
	return nil
}

func opSload(f *Frame) error {
	slot := f.Stack.Back(0)
	key := slot.Bytes32()
	val := f.State.GetStorage(f.Contract, key)
	slot.SetBytes32(val[:])
	return nil
}

func opSstore(f *Frame) error {
	if f.Static {
		return ErrWriteProtection
	}
	slot, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	val, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	key := slot.Bytes32()
	value := val.Bytes32()
	f.State.SetStorage(f.Contract, key, value)
	return nil
}

func opJump(f *Frame) error {
	dest, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	d := dest.Uint64()
	if !f.jumpdests.has(d) {
		return ErrInvalidJump
	}
	f.PC = d
	return nil
}

func opJumpi(f *Frame) error {
	dest, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	cond, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	if !cond.IsZero() {
		d := dest.Uint64()
		if !f.jumpdests.has(d) {
			return ErrInvalidJump
		}
		f.PC = d
	}
	return nil
}

func opPC(f *Frame) error {
	return f.Stack.PushRaw(f.PC - 1)
}

func opMsize(f *Frame) error {
	return f.Stack.PushRaw(uint64(f.Memory.Len()))
}

func opGas(f *Frame) error {
	return f.Stack.PushRaw(f.Gas)
}

func opJumpdest(_ *Frame) error { return nil }

func opPush(f *Frame, n int) error {
	var v uint256.Int
	end := f.PC + uint64(n)
	if end > uint64(len(f.Code)) {
		end = uint64(len(f.Code))
	}
	v.SetBytes(f.Code[f.PC:end])
	f.PC += uint64(n)
	return f.Stack.Push(&v)
}

func opDup(f *Frame, n int) error {
	return f.Stack.DupN(n)
}

func opSwap(f *Frame, n int) error {
	return f.Stack.SwapN(n)
}
