package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"
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

	// the shutdown process is a slow one for some reason, so lets see if we can
	// get some goroutine info out of ipfs in the middle of it
	printIpfsGoroutines(addr)
	go func() {
		time.Sleep(3 * time.Second)
		printIpfsGoroutines(addr)
	}()
}

func printIpfsGoroutines(address string) {
	res, err := http.Get("http://" + address + "/debug/pprof/goroutine?debug=2")
	if err != nil {
		log.Println("failed to get ipfs pprof goroutine:", err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("failed to read pprof body:", err)
		return
	}

	log.Printf("pprof goroutine: %s\n", body)
}
