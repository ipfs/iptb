package iptbutil

import (
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"
)

// SetupNodesForTest is a helper function intended to be used in the context of
// a TestMain(*testing.M) function. It sets up some iptb based nodes and
// returns the name of the temporary iptb directory it created, the slice of
// nodes, and an error. Even when the error is not nil, the the directory or
// the nodes may contain useful information. The string and slice returned are
// always safe to pass to the TeardownNodesForTest function.
func SetupNodesForTest(count int, prefix string) (string, []IpfsNode, error) {
	iptbDir, err := ioutil.TempDir("", prefix)
	if err != nil {
		return "", nil, err
	}

	err = os.Setenv("IPTB_ROOT", iptbDir)
	if err != nil {
		return iptbDir, nil, err
	}

	// Need to randomize ports in case a previous run is taking its sweet time to
	// shut down, or something.
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	ps := 15000 + (rnd.Int()%500)*10

	cfg := &InitCfg{
		Count:     count,
		Force:     true,
		Bootstrap: "star",
		PortStart: ps,
		Mdns:      false,
		Utp:       false,
		Override:  "",
		NodeType:  "",
	}
	err = IpfsInit(cfg)
	if err != nil {
		return iptbDir, nil, err
	}

	nodes, err := LoadNodes()
	if err != nil {
		return iptbDir, nil, err
	}

	err = IpfsStart(nodes, true, []string{})
	if err != nil {
		for i, n := range nodes {
			killerr := n.Kill()
			if killerr != nil {
				log.Println("failed to kill node", i, ":", killerr)
			} else {
				log.Println("killed node", i)
			}
		}
		return iptbDir, nodes, err
	}

	return iptbDir, nodes, nil
}

// TeardownNodesForTest cleans up the iptb cluster that was set up by
// SetupNodesForTest. Always safe to call with the things returned by
// SetupNodesForTest.
func TeardownNodesForTest(iptbDir string, nodes []IpfsNode) error {
	if nodes != nil {
		err := IpfsKillAll(nodes)
		if err != nil {
			return err
		}
	}

	if iptbDir != "" {
		return os.RemoveAll(iptbDir)
	}

	return nil
}
