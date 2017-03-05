package main

import (
	"io/ioutil"
	"net/http"
	"testing"
)

func TestWhatIPFSIsUpTo(t *testing.T) {
	addr, err := AddressForNode(0)
	if err != nil {
		t.Fatal("failed to get shell for node 0:", err)
	}

	res, err := http.Get("http://" + addr + "/api/v0/diag/cmds")
	if err != nil {
		t.Fatal("failed to get ipfs diag cmds:", err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("failed to read response body:", err)
	}

	t.Logf("body: %s", body)
}
