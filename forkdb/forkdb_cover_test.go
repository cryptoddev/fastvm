package forkdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchAccountDoubleCheck(t *testing.T) {
	srv := mockRPC(t, func(method string, params []interface{}) interface{} {
		return "0x0"
	})
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 0xDD

	if err := db.EnsureAccount(context.Background(), addr); err != nil {
		t.Fatal(err)
	}

	err := db.fetchAccount(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFetchSlotDoubleCheckInternal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	addr[19] = 1
	var slot Hash
	slot[31] = 1

	if err := db.EnsureSlot(context.Background(), addr, slot); err != nil {
		t.Fatal(err)
	}

	err := db.fetchSlot(context.Background(), addr, slot)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBatchCallMarshalError(t *testing.T) {
	db := New("http://example.com", 0)
	_, err := db.batchCall(context.Background(), make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestCallMarshalError(t *testing.T) {
	db := New("http://example.com", 0)
	var resp struct {
		Result string `json:"result"`
	}
	err := db.call(context.Background(), make(chan int), &resp)
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestBatchCallNewRequestError(t *testing.T) {
	db := &ForkDB{
		rpcURL: "\x00",
		httpClient: &http.Client{Timeout: time.Second},
	}
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
		t.Fatal("expected error from invalid URL")
	}
}

func TestCallNewRequestError(t *testing.T) {
	db := &ForkDB{
		rpcURL: "\x00",
		httpClient: &http.Client{Timeout: time.Second},
	}
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
		t.Fatal("expected error from invalid URL")
	}
}

func TestBatchCallReadAllError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":1,"result":"0x0"}]`))
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
		t.Fatal("expected error from body read")
	}
}

func TestFetchAccountInvalidCodeHex(t *testing.T) {
	srv := mockRPC(t, func(method string, params []interface{}) interface{} {
		switch method {
		case "eth_getBalance":
			return "0x0"
		case "eth_getTransactionCount":
			return "0x0"
		case "eth_getCode":
			return "0xZZ"
		}
		return "0x0"
	})
	defer srv.Close()

	db := New(srv.URL, 0)
	var addr Address
	addr[19] = 0xBB

	if err := db.EnsureAccount(context.Background(), addr); err != nil {
		t.Fatal(err)
	}

	code := db.State().GetCode(addr)
	if code != nil {
		t.Fatal("expected nil code for invalid code hex")
	}
}

func TestFetchSlotInvalidHex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0xZZ",
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
