package iptbutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	serial "github.com/ipfs/go-ipfs/repo/fsrepo/serialize"
)

// GetNumNodes returns the number of testbed nodes configured in the testbed directory
func GetNumNodes() int {
	for i := 0; i < 2000; i++ {
		dir, err := IpfsDirN(i)
		if err != nil {
			return i
		}
		_, err = os.Stat(dir)
		if os.IsNotExist(err) {
			return i
		}
	}
	panic("i dont know whats going on")
}

func TestBedDir() (string, error) {
	tbd := os.Getenv("IPTB_ROOT")
	if len(tbd) != 0 {
		return tbd, nil
	}

	home := os.Getenv("HOME")
	if len(home) == 0 {
		return "", fmt.Errorf("environment variable HOME is not set")
	}

	return path.Join(home, "testbed"), nil
}

func IpfsDirN(n int) (string, error) {
	tbd, err := TestBedDir()
	if err != nil {
		return "", err
	}
	return path.Join(tbd, fmt.Sprint(n)), nil
}

type InitCfg struct {
	Count     int
	Force     bool
	Bootstrap string
	PortStart int
	Mdns      bool
	Utp       bool
	Override  string
}

func (c *InitCfg) swarmAddrForPeer(i int) string {
	str := "/ip4/0.0.0.0/tcp/%d"
	if c.Utp {
		str = "/ip4/0.0.0.0/udp/%d/utp"
	}

	if c.PortStart == 0 {
		return fmt.Sprintf(str, 0)
	}
	return fmt.Sprintf(str, c.PortStart+i)
}

func (c *InitCfg) apiAddrForPeer(i int) string {
	if c.PortStart == 0 {
		return "/ip4/127.0.0.1/tcp/0"
	}
	return fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", c.PortStart+1000+i)
}

func YesNoPrompt(prompt string) bool {
	var s string
	for {
		fmt.Println(prompt)
		fmt.Scanf("%s", &s)
		switch s {
		case "y", "Y":
			return true
		case "n", "N":
			return false
		}
		fmt.Println("Please press either 'y' or 'n'")
	}
}

func LoadLocalNodeN(n int) (*LocalNode, error) {
	dir, err := IpfsDirN(n)
	if err != nil {
		return nil, err
	}
	pid, err := GetPeerID(n)
	if err != nil {
		return nil, err
	}

	return &LocalNode{
		Dir:    dir,
		PeerID: pid,
	}, nil
}

func LoadNodes() ([]IpfsNode, error) {
	n := GetNumNodes()
	var out []IpfsNode
	for i := 0; i < n; i++ {
		ln, err := LoadLocalNodeN(i)
		if err != nil {
			return nil, err
		}
		out = append(out, ln)
	}
	return out, nil
}

func IpfsInit(cfg *InitCfg) error {
	dir, err := IpfsDirN(0)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		if !cfg.Force && !YesNoPrompt("testbed nodes already exist, overwrite? [y/n]") {
			return nil
		}
		tbd, err := TestBedDir()
		err = os.RemoveAll(tbd)
		if err != nil {
			return err
		}
	}
	wait := sync.WaitGroup{}
	for i := 0; i < cfg.Count; i++ {
		wait.Add(1)
		go func(v int) {
			defer wait.Done()
			dir, err := IpfsDirN(v)
			if err != nil {
				log.Println("ERROR: ", err)
				return
			}
			err = os.MkdirAll(dir, 0777)
			if err != nil {
				log.Println("ERROR: ", err)
				return
			}

			cmd := exec.Command("ipfs", "init", "-b=1024")
			cmd.Env = append(cmd.Env, "IPFS_PATH="+dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Println("ERROR: ", err)
				log.Println(string(out))
			}
		}(i)
	}
	wait.Wait()

	// Now setup bootstrapping
	switch cfg.Bootstrap {
	case "star":
		err := starBootstrap(cfg)
		if err != nil {
			return err
		}
	case "none":
		err := clearBootstrapping(cfg)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized bootstrapping option: %s", cfg.Bootstrap)
	}

	/*
		if cfg.Override != "" {
			err := ApplyConfigOverride(cfg)
			if err != nil {
				return err
			}
		}
	*/

	return nil
}

func ApplyConfigOverride(cfg *InitCfg) error {
	fir, err := os.Open(cfg.Override)
	if err != nil {
		return err
	}
	defer fir.Close()

	var configs map[string]interface{}
	err = json.NewDecoder(fir).Decode(&configs)
	if err != nil {
		return err
	}

	for i := 0; i < cfg.Count; i++ {
		err := applyOverrideToNode(configs, i)
		if err != nil {
			return err
		}
	}

	return nil
}

func applyOverrideToNode(ovr map[string]interface{}, node int) error {
	for k, v := range ovr {
		_ = k
		switch v.(type) {
		case map[string]interface{}:
		default:
		}

	}

	panic("not implemented")
}

func starBootstrap(icfg *InitCfg) error {
	// '0' node is the bootstrap node
	dir, err := IpfsDirN(0)
	if err != nil {
		return err
	}

	cfgpath := path.Join(dir, "config")
	bcfg, err := serial.Load(cfgpath)
	if err != nil {
		return err
	}
	bcfg.Bootstrap = nil
	bcfg.Addresses.Swarm = []string{icfg.swarmAddrForPeer(0)}
	bcfg.Addresses.API = icfg.apiAddrForPeer(0)
	bcfg.Addresses.Gateway = ""
	bcfg.Discovery.MDNS.Enabled = icfg.Mdns
	err = serial.WriteConfigFile(cfgpath, bcfg)
	if err != nil {
		return err
	}

	for i := 1; i < icfg.Count; i++ {
		dir, err := IpfsDirN(i)
		if err != nil {
			return err
		}
		cfgpath := path.Join(dir, "config")
		cfg, err := serial.Load(cfgpath)
		if err != nil {
			return err
		}

		ba := fmt.Sprintf("%s/ipfs/%s", bcfg.Addresses.Swarm[0], bcfg.Identity.PeerID)
		ba = strings.Replace(ba, "0.0.0.0", "127.0.0.1", -1)
		cfg.Bootstrap = []string{ba}
		cfg.Addresses.Gateway = ""
		cfg.Discovery.MDNS.Enabled = icfg.Mdns
		cfg.Addresses.Swarm = []string{
			icfg.swarmAddrForPeer(i),
		}
		cfg.Addresses.API = icfg.apiAddrForPeer(i)
		err = serial.WriteConfigFile(cfgpath, cfg)
		if err != nil {
			return err
		}
	}
	return nil
}

func clearBootstrapping(icfg *InitCfg) error {
	for i := 0; i < icfg.Count; i++ {
		dir, err := IpfsDirN(i)
		if err != nil {
			return err
		}
		cfgpath := path.Join(dir, "config")
		cfg, err := serial.Load(cfgpath)
		if err != nil {
			return err
		}

		cfg.Bootstrap = nil
		cfg.Addresses.Gateway = ""
		cfg.Addresses.Swarm = []string{icfg.swarmAddrForPeer(i)}
		cfg.Addresses.API = icfg.apiAddrForPeer(i)
		cfg.Discovery.MDNS.Enabled = icfg.Mdns
		err = serial.WriteConfigFile(cfgpath, cfg)
		if err != nil {
			return err
		}
	}
	return nil
}

func IpfsKillAll(nds []IpfsNode) error {
	for _, n := range nds {
		err := n.Kill()
		if err != nil {
			return err
		}
	}
	return nil
}

func IpfsStart(nodes []IpfsNode, waitall bool) error {
	for _, n := range nodes {
		if err := n.Start(); err != nil {
			return err
		}
	}
	if waitall {
		for _, n := range nodes {
			err := waitOnSwarmPeers(n)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func waitOnAPI(n IpfsNode) error {
	for i := 0; i < 50; i++ {
		err := tryAPICheck(n)
		if err == nil {
			return nil
		}
		time.Sleep(time.Millisecond * 200)
	}
	return fmt.Errorf("node %s failed to come online in given time period", n.GetPeerID())
}

func tryAPICheck(n IpfsNode) error {
	addr, err := n.APIAddr()
	if err != nil {
		return err
	}

	resp, err := http.Get("http://" + addr + "/api/v0/id")
	if err != nil {
		return err
	}

	out := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return fmt.Errorf("liveness check failed: %s", err)
	}

	id, ok := out["ID"]
	if !ok {
		return fmt.Errorf("liveness check failed: ID field not present in output")
	}

	idstr := id.(string)
	if idstr != n.GetPeerID() {
		return fmt.Errorf("liveness check failed: unexpected peer at endpoint")
	}

	return nil
}

func waitOnSwarmPeers(n IpfsNode) error {
	addr, err := n.APIAddr()
	if err != nil {
		return err
	}

	for i := 0; i < 50; i++ {
		resp, err := http.Get("http://" + addr + "/api/v0/swarm/peers")
		if err == nil {
			out := make(map[string]interface{})
			err := json.NewDecoder(resp.Body).Decode(&out)
			if err != nil {
				return fmt.Errorf("liveness check failed: %s", err)
			}

			peers := out["Strings"].([]interface{})
			if len(peers) == 0 {
				time.Sleep(time.Millisecond * 200)
				continue
			}

			return nil
		}
		time.Sleep(time.Millisecond * 200)
	}
	return fmt.Errorf("node at %s failed to bootstrap in given time period", addr)
}

// GetPeerID reads the config of node 'n' and returns its peer ID
func GetPeerID(n int) (string, error) {
	dir, err := IpfsDirN(n)
	if err != nil {
		return "", err
	}
	cfg, err := serial.Load(path.Join(dir, "config"))
	if err != nil {
		return "", err
	}
	return cfg.Identity.PeerID, nil
}

func ConnectNodes(from, to IpfsNode) error {
	if from == to {
		// skip connecting to self..
		return nil
	}

	out, err := to.RunCmd("ipfs", "id", "-f", "<addrs>")
	if err != nil {
		return err
	}

	addr := strings.Split(string(out), "\n")[0]
	fmt.Printf("connecting %s -> %s\n", from, to)

	_, err = from.RunCmd("ipfs", "swarm", "connect", addr)
	if err != nil {
		return err
	}

	return nil
}

type BW struct {
	TotalIn  int
	TotalOut int
}

func GetBW(n IpfsNode) (*BW, error) {
	addr, err := n.APIAddr()
	if err != nil {
		return nil, err
	}

	resp, err := http.Get("http://" + addr + "/api/v0/stats/bw")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var bw BW
	err = json.NewDecoder(resp.Body).Decode(&bw)
	if err != nil {
		return nil, err
	}

	return &bw, nil
}

const (
	attrId    = "id"
	attrPath  = "path"
	attrBwIn  = "bw_in"
	attrBwOut = "bw_out"
)

func GetListOfAttr() []string {
	return []string{attrId, attrPath, attrBwIn, attrBwOut}
}

func GetAttrDescr(attr string) (string, error) {
	switch attr {
	case attrId:
		return "node ID", nil
	case attrPath:
		return "node IPFS_PATH", nil
	case attrBwIn:
		return "node input bandwidth", nil
	case attrBwOut:
		return "node output bandwidth", nil
	default:
		return "", errors.New("unrecognized attribute")
	}
}
