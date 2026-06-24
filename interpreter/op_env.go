package interpreter

import (
	"github.com/holiman/uint256"
)

func opAddress(f *Frame) error {
	var w uint256.Int
	addrToWord(&f.Contract, &w)
	return f.Stack.Push(&w)
}

func opBalance(f *Frame) error {
	addr := wordToAddr(f.Stack.Back(0))
	bal := f.State.GetBalance(addr)
	f.Stack.Back(0).Set(bal)
	return nil
}

func opOrigin(f *Frame) error {
	var w uint256.Int
	addrToWord(&f.Tx.Origin, &w)
	return f.Stack.Push(&w)
}

func opCaller(f *Frame) error {
	var w uint256.Int
	addrToWord(&f.Caller, &w)
	return f.Stack.Push(&w)
}

func opCallValue(f *Frame) error {
	var v uint256.Int
	v.Set(&f.Value)
	return f.Stack.Push(&v)
}

func opCalldataLoad(f *Frame) error {
	x := f.Stack.Back(0)
	off := x.Uint64()
	var buf [32]byte
	if off < uint64(len(f.Input)) {
		n := copy(buf[:], f.Input[off:])
		_ = n
	}
	x.SetBytes32(buf[:])
	return nil
}

func opCalldataSize(f *Frame) error {
	return f.Stack.PushRaw(uint64(len(f.Input)))
}

func opCalldataCopy(f *Frame) error {
	destOffset, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	offset, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	size, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	dest := destOffset.Uint64()
	off := offset.Uint64()
	sz := size.Uint64()
	if sz == 0 {
		return nil
	}
	dst := f.Memory.GetPtr(dest, sz)
	for i := range dst {
		dst[i] = 0
	}
	if off < uint64(len(f.Input)) {
		copy(dst, f.Input[off:])
	}
	return nil
}

func opCodeSize(f *Frame) error {
	return f.Stack.PushRaw(uint64(len(f.Code)))
}

func opCodeCopy(f *Frame) error {
	destOffset, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	offset, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	size, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	dest := destOffset.Uint64()
	off := offset.Uint64()
	sz := size.Uint64()
	if sz == 0 {
		return nil
	}
	dst := f.Memory.GetPtr(dest, sz)
	for i := range dst {
		dst[i] = 0
	}
	if off < uint64(len(f.Code)) {
		copy(dst, f.Code[off:])
	}
	return nil
}

func opGasPrice(f *Frame) error {
	var v uint256.Int
	v.Set(&f.Tx.GasPrice)
	return f.Stack.Push(&v)
}

func opExtCodeSize(f *Frame) error {
	addr := wordToAddr(f.Stack.Back(0))
	code := f.State.GetCode(addr)
	f.Stack.Back(0).SetUint64(uint64(len(code)))
	return nil
}

func opExtCodeCopy(f *Frame) error {
	addrWord, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	destOffset, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	offset, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	size, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	addr := wordToAddr(addrWord)
	code := f.State.GetCode(addr)
	dest := destOffset.Uint64()
	off := offset.Uint64()
	sz := size.Uint64()
	if sz == 0 {
		return nil
	}
	dst := f.Memory.GetPtr(dest, sz)
	for i := range dst {
		dst[i] = 0
	}
	if off < uint64(len(code)) {
		copy(dst, code[off:])
	}
	return nil
}

func opReturnDataSize(f *Frame) error {
	return f.Stack.PushRaw(uint64(len(f.ReturnData)))
}

func opReturnDataCopy(f *Frame) error {
	destOffset, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	offset, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	size, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	dest := destOffset.Uint64()
	off := offset.Uint64()
	sz := size.Uint64()
	end := off + sz
	if end < off || end > uint64(len(f.ReturnData)) {
		return ErrReturnDataOOB
	}
	if sz == 0 {
		return nil
	}
	dst := f.Memory.GetPtr(dest, sz)
	copy(dst, f.ReturnData[off:end])
	return nil
}

func opExtCodeHash(f *Frame) error {
	addr := wordToAddr(f.Stack.Back(0))
	if f.State.Empty(addr) {
		f.Stack.Back(0).Clear()
		return nil
	}
	h := f.State.GetCodeHash(addr)
	f.Stack.Back(0).SetBytes32(h[:])
	return nil
}

func opBlockHash(f *Frame) error {
	f.Stack.Back(0).Clear()
	return nil
}

func opCoinbase(f *Frame) error {
	var w uint256.Int
	addrToWord(&f.Block.Coinbase, &w)
	return f.Stack.Push(&w)
}

func opTimestamp(f *Frame) error {
	return f.Stack.PushRaw(f.Block.Time)
}

func opNumber(f *Frame) error {
	return f.Stack.PushRaw(f.Block.BlockNumber)
}

func opDifficulty(f *Frame) error {
	var v uint256.Int
	v.Set(&f.Block.Difficulty)
	return f.Stack.Push(&v)
}

func opGasLimit(f *Frame) error {
	return f.Stack.PushRaw(f.Block.GasLimit)
}

func opChainID(f *Frame) error {
	var v uint256.Int
	v.Set(&f.Block.ChainID)
	return f.Stack.Push(&v)
}

func opSelfBalance(f *Frame) error {
	bal := f.State.GetBalance(f.Contract)
	return f.Stack.Push(bal)
}

func opBaseFee(f *Frame) error {
	var v uint256.Int
	v.Set(&f.Block.BaseFee)
	return f.Stack.Push(&v)
}

func addrToWord(a *Address, w *uint256.Int) {
	var b [32]byte
	copy(b[12:], a[:])
	w.SetBytes32(b[:])
}

func wordToAddr(w *uint256.Int) Address {
	b := w.Bytes32()
	var a Address
	copy(a[:], b[12:])
	return a
}
