package interpreter

import (
	"github.com/holiman/uint256"
)

func opAdd(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	b.Add(a, b)
	return nil
}

func opMul(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	b.Mul(a, b)
	return nil
}

func opSub(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	b.Sub(a, b)
	return nil
}

func opDiv(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	if b.IsZero() {
		b.Clear()
	} else {
		b.Div(a, b)
	}
	return nil
}

func opSDiv(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	if b.IsZero() {
		b.Clear()
	} else {
		b.SDiv(a, b)
	}
	return nil
}

func opMod(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	if b.IsZero() {
		b.Clear()
	} else {
		b.Mod(a, b)
	}
	return nil
}

func opSMod(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	if b.IsZero() {
		b.Clear()
	} else {
		b.SMod(a, b)
	}
	return nil
}

func opAddMod(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	n := f.Stack.Back(0)
	if n.IsZero() {
		n.Clear()
	} else {
		n.AddMod(a, b, n)
	}
	return nil
}

func opMulMod(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	n := f.Stack.Back(0)
	if n.IsZero() {
		n.Clear()
	} else {
		n.MulMod(a, b, n)
	}
	return nil
}

func opExp(f *Frame) error {
	base, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	exp := f.Stack.Back(0)
	exp.Exp(base, exp)
	return nil
}

func opSignExtend(f *Frame) error {
	b, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	x := f.Stack.Back(0)
	x.ExtendSign(x, b)
	return nil
}

func opNot(f *Frame) error {
	x := f.Stack.Back(0)
	x.Not(x)
	return nil
}

func opLt(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	if a.Lt(b) {
		b.SetOne()
	} else {
		b.Clear()
	}
	return nil
}

func opGt(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	if a.Gt(b) {
		b.SetOne()
	} else {
		b.Clear()
	}
	return nil
}

func opSlt(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	if a.Slt(b) {
		b.SetOne()
	} else {
		b.Clear()
	}
	return nil
}

func opSgt(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	if a.Sgt(b) {
		b.SetOne()
	} else {
		b.Clear()
	}
	return nil
}

func opEq(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	if a.Eq(b) {
		b.SetOne()
	} else {
		b.Clear()
	}
	return nil
}

func opIsZero(f *Frame) error {
	x := f.Stack.Back(0)
	if x.IsZero() {
		x.SetOne()
	} else {
		x.Clear()
	}
	return nil
}

func opAnd(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	b.And(a, b)
	return nil
}

func opOr(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	b.Or(a, b)
	return nil
}

func opXor(f *Frame) error {
	a, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	b := f.Stack.Back(0)
	b.Xor(a, b)
	return nil
}

func opByte(f *Frame) error {
	i, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	x := f.Stack.Back(0)
	if i.LtUint64(32) {
		b32 := x.Bytes32()
		x.SetUint64(uint64(b32[i.Uint64()]))
	} else {
		x.Clear()
	}
	return nil
}

func opShl(f *Frame) error {
	shift, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	x := f.Stack.Back(0)
	if shift.LtUint64(256) {
		x.Lsh(x, uint(shift.Uint64()))
	} else {
		x.Clear()
	}
	return nil
}

func opShr(f *Frame) error {
	shift, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	x := f.Stack.Back(0)
	if shift.LtUint64(256) {
		x.Rsh(x, uint(shift.Uint64()))
	} else {
		x.Clear()
	}
	return nil
}

func opSar(f *Frame) error {
	shift, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	x := f.Stack.Back(0)
	if shift.GtUint64(255) {
		if x.Sign() < 0 {
			x.SetAllOne()
		} else {
			x.Clear()
		}
	} else {
		x.SRsh(x, uint(shift.Uint64()))
	}
	return nil
}

func opSha3(f *Frame) error {
	offset, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	size := f.Stack.Back(0)

	off := offset.Uint64()
	sz := size.Uint64()

	data := f.Memory.Get(off, sz)

	h := keccak256(data)
	size.SetBytes32(h[:])
	return nil
}

func keccak256(data []byte) [32]byte {
	h := newKeccak()
	h.Write(data)
	var out [32]byte
	h.Sum(out[:0])
	return out
}

var newKeccak func() interface{ Write([]byte) (int, error); Sum([]byte) []byte }

var uint256One = uint256.NewInt(1)
