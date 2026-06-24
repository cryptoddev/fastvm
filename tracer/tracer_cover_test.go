package tracer

import (
	"errors"
	"testing"

	"github.com/holiman/uint256"
)

func TestCaptureNoop(t *testing.T) {
	tr := New()
	tr.Capture(0, 0, 0, 0, 0, nil, 0, nil)
	tr.Capture(1, 1, 1, 1, 1, []uint256.Int{*uint256.NewInt(1)}, 1, nil)
	tr.Capture(2, 2, 2, 2, 2, nil, 2, errors.New("test error"))
}

func TestNoopProperties(t *testing.T) {
	tr := Noop()
	if tr.Enabled() {
		t.Fatal("Noop tracer should not be enabled")
	}
	if tr.Root != nil {
		t.Fatal("Noop tracer should have nil root")
	}
}
