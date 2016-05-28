package iptbutil

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"

	serial "github.com/ipfs/go-ipfs/repo/fsrepo/serialize"

	manet "gx/ipfs/QmUBa4w6CbHJUMeGJPDiMEDWsM93xToK1fTnFXnrC8Hksw/go-multiaddr-net"
	ma "gx/ipfs/QmYzDkkgAEmrcNzFCiYo6L1dTX4EAG1gZkbtdbd9trL4vd/go-multiaddr"
)

type LocalNode struct {
	Dir    string
	PeerID string
}

func (n *LocalNode) GetPeerID() string {
	return n.PeerID
}

func (n *LocalNode) String() string {
	return n.PeerID
}

// Shell sets up environment variables for a new shell to more easily
// control the given daemon
func (n *LocalNode) Shell() error {
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

	return syscall.Exec(shell, []string{shell}, nenvs)
}

func (n *LocalNode) RunCmd(args ...string) (string, error) {
	dir := n.Dir
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = []string{"IPFS_PATH=" + dir}
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, string(out))
	}

	return string(out), nil
}

func (n *LocalNode) APIAddr() (string, error) {
	dir := n.Dir

	addrb, err := ioutil.ReadFile(path.Join(dir, "api"))
	if err != nil {
		return "", err
	}

	maddr, err := ma.NewMultiaddr(string(addrb))
	if err != nil {
		fmt.Println("error parsing multiaddr: ", err)
		return "", err
	}

	_, addr, err := manet.DialArgs(maddr)
	if err != nil {
		fmt.Println("error on multiaddr dialargs: ", err)
		return "", err
	}
	return addr, nil
}

func (n *LocalNode) envForDaemon() ([]string, error) {
	envs := os.Environ()
	dir := n.Dir
	npath := "IPFS_PATH=" + dir
	for i, e := range envs {
		p := strings.Split(e, "=")
		if p[0] == "IPFS_PATH" {
			envs[i] = npath
			return envs, nil
		}
	}

	return append(envs, npath), nil
}

func (n *LocalNode) Start() error {
	dir := n.Dir
	cmd := exec.Command("ipfs", "daemon")
	cmd.Dir = dir

	var err error
	cmd.Env, err = n.envForDaemon()
	if err != nil {
		return err
	}

	setupOpt(cmd)

	stdout, err := os.Create(path.Join(dir, "daemon.stdout"))
	if err != nil {
		return err
	}

	stderr, err := os.Create(path.Join(dir, "daemon.stderr"))
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
	err = ioutil.WriteFile(path.Join(dir, "daemon.pid"), []byte(fmt.Sprint(pid)), 0666)
	if err != nil {
		return err
	}

	// Make sure node 0 is up before starting the rest so
	// bootstrapping works properly
	cfg, err := serial.Load(path.Join(dir, "config"))
	if err != nil {
		return err
	}

	n.PeerID = cfg.Identity.PeerID

	err = waitOnAPI(n)
	if err != nil {
		return err
	}

	return nil
}

func (n *LocalNode) getPID() (int, error) {
	b, err := ioutil.ReadFile(path.Join(n.Dir, "daemon.pid"))
	if err != nil {
		return -1, err
	}

	return strconv.Atoi(string(b))
}

func (n *LocalNode) Kill() error {
	pid, err := n.getPID()
	if err != nil {
		return fmt.Errorf("error killing daemon %s: %s", n.Dir, err)
	}

	p, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("error killing daemon %s: %s", n.Dir, err)
	}
	err = p.Kill()
	if err != nil {
		return fmt.Errorf("error killing daemon %s: %s\n", n.Dir, err)
	}

	p.Wait()

	err = os.Remove(path.Join(n.Dir, "daemon.pid"))
	if err != nil {
		return fmt.Errorf("error removing pid file for daemon at %s: %s\n", n.Dir, err)
	}

	return nil
}

func (n *LocalNode) GetAttr(attr string) (string, error) {
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
