package main

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/ipfs/go-ipfs-api"
	"github.com/whyrusleeping/iptb/util"
)

func TestMain(m *testing.M) {
	iptbDir, nodes, err := iptbutil.SetupNodesForTest(2, "example-test-")
	if err != nil {
		tdErr := iptbutil.TeardownNodesForTest(iptbDir, nodes)
		if tdErr != nil {
			log.Fatalf("failed to setup nodes for test: %s, and then failed to tear them down: %s", err, tdErr)
		}
		log.Fatal("failed to setup nodes for test:", err)
	}

	r := m.Run()

	err = iptbutil.TeardownNodesForTest(iptbDir, nodes)
	if err != nil {
		log.Print("failed to tear down nodes for tests:", err)
	}

	os.Exit(r)
}

func AddressForNode(n int) (string, error) {
	node, err := iptbutil.LoadNodeN(n)
	if err != nil {
		return "", err
	}

	return node.APIAddr()
}

func NewShellForNode(n int) (*shell.Shell, error) {
	addr, err := AddressForNode(n)
	if err != nil {
		return nil, err
	}

	s := shell.NewShell(addr)
	if !s.IsUp() {
		return nil, fmt.Errorf("ipfs node does not seem to be up")
	}

	return s, nil
}
