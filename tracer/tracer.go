package tracer

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/cryptoddev/fastvm/interpreter"
	"github.com/holiman/uint256"
)

type CallType = interpreter.CallType

const (
	CallTypeCall         = interpreter.CallTypeCall
	CallTypeCallCode     = interpreter.CallTypeCallCode
	CallTypeDelegateCall = interpreter.CallTypeDelegateCall
	CallTypeStaticCall   = interpreter.CallTypeStaticCall
	CallTypeCreate       = interpreter.CallTypeCreate
	CallTypeCreate2      = interpreter.CallTypeCreate2
)

type Node struct {
	CallType   CallType
	From       [20]byte
	To         [20]byte
	Gas        uint64
	GasUsed    uint64
	Value      uint256.Int
	Input      []byte
	Output     []byte
	Err        error
	Children   []*Node
	parent     *Node
}

type CallTracer struct {
	Root    *Node
	current *Node
	enabled bool
}

func New() *CallTracer        { return &CallTracer{enabled: true} }
func Noop() *CallTracer       { return &CallTracer{} }
func (t *CallTracer) Enabled() bool { return t.enabled }

func (t *CallTracer) OnEnter(ct CallType, from, to [20]byte, input []byte, gas uint64, value *uint256.Int) {
	if !t.enabled {
		return
	}
	n := &Node{
		CallType: ct,
		From:     from,
		To:       to,
		Gas:      gas,
		Input:    input,
		parent:   t.current,
	}
	if value != nil {
		n.Value.Set(value)
	}
	if t.current == nil {
		t.Root = n
	} else {
		t.current.Children = append(t.current.Children, n)
	}
	t.current = n
}

func (t *CallTracer) OnExit(output []byte, gasUsed uint64, err error) {
	if !t.enabled || t.current == nil {
		return
	}
	t.current.Output = output
	t.current.GasUsed = gasUsed
	t.current.Err = err
	t.current = t.current.parent
}

func (t *CallTracer) Capture(_ uint64, _ byte, _, _ uint64, _ int, _ []uint256.Int, _ int, _ error) {}

func (t *CallTracer) String() string {
	if t.Root == nil {
		return "(empty trace)"
	}
	var sb strings.Builder
	renderNode(&sb, t.Root, "", true)
	return sb.String()
}

func renderNode(sb *strings.Builder, n *Node, prefix string, isLast bool) {
	branch := "├─ "
	childPrefix := prefix + "│   "
	if isLast {
		branch = "└─ "
		childPrefix = prefix + "    "
	}

	callStr := callTypeName(n.CallType)
	addrStr := fmt.Sprintf("%x", n.To)
	if len(addrStr) > 8 {
		addrStr = addrStr[:4] + ".." + addrStr[len(addrStr)-4:]
	}

	inputStr := ""
	if len(n.Input) >= 4 {
		inputStr = hex.EncodeToString(n.Input[:4])
		if len(n.Input) > 4 {
			inputStr += fmt.Sprintf("(%db)", len(n.Input)-4)
		}
	} else if len(n.Input) > 0 {
		inputStr = hex.EncodeToString(n.Input)
	}

	line := fmt.Sprintf("[%d] %s::%s(%s)", n.GasUsed, addrStr, callStr, inputStr)
	if n.parent == nil {
		sb.WriteString(line + "\n")
	} else {
		sb.WriteString(prefix + branch + line + "\n")
	}

	for i, child := range n.Children {
		last := i == len(n.Children)-1
		renderNode(sb, child, childPrefix, last)
	}

	retPrefix := childPrefix
	if n.Err != nil {
		sb.WriteString(retPrefix + "└─ ← REVERT " + revertReason(n.Output) + "\n")
	} else {
		out := ""
		if len(n.Output) > 0 {
			if len(n.Output) > 32 {
				out = fmt.Sprintf("0x%s..(%db)", hex.EncodeToString(n.Output[:4]), len(n.Output))
			} else {
				out = "0x" + hex.EncodeToString(n.Output)
			}
		}
		sb.WriteString(retPrefix + "└─ ← " + out + "\n")
	}
}

func callTypeName(ct CallType) string {
	switch ct {
	case CallTypeCall:
		return "CALL"
	case CallTypeCallCode:
		return "CALLCODE"
	case CallTypeDelegateCall:
		return "DELEGATECALL"
	case CallTypeStaticCall:
		return "STATICCALL"
	case CallTypeCreate:
		return "CREATE"
	case CallTypeCreate2:
		return "CREATE2"
	}
	return "CALL"
}

func revertReason(data []byte) string {
	// ABI-encoded Error(string): selector 0x08c379a0, offset at [4:36], length at [36:68], string at [68:]
	if len(data) >= 68 &&
		data[0] == 0x08 && data[1] == 0xc3 && data[2] == 0x79 && data[3] == 0xa0 {
		strLen := int(data[67])
		end := 68 + strLen
		if end <= len(data) {
			return fmt.Sprintf(`"%s"`, string(data[68:end]))
		}
	}
	if len(data) > 0 {
		return "0x" + hex.EncodeToString(data)
	}
	return ""
}
