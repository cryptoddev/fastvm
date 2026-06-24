package interpreter

import "errors"

type ExecResult struct {
	ReturnData []byte
	GasLeft    uint64
	Err        error
}

var (
	ErrOutOfGas          = errors.New("out of gas")
	ErrStackOverflow     = errors.New("stack overflow")
	ErrStackUnderflow    = errors.New("stack underflow")
	ErrInvalidJump       = errors.New("invalid jump destination")
	ErrWriteProtection   = errors.New("write protection")
	ErrReturnDataOOB     = errors.New("return data out of bounds")
	ErrInvalidOpcode     = errors.New("invalid opcode")
	ErrDepthLimit        = errors.New("call depth limit reached")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrGasUintOverflow   = errors.New("gas uint64 overflow")
)

const MaxCallDepth = 1024
