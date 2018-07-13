package iptbutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	config "github.com/ipfs/go-ipfs/repo/config"
	serial "github.com/ipfs/go-ipfs/repo/fsrepo/serialize"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

// JavaScriptNode is a machine-local IPFS node controlled by IPTB
type JavaScriptNode struct {
	Dir    string
	PeerID string
}

// assert JavaScriptNode satisfies the IpfsNode interface
var _ IpfsNode = (*JavaScriptNode)(nil)

func (n *JavaScriptNode) Init() error {
	err := os.MkdirAll(n.Dir, 0777)
	if err != nil {
		return err
	}

	cmd := exec.Command("jsipfs", "init", "-b=1024")
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

func (n *JavaScriptNode) GetPeerID() string {
	return n.PeerID
}

func (n *JavaScriptNode) String() string {
	return n.PeerID
}

// Shell sets up environment variables for a new shell to more easily
// control the given daemon
func (n *JavaScriptNode) Shell() error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return fmt.Errorf("couldnt find shell!")
	}

	existingIpfsPath := os.Getenv("IPFS_PATH")

	if existingIpfsPath == n.Dir {
		return errors.New("You are already using the selected node")
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

func (n *JavaScriptNode) RunCmd(args ...string) (string, error) {
	spew.Dump(args)

	args = append([]string{"-c"}, args...)
	spew.Dump("bash", args)
	cmd := exec.Command("bash", args[0:]...)
	// cmd := exec.Command(args[0], args[1:]...)

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

func (n *JavaScriptNode) APIAddr() (string, error) {
	dir := n.Dir

	addrb, err := ioutil.ReadFile(filepath.Join(dir, "api"))
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

func (n *JavaScriptNode) envForDaemon() ([]string, error) {
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

func (n *JavaScriptNode) Start(args []string) error {
	alive, err := n.isAlive()
	if err != nil {
		return err
	}

	if alive {
		return fmt.Errorf("node is already running")
	}

	dir := n.Dir
	dargs := append([]string{"daemon"}, args...)
	cmd := exec.Command("jsipfs", dargs...)
	cmd.Dir = dir

	cmd.Env, err = n.envForDaemon()
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

	n.PeerID = cfg.Identity.PeerID

	err = waitOnAPI(n)
	if err != nil {
		return err
	}

	return nil
}

func (n *JavaScriptNode) getPID() (int, error) {
	b, err := ioutil.ReadFile(filepath.Join(n.Dir, "daemon.pid"))
	if err != nil {
		return -1, err
	}

	return strconv.Atoi(string(b))
}

func (n *JavaScriptNode) isAlive() (bool, error) {
	pid, err := n.getPID()
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, nil
	}

	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true, nil
	}
	return false, nil
}

func (n *JavaScriptNode) Kill() error {
	pid, err := n.getPID()
	if err != nil {
		return fmt.Errorf("error killing daemon %s: %s", n.Dir, err)
	}

	p, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("error killing daemon %s: %s", n.Dir, err)
	}

	waitch := make(chan struct{}, 1)
	go func() {
		p.Wait() //TODO: pass return state
		waitch <- struct{}{}
	}()

	defer func() {
		err := os.Remove(filepath.Join(n.Dir, "daemon.pid"))
		if err != nil && !os.IsNotExist(err) {
			panic(fmt.Errorf("error removing pid file for daemon at %s: %s\n", n.Dir, err))
		}
	}()

	if err := n.signalAndWait(p, waitch, syscall.SIGTERM, 1*time.Second); err != ErrTimeout {
		return err
	}

	if err := n.signalAndWait(p, waitch, syscall.SIGTERM, 2*time.Second); err != ErrTimeout {
		return err
	}

	if err := n.signalAndWait(p, waitch, syscall.SIGQUIT, 5*time.Second); err != ErrTimeout {
		return err
	}

	if err := n.signalAndWait(p, waitch, syscall.SIGKILL, 5*time.Second); err != ErrTimeout {
		return err
	}

	for {
		err := p.Signal(syscall.Signal(0))
		if err != nil {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}

	return nil
}

func (n *JavaScriptNode) signalAndWait(p *os.Process, waitch <-chan struct{}, signal os.Signal, t time.Duration) error {
	err := p.Signal(signal)
	if err != nil {
		return fmt.Errorf("error killing daemon %s: %s\n", n.Dir, err)
	}

	select {
	case <-waitch:
		return nil
	case <-time.After(t):
		return ErrTimeout
	}
}

func (n *JavaScriptNode) GetAttr(attr string) (string, error) {
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

func (n *JavaScriptNode) GetConfig() (*config.Config, error) {
	return serial.Load(filepath.Join(n.Dir, "config"))
}

func (n *JavaScriptNode) WriteConfig(c *config.Config) error {
	return serial.WriteConfigFile(filepath.Join(n.Dir, "config"), c)
}

func (n *JavaScriptNode) SetAttr(name, val string) error {
	return fmt.Errorf("no atttributes to set")
}

func (n *JavaScriptNode) StdoutReader() (io.ReadCloser, error) {
	return n.readerFor("daemon.stdout")
}

func (n *JavaScriptNode) StderrReader() (io.ReadCloser, error) {
	return n.readerFor("daemon.stderr")
}

func (n *JavaScriptNode) readerFor(file string) (io.ReadCloser, error) {
	f, err := os.OpenFile(filepath.Join(n.Dir, file), os.O_RDONLY, 0)
	return f, err
}
