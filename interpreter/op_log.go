package interpreter

import "github.com/cryptoddev/fastvm/state"

func opLog(f *Frame, n int) error {
	if f.Static {
		return ErrWriteProtection
	}
	offset, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	size, err := f.Stack.Pop()
	if err != nil {
		return err
	}
	topics := make([]state.Hash, n)
	for i := 0; i < n; i++ {
		t, err := f.Stack.Pop()
		if err != nil {
			return err
		}
		topics[i] = t.Bytes32()
	}
	data := f.Memory.GetCopy(offset.Uint64(), size.Uint64())
	f.State.AddLog(&state.Log{
		Address: f.Contract,
		Topics:  topics,
		Data:    data,
	})
	return nil
}
