package iptbutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	serial "github.com/ipfs/go-ipfs/repo/fsrepo/serialize"
	"github.com/whyrusleeping/stump"
)

// NetnsNode is an IPFS node in its own network namespace controlled
// by IPTB
type NetnsNode struct {
	Name string

	apiAddr string

	LocalNode
}

// assert DockerNode satisfies the testbed IpfsNode interface
var _ IpfsNode = (*NetnsNode)(nil)

func haveNamespace(name string) (bool, error) {
	out, err := exec.Command("ip", "netns").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf(string(out))
	}
	lines := bytes.Split(out, []byte("\n"))
	for _, l := range lines {
		if bytes.HasPrefix(l, []byte(name)) {
			return true, nil
		}
	}
	return false, nil
}

func (nsn *NetnsNode) Start(args []string) error {
	have, err := haveNamespace(nsn.Name)
	if err != nil {
		return err
	}
	if !have {
		return fmt.Errorf("network namespace %q not found", nsn.Name)
	}

	if len(args) > 0 {
		return fmt.Errorf("cannot yet pass daemon args to docker nodes")
	}

	alive, err := nsn.isAlive()
	if err != nil {
		return err
	}

	if alive {
		return fmt.Errorf("node is already running")
	}

	dir := nsn.LocalNode.Dir
	dargs := append([]string{"netns", "exec", nsn.Name, "ipfs", "daemon"}, args...)
	cmd := exec.Command("ip", dargs...)
	cmd.Dir = dir

	cmd.Env, err = nsn.envForDaemon()
	if err != nil {
		return err
	}

	setupOpt(cmd)

	stdout, err := os.Create(filepath.Join(dir, "daemon.stdout"))
	if err != nil {
		return err
	}

	stderr, err := os.Create(filepath.Join(dir, "daemon.stderr"))
	if err != nil {
		return err
	}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Start()
	if err != nil {
		return err
	}
	pid := cmd.Process.Pid

	fmt.Printf("Started daemon %s, pid = %d\n", dir, pid)
	err = ioutil.WriteFile(filepath.Join(dir, "daemon.pid"), []byte(fmt.Sprint(pid)), 0666)
	if err != nil {
		return err
	}

	// Make sure node 0 is up before starting the rest so
	// bootstrapping works properly
	cfg, err := serial.Load(filepath.Join(dir, "config"))
	if err != nil {
		return err
	}

	nsn.PeerID = cfg.Identity.PeerID

	if err := nsn.waitOnAPI(); err != nil {
		return err
	}

	return nil
}

func (n *NetnsNode) waitOnAPI() error {
	for i := 0; i < 50; i++ {
		err := n.tryAPICheck()
		if err == nil {
			return nil
		}
		stump.VLog("temp error waiting on API: ", err)
		time.Sleep(time.Millisecond * 400)
	}
	return fmt.Errorf("node %s failed to come online in given time period", n.GetPeerID())
}

func (n *NetnsNode) tryAPICheck() error {
	addr, err := n.APIAddr()
	if err != nil {
		return err
	}

	stump.VLog("checking api addresss at: ", addr)
	resp, err := n.RunCmd("curl", "http://"+addr+"/api/v0/id")
	if err != nil {
		return err
	}

	out := make(map[string]interface{})
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
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

func (n *NetnsNode) Init() error {
	err := os.MkdirAll(n.Dir, 0777)
	if err != nil {
		return err
	}

	cmd := exec.Command("ipfs", "init", "-b=1024")
	cmd.Env, err = n.envForDaemon()
	if err != nil {
		return err
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(out))
	}

	return nil
}

// Shell sets up environment variables for a new shell to more easily
// control the given daemon
func (n *NetnsNode) Shell() error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return fmt.Errorf("couldnt find shell!")
	}

	nenvs := []string{"IPFS_PATH=" + n.Dir}

	nodes, err := LoadNodes()
	if err != nil {
		return err
	}

	for i, n := range nodes {
		peerid := n.GetPeerID()
		if peerid == "" {
			return fmt.Errorf("failed to check peerID")
		}

		nenvs = append(nenvs, fmt.Sprintf("NODE%d=%s", i, peerid))
	}
	nenvs = append(os.Environ(), nenvs...)

	return syscall.Exec("ip", []string{"ip", "netns", "exec", n.Name, shell}, nenvs)
}

func (n *NetnsNode) RunCmd(args ...string) (string, error) {
	baseargs := []string{"ip", "netns", "exec", n.Name}
	baseargs = append(baseargs, args...)
	cmd := exec.Command(baseargs[0], baseargs[1:]...)

	var err error
	cmd.Env, err = n.envForDaemon()
	if err != nil {
		return "", err
	}

	outbuf := new(bytes.Buffer)
	errbuf := new(bytes.Buffer)
	cmd.Stdout = outbuf
	cmd.Stderr = errbuf

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s: %s %s", err, outbuf.String(), errbuf.String())
	}

	return outbuf.String(), nil
}

func (n *NetnsNode) GetAttr(attr string) (string, error) {
	switch attr {
	case attrId:
		return n.GetPeerID(), nil
	case attrPath:
		return n.Dir, nil
	case attrBwIn:
		bw, err := GetBW(n)
		if err != nil {
			return "", err
		}
		return fmt.Sprint(bw.TotalIn), nil
	case attrBwOut:
		bw, err := GetBW(n)
		if err != nil {
			return "", err
		}
		return fmt.Sprint(bw.TotalOut), nil
	default:
		return "", errors.New("unrecognized attribute: " + attr)
	}
}
