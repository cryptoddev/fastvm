package evm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cryptoddev/fastvm/interpreter"
	"github.com/holiman/uint256"
)

func mockBatchRPC(t *testing.T, handler func(method string, params []interface{}) interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqs []struct {
			Method string        `json:"method"`
			Params []interface{} `json:"params"`
			ID     int           `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		type resp struct {
			JSONRPC string      `json:"jsonrpc"`
			ID      int         `json:"id"`
			Result  interface{} `json:"result"`
		}
		var responses []resp
		for _, req := range reqs {
			responses = append(responses, resp{JSONRPC: "2.0", ID: req.ID, Result: handler(req.Method, req.Params)})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responses)
	}))
}

func mockSingleRPC(t *testing.T, result interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  result,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestForkDBAdapterEnsureAndBalance(t *testing.T) {
	srv := mockBatchRPC(t, func(method string, params []interface{}) interface{} {
		switch method {
		case "eth_getBalance":
			return "0x64"
		case "eth_getTransactionCount":
			return "0x1"
		case "eth_getCode":
			return "0x"
		}
		return "0x0"
	})
	defer srv.Close()

	d := newForkDB(srv.URL, 0)
	var addr Address
	addr[19] = 0xAA
	d.ensure(addr)
	if d.GetBalance(addr).Uint64() != 100 {
		t.Fatalf("want balance=100, got %d", d.GetBalance(addr).Uint64())
	}
	if d.GetNonce(addr) != 1 {
		t.Fatalf("want nonce=1, got %d", d.GetNonce(addr))
	}
	if d.Exists(addr) != true {
		t.Fatal("want exists=true after ensure")
	}
	if d.Empty(addr) != false {
		t.Fatal("want empty=false after ensure with non-zero balance")
	}
}

func TestForkDBAdapterGetCodeAndCodeHash(t *testing.T) {
	srv := mockBatchRPC(t, func(method string, params []interface{}) interface{} {
		switch method {
		case "eth_getBalance":
			return "0x0"
		case "eth_getTransactionCount":
			return "0x0"
		case "eth_getCode":
			return "0x6001600053"
		}
		return "0x0"
	})
	defer srv.Close()

	d := newForkDB(srv.URL, 0)
	var addr Address
	addr[19] = 0xBB
	d.ensure(addr)
	code := d.GetCode(addr)
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
	h := d.GetCodeHash(addr)
	if h == (Hash{}) {
		t.Fatal("expected non-zero code hash")
	}
}

func TestForkDBAdapterEnsureSlot(t *testing.T) {
	srv := mockSingleRPC(t, "0x000000000000000000000000000000000000000000000000000000000000002a")
	defer srv.Close()

	d := newForkDB(srv.URL, 0)
	var addr Address
	addr[19] = 0xCC
	var slot Hash
	slot[31] = 1

	d.ensureSlot(addr, slot)
	val := d.GetStorage(addr, slot)
	if val[31] != 0x2a {
		t.Fatalf("want 0x2a, got %x", val[31])
	}
	committed := d.GetCommittedStorage(addr, slot)
	if committed[31] != 0x2a {
		t.Fatalf("want 0x2a, got %x", committed[31])
	}
}

func TestForkDBAdapterWriteMethods(t *testing.T) {
	srv := mockBatchRPC(t, func(method string, params []interface{}) interface{} {
		return "0x0"
	})
	defer srv.Close()

	d := newForkDB(srv.URL, 0)
	var addr, addr2 Address
	addr[19] = 0xDD
	addr2[19] = 0xEE

	d.CreateAccount(addr)
	d.SetBalance(addr, uint256.NewInt(1000))
	d.AddBalance(addr, uint256.NewInt(500))
	d.SubBalance(addr, uint256.NewInt(200))
	d.SetNonce(addr, 5)
	if d.GetBalance(addr).Uint64() != 1300 {
		t.Fatalf("want balance=1300, got %d", d.GetBalance(addr).Uint64())
	}
	if d.GetNonce(addr) != 5 {
		t.Fatalf("want nonce=5, got %d", d.GetNonce(addr))
	}

	code := []byte{interpreter.STOP}
	var h Hash
	h[0] = 1
	d.SetCode(addr, code, h)
	if len(d.GetCode(addr)) != 1 {
		t.Fatal("expected code to be set")
	}

	var slot, val Hash
	slot[31] = 1
	val[31] = 42
	d.SetStorage(addr, slot, val)
	if d.GetStorage(addr, slot)[31] != 42 {
		t.Fatal("expected storage to be 42")
	}
}

func TestForkDBAdapterRefundAndLog(t *testing.T) {
	srv := mockBatchRPC(t, func(method string, params []interface{}) interface{} {
		return "0x0"
	})
	defer srv.Close()

	d := newForkDB(srv.URL, 0)
	var addr Address
	addr[19] = 0xFF

	d.CreateAccount(addr)
	d.AddRefund(100)
	if d.GetRefund() != 100 {
		t.Fatalf("want refund=100, got %d", d.GetRefund())
	}
	d.SubRefund(30)
	if d.GetRefund() != 70 {
		t.Fatalf("want refund=70, got %d", d.GetRefund())
	}

}

func TestForkDBAdapterWarmAddress(t *testing.T) {
	srv := mockBatchRPC(t, func(method string, params []interface{}) interface{} {
		return "0x0"
	})
	defer srv.Close()

	d := newForkDB(srv.URL, 0)
	var addr Address
	addr[19] = 0xAA

	if d.AddressWarm(addr) {
		t.Fatal("address should be cold initially")
	}
	d.WarmAddress(addr)
	if !d.AddressWarm(addr) {
		t.Fatal("address should be warm after warming")
	}
}

func TestForkDBAdapterWarmSlot(t *testing.T) {
	srv := mockBatchRPC(t, func(method string, params []interface{}) interface{} {
		return "0x0"
	})
	defer srv.Close()

	d := newForkDB(srv.URL, 0)
	var addr Address
	addr[19] = 0xBB
	var slot Hash
	slot[31] = 5

	d.CreateAccount(addr)
	if d.SlotWarm(addr, slot) {
		t.Fatal("slot should be cold initially")
	}
	d.WarmSlot(addr, slot)
	if !d.SlotWarm(addr, slot) {
		t.Fatal("slot should be warm after warming")
	}
}

func TestForkDBAdapterSnapshotRevertFinalise(t *testing.T) {
	srv := mockBatchRPC(t, func(method string, params []interface{}) interface{} {
		return "0x0"
	})
	defer srv.Close()

	d := newForkDB(srv.URL, 0)
	var addr Address
	addr[19] = 0xCC

	d.CreateAccount(addr)
	d.SetBalance(addr, uint256.NewInt(100))
	snap := d.Snapshot()
	d.SetBalance(addr, uint256.NewInt(200))
	d.RevertToSnapshot(snap)
	if d.GetBalance(addr).Uint64() != 100 {
		t.Fatalf("want balance=100 after revert, got %d", d.GetBalance(addr).Uint64())
	}
	d.Finalise()
}

func TestForkDBAdapterSuicide(t *testing.T) {
	srv := mockBatchRPC(t, func(method string, params []interface{}) interface{} {
		return "0x0"
	})
	defer srv.Close()

	d := newForkDB(srv.URL, 0)
	var addr Address
	addr[19] = 0xDD

	d.CreateAccount(addr)
	d.SetBalance(addr, uint256.NewInt(100))
	if d.HasSuicided(addr) {
		t.Fatal("should not be suicided yet")
	}
	if !d.CreatedInTx(addr) {
		t.Fatal("should be created in tx (CreateAccount sets this)")
	}
	d.Suicide(addr)
	if !d.HasSuicided(addr) {
		t.Fatal("should be suicided after Suicide()")
	}
}

func TestForkDBAdapterEmpty(t *testing.T) {
	srv := mockBatchRPC(t, func(method string, params []interface{}) interface{} {
		switch method {
		case "eth_getBalance":
			return "0x0"
		case "eth_getTransactionCount":
			return "0x0"
		case "eth_getCode":
			return "0x"
		}
		return "0x0"
	})
	defer srv.Close()

	d := newForkDB(srv.URL, 0)
	var addr Address
	addr[19] = 0xEE

	d.ensure(addr)
	if !d.Empty(addr) {
		t.Fatal("account with zero balance/nonce/code should be empty")
	}
}


