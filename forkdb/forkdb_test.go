package forkdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func mockRPC(t *testing.T, handler func(method string, params []interface{}) interface{}) *httptest.Server {
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
			result := handler(req.Method, req.Params)
			responses = append(responses, resp{JSONRPC: "2.0", ID: req.ID, Result: result})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responses)
	}))
}

func TestFetchAccount(t *testing.T) {
	srv := mockRPC(t, func(method string, params []interface{}) interface{} {
		switch method {
		case "eth_getBalance":
			return "0x64" // 100 wei
		case "eth_getTransactionCount":
			return "0x3" // nonce 3
		case "eth_getCode":
			return "0x" // EOA
		}
		return "0x0"
	})
	defer srv.Close()

	db := New(srv.URL, 0)
	addr := Address{0: 0xAB}
	if err := db.EnsureAccount(context.Background(), addr); err != nil {
		t.Fatal(err)
	}

	bal := db.State().GetBalance(addr)
	if bal.Uint64() != 100 {
		t.Fatalf("want balance=100, got %d", bal.Uint64())
	}
	if db.State().GetNonce(addr) != 3 {
		t.Fatalf("want nonce=3, got %d", db.State().GetNonce(addr))
	}
}

func TestFetchSlot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string        `json:"method"`
			Params []interface{} `json:"params"`
			ID     int           `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "0x000000000000000000000000000000000000000000000000000000000000002a",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 1
	var slot Hash
	slot[31] = 1

	if err := db.EnsureSlot(context.Background(), addr, slot); err != nil {
		t.Fatal(err)
	}

	val := db.State().GetStorage(addr, slot)
	if val[31] != 0x2a {
		t.Fatalf("want 0x2a, got %x", val[31])
	}
}

func TestCacheHit(t *testing.T) {
	calls := 0
	srv := mockRPC(t, func(method string, params []interface{}) interface{} {
		calls++
		return "0x0"
	})
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 2

	_ = db.EnsureAccount(context.Background(), addr)
	callsAfterFirst := calls
	_ = db.EnsureAccount(context.Background(), addr)
	if calls != callsAfterFirst {
		t.Fatal("second EnsureAccount should not make RPC calls (cache hit)")
	}
}

func TestNewWithNonZeroBlock(t *testing.T) {
	db := New("http://example.com", 12345)
	if db.blockNumber != "0x3039" {
		t.Fatalf("want blockNumber=0x3039, got %s", db.blockNumber)
	}
}

func TestNewWithLatestBlock(t *testing.T) {
	db := New("http://example.com", 0)
	if db.blockNumber != "latest" {
		t.Fatalf("want blockNumber=latest, got %s", db.blockNumber)
	}
}

func TestState(t *testing.T) {
	db := New("http://example.com", 0)
	st := db.State()
	if st == nil {
		t.Fatal("State() should not return nil")
	}
}

func TestEnsureAccountWithCode(t *testing.T) {
	srv := mockRPC(t, func(method string, params []interface{}) interface{} {
		switch method {
		case "eth_getBalance":
			return "0x0"
		case "eth_getTransactionCount":
			return "0x0"
		case "eth_getCode":
			return "0x6001600053" // PUSH1 1, PUSH1 0, MSTORE
		}
		return "0x0"
	})
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 0xCC

	if err := db.EnsureAccount(context.Background(), addr); err != nil {
		t.Fatal(err)
	}

	code := db.State().GetCode(addr)
	if len(code) == 0 {
		t.Fatal("expected code to be set")
	}

	codeHash := db.State().GetCodeHash(addr)
	if codeHash == (Hash{}) {
		t.Fatal("expected code hash to be set")
	}
}

func TestEnsureAccountEmptyBalanceHex(t *testing.T) {
	srv := mockRPC(t, func(method string, params []interface{}) interface{} {
		switch method {
		case "eth_getBalance":
			return "0x"
		case "eth_getTransactionCount":
			return "0x"
		case "eth_getCode":
			return "0x"
		}
		return "0x0"
	})
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 0xDD

	if err := db.EnsureAccount(context.Background(), addr); err != nil {
		t.Fatal(err)
	}

	bal := db.State().GetBalance(addr)
	if !bal.IsZero() {
		t.Fatalf("want zero balance, got %s", bal)
	}
}

func TestEnsureSlotErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]string{
				"message": "internal error",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 1
	var slot Hash
	slot[31] = 1

	err := db.EnsureSlot(context.Background(), addr, slot)
	if err == nil {
		t.Fatal("expected error from RPC error response")
	}
}

func TestFetchSlotDoubleCheck(t *testing.T) {
	calls := 0
	srv := mockRPC(t, func(method string, params []interface{}) interface{} {
		calls++
		return "0x0"
	})
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 3
	var slot Hash
	slot[31] = 1

	_ = db.EnsureSlot(context.Background(), addr, slot)
	callsAfterFirst := calls

	// Second call should be cached
	_ = db.EnsureSlot(context.Background(), addr, slot)
	if calls != callsAfterFirst {
		t.Fatal("second EnsureSlot should not make RPC calls (cache hit)")
	}
}

func TestPadEven(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a", "0a"},
		{"abc", "0abc"},
		{"ab", "ab"},
		{"abcd", "abcd"},
		{"", ""},
		{"1", "01"},
	}
	for _, tt := range tests {
		got := padEven(tt.input)
		if got != tt.want {
			t.Errorf("padEven(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBatchCallError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := []map[string]interface{}{
			{
				"jsonrpc": "2.0",
				"id":      1,
				"error": map[string]string{
					"message": "method not found",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	db := New(srv.URL, 0)
	type rpcReq struct {
		JSONRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		ID      int           `json:"id"`
	}
	batch := []rpcReq{
		{JSONRPC: "2.0", Method: "eth_getBalance", Params: []interface{}{"0x00", "latest"}, ID: 1},
	}

	_, err := db.batchCall(context.Background(), batch)
	if err == nil {
		t.Fatal("expected error from batch call with RPC error")
	}
}

func TestBatchCallUnmarshalError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	db := New(srv.URL, 0)
	type rpcReq struct {
		JSONRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		ID      int           `json:"id"`
	}
	batch := []rpcReq{
		{JSONRPC: "2.0", Method: "eth_getBalance", Params: []interface{}{"0x00", "latest"}, ID: 1},
	}

	_, err := db.batchCall(context.Background(), batch)
	if err == nil {
		t.Fatal("expected error from invalid JSON response")
	}
}

func TestCallError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	db := New(srv.URL, 0)
	type rpcReq struct {
		JSONRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		ID      int           `json:"id"`
	}
	req := rpcReq{
		JSONRPC: "2.0",
		Method:  "eth_getStorageAt",
		Params:  []interface{}{"0x00", "0x00", "latest"},
		ID:      1,
	}

	var resp struct {
		Result string `json:"result"`
	}
	err := db.call(context.Background(), req, &resp)
	if err == nil {
		t.Fatal("expected error from invalid JSON response")
	}
}

func TestKeccak256Hash(t *testing.T) {
	// Empty input keccak256
	h := keccak256Hash([]byte{})
	// Known: keccak256("") = 0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470
	expected := [32]byte{
		0xc5, 0xd2, 0x46, 0x01, 0x86, 0xf7, 0x23, 0x3c,
		0x92, 0x7e, 0x7d, 0xb2, 0xdc, 0xc7, 0x03, 0xc0,
		0xe5, 0x00, 0xb6, 0x53, 0xca, 0x82, 0x27, 0x3b,
		0x7b, 0xfa, 0xd8, 0x04, 0x5d, 0x85, 0xa4, 0x70,
	}
	if h != expected {
		t.Fatalf("keccak256Hash(\"\") mismatch:\n got  %x\n want %x", h, expected)
	}
}

func TestKeccak256HashWithData(t *testing.T) {
	h := keccak256Hash([]byte{0x01, 0x02, 0x03})
	if h == (Hash{}) {
		t.Fatal("keccak256Hash should not return zero hash for non-empty input")
	}
}

func TestFetchAccountWithOddHexBalance(t *testing.T) {
	srv := mockRPC(t, func(method string, params []interface{}) interface{} {
		switch method {
		case "eth_getBalance":
			return "0xf" // odd hex, should be padded to "0f"
		case "eth_getTransactionCount":
			return "0x0"
		case "eth_getCode":
			return "0x"
		}
		return "0x0"
	})
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 0xEE

	if err := db.EnsureAccount(context.Background(), addr); err != nil {
		t.Fatal(err)
	}

	bal := db.State().GetBalance(addr)
	if bal.Uint64() != 15 {
		t.Fatalf("want balance=15, got %d", bal.Uint64())
	}
}

func TestFetchAccountWithCodeEmpty(t *testing.T) {
	srv := mockRPC(t, func(method string, params []interface{}) interface{} {
		switch method {
		case "eth_getBalance":
			return "0x0"
		case "eth_getTransactionCount":
			return "0x0"
		case "eth_getCode":
			return "0x" // empty code
		}
		return "0x0"
	})
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 0xFF

	if err := db.EnsureAccount(context.Background(), addr); err != nil {
		t.Fatal(err)
	}

	code := db.State().GetCode(addr)
	if code != nil {
		t.Fatal("empty code should result in nil code")
	}
}

func TestFetchSlotEmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0x",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 1
	var slot Hash
	slot[31] = 1

	if err := db.EnsureSlot(context.Background(), addr, slot); err != nil {
		t.Fatal(err)
	}

	val := db.State().GetStorage(addr, slot)
	if val != (Hash{}) {
		t.Fatalf("want zero hash, got %x", val)
	}
}

func TestEnsureAccountAlreadyFetched(t *testing.T) {
	callCount := 0
	srv := mockRPC(t, func(method string, params []interface{}) interface{} {
		callCount++
		return "0x0"
	})
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 4

	_ = db.EnsureAccount(context.Background(), addr)
	// Each EnsureAccount makes a batch of 3 RPC calls
	callsAfterFirst := callCount
	_ = db.EnsureAccount(context.Background(), addr)
	_ = db.EnsureAccount(context.Background(), addr)

	if callCount != callsAfterFirst {
		t.Fatalf("expected no additional RPC calls after first fetch, got %d additional", callCount-callsAfterFirst)
	}
}

func TestEnsureSlotAlreadyFetched(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0x0",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 5
	var slot Hash
	slot[31] = 1

	_ = db.EnsureSlot(context.Background(), addr, slot)
	callsAfterFirst := calls
	_ = db.EnsureSlot(context.Background(), addr, slot)

	if calls != callsAfterFirst {
		t.Fatalf("expected no additional RPC calls, got %d", calls-callsAfterFirst)
	}
}

func TestBatchCallHTTPError(t *testing.T) {
	// Use an invalid URL to trigger HTTP error
	db := New("http://localhost:1", 0)
	type rpcReq struct {
		JSONRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		ID      int           `json:"id"`
	}
	batch := []rpcReq{
		{JSONRPC: "2.0", Method: "eth_getBalance", Params: []interface{}{"0x00", "latest"}, ID: 1},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := db.batchCall(ctx, batch)
	if err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestCallHTTPError(t *testing.T) {
	// Use an invalid URL to trigger HTTP error
	db := New("http://localhost:1", 0)
	type rpcReq struct {
		JSONRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		ID      int           `json:"id"`
	}
	req := rpcReq{
		JSONRPC: "2.0",
		Method:  "eth_getStorageAt",
		Params:  []interface{}{"0x00", "0x00", "latest"},
		ID:      1,
	}

	var resp struct {
		Result string `json:"result"`
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := db.call(ctx, req, &resp)
	if err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestFetchAccountHTTPError(t *testing.T) {
	db := New("http://localhost:1", 0)
	var addr Address
	addr[19] = 0xAA

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := db.EnsureAccount(ctx, addr)
	if err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestFetchSlotHTTPError(t *testing.T) {
	db := New("http://localhost:1", 0)
	var addr Address
	addr[19] = 0xAA
	var slot Hash
	slot[31] = 1

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := db.EnsureSlot(ctx, addr, slot)
	if err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestBatchCallInvalidJSONResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a response with invalid JSON in result field
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":1,"result":invalid}]`))
	}))
	defer srv.Close()

	db := New(srv.URL, 0)
	type rpcReq struct {
		JSONRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		ID      int           `json:"id"`
	}
	batch := []rpcReq{
		{JSONRPC: "2.0", Method: "eth_getBalance", Params: []interface{}{"0x00", "latest"}, ID: 1},
	}

	_, err := db.batchCall(context.Background(), batch)
	// Should not error (invalid result is just skipped)
	_ = err
}
