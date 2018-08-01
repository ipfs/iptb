package pluginlocalipfs

import (
	"context"
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

	"github.com/ipfs/go-cid"
	config "github.com/ipfs/go-ipfs-config"
	serial "github.com/ipfs/go-ipfs-config/serialize"
	"github.com/ipfs/iptb/plugins/ipfs"
	"github.com/ipfs/iptb/testbed/interfaces"
	"github.com/ipfs/iptb/util"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
)

var errTimeout = errors.New("timeout")

var PluginName = "localipfs"

type Localipfs struct {
	dir        string
	peerid     *cid.Cid
	apiaddr    multiaddr.Multiaddr
	swarmaddrs []multiaddr.Multiaddr
	lt         ipfs.ListenType
	mdns       bool
}

func NewNode(dir string, attrs map[string]interface{}) (testbedi.Core, error) {
	if _, err := exec.LookPath("ipfs"); err != nil {
		return nil, err
	}

	lt := ipfs.LT_TCP
	mdns := false

	if v, ok := attrs["listentype"]; ok {
		ltstr, ok := v.(string)

		if !ok {
			return nil, fmt.Errorf("Attr `listentype` should be a string")
		}

		switch ltstr {
		case "":
			lt = ipfs.LT_TCP
			break
		case "ws":
			lt = ipfs.LT_WS
			break
		case "utp":
			lt = ipfs.LT_UTP
			break
		default:
			return nil, fmt.Errorf("Unsupported `listentype` %s", ltstr)
		}
	}

	if _, ok := attrs["mdns"]; ok {
		mdns = true
	}

	return &Localipfs{
		dir:  dir,
		lt:   lt,
		mdns: mdns,
	}, nil

}

func GetAttrList() []string {
	return ipfs.GetAttrList()
}

func GetAttrDesc(attr string) (string, error) {
	return ipfs.GetAttrDesc(attr)
}

func GetMetricList() []string {
	return ipfs.GetMetricList()
}

func GetMetricDesc(attr string) (string, error) {
	return ipfs.GetMetricDesc(attr)
}

/// TestbedNode Interface

func (l *Localipfs) Init(ctx context.Context, agrs ...string) (testbedi.Output, error) {
	agrs = append([]string{"ipfs", "init"}, agrs...)
	output, oerr := l.RunCmd(ctx, nil, agrs...)
	if oerr != nil {
		return nil, oerr
	}

	icfg, err := l.GetConfig()
	if err != nil {
		return nil, err
	}

	lcfg := icfg.(*config.Config)

	lcfg.Bootstrap = nil
	lcfg.Addresses.Swarm = []string{ipfs.SwarmAddr(l.lt, "127.0.0.1", 0)}
	lcfg.Addresses.API = fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 0)
	lcfg.Addresses.Gateway = ""
	lcfg.Discovery.MDNS.Enabled = l.mdns

	err = l.WriteConfig(lcfg)
	if err != nil {
		return nil, err
	}

	return output, oerr
}

func (l *Localipfs) Start(ctx context.Context, wait bool, args ...string) (testbedi.Output, error) {
	alive, err := l.isAlive()
	if err != nil {
		return nil, err
	}

	if alive {
		return nil, fmt.Errorf("node is already running")
	}

	dir := l.dir
	dargs := append([]string{"daemon"}, args...)
	cmd := exec.Command("ipfs", dargs...)
	cmd.Dir = dir

	cmd.Env, err = l.env()
	if err != nil {
		return nil, err
	}

	iptbutil.SetupOpt(cmd)

	stdout, err := os.Create(filepath.Join(dir, "daemon.stdout"))
	if err != nil {
		return nil, err
	}

	stderr, err := os.Create(filepath.Join(dir, "daemon.stderr"))
	if err != nil {
		return nil, err
	}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	pid := cmd.Process.Pid

	err = ioutil.WriteFile(filepath.Join(dir, "daemon.pid"), []byte(fmt.Sprint(pid)), 0666)
	if err != nil {
		return nil, err
	}

	return nil, ipfs.WaitOnAPI(l)
}

func (l *Localipfs) Stop(ctx context.Context, wait bool) error {
	pid, err := l.getPID()
	if err != nil {
		return fmt.Errorf("error killing daemon %s: %s", l.dir, err)
	}

	p, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("error killing daemon %s: %s", l.dir, err)
	}

	waitch := make(chan struct{}, 1)
	go func() {
		p.Wait() //TODO: pass return state
		waitch <- struct{}{}
	}()

	defer func() {
		err := os.Remove(filepath.Join(l.dir, "daemon.pid"))
		if err != nil && !os.IsNotExist(err) {
			panic(fmt.Errorf("error removing pid file for daemon at %s: %s", l.dir, err))
		}
	}()

	if err := l.signalAndWait(p, waitch, syscall.SIGTERM, 1*time.Second); err != errTimeout {
		return err
	}

	if err := l.signalAndWait(p, waitch, syscall.SIGTERM, 2*time.Second); err != errTimeout {
		return err
	}

	if err := l.signalAndWait(p, waitch, syscall.SIGQUIT, 5*time.Second); err != errTimeout {
		return err
	}

	if err := l.signalAndWait(p, waitch, syscall.SIGKILL, 5*time.Second); err != errTimeout {
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

func (l *Localipfs) RunCmd(ctx context.Context, stdin io.Reader, args ...string) (testbedi.Output, error) {
	env, err := l.env()

	if err != nil {
		return nil, fmt.Errorf("error getting env: %s", err)
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = env
	cmd.Stdin = stdin

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()

	stderrbytes, err := ioutil.ReadAll(stderr)
	if err != nil {
		return nil, err
	}

	stdoutbytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	exiterr := cmd.Wait()

	var exitcode = 0
	switch oerr := exiterr.(type) {
	case *exec.ExitError:
		if ctx.Err() == context.DeadlineExceeded {
			err = errors.Wrapf(oerr, "context deadline exceeded for command: %q", strings.Join(cmd.Args, " "))
		}

		exitcode = 1
	case nil:
		err = oerr
	}

	return iptbutil.NewOutput(args, stdoutbytes, stderrbytes, exitcode, err), nil
}

func (l *Localipfs) Connect(ctx context.Context, tbn testbedi.Core) error {
	swarmaddrs, err := tbn.SwarmAddrs()
	if err != nil {
		return err
	}

	_, err = l.RunCmd(ctx, nil, "ipfs", "swarm", "connect", swarmaddrs[0])

	return err
}

func (l *Localipfs) Shell(ctx context.Context, nodes []testbedi.Core) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return fmt.Errorf("no shell found")
	}

	if len(os.Getenv("IPFS_PATH")) != 0 {
		// If the users shell sets IPFS_PATH, it will just be overridden by the shell again
		return fmt.Errorf("shell has IPFS_PATH set, please unset before trying to use iptb shell")
	}

	nenvs, err := l.env()
	if err != nil {
		return err
	}

	// TODO(tperson): It would be great if we could guarantee that the shell
	// is using the same binary. However, the users shell may prepend anything
	// we change in the PATH

	for i, n := range nodes {
		peerid, err := n.PeerID()

		if err != nil {
			return err
		}

		nenvs = append(nenvs, fmt.Sprintf("NODE%d=%s", i, peerid))
	}

	return syscall.Exec(shell, []string{shell}, nenvs)
}

func (l *Localipfs) String() string {
	pcid, err := l.PeerID()
	if err != nil {
		return fmt.Sprintf("%s", l.Type())
	}
	return fmt.Sprintf("%s", pcid[0:12])
}

func (l *Localipfs) Infof(format string, args ...interface{}) {
	nformat := fmt.Sprintf("%s %s\n", l, format)
	fmt.Fprintf(os.Stdout, nformat, args...)
}

func (l *Localipfs) Errorf(format string, args ...interface{}) {
	nformat := fmt.Sprintf("%s %s\n", l, format)
	fmt.Fprintf(os.Stderr, nformat, args...)
}

func (l *Localipfs) APIAddr() (string, error) {
	if l.apiaddr != nil {
		return l.apiaddr.String(), nil
	}
	return ipfs.GetAPIAddrFromRepo(l.dir)
}

func (l *Localipfs) SwarmAddrs() ([]string, error) {
	var out []string
	if l.swarmaddrs != nil {
		for _, sa := range l.swarmaddrs {
			out = append(out, sa.String())
		}
		return out, nil
	}
	return ipfs.SwarmAddrs(l)
}

func (l *Localipfs) Dir() string {
	return l.dir
}

func (l *Localipfs) PeerID() (string, error) {
	if l.peerid != nil {
		return l.peerid.String(), nil
	}

	var err error
	l.peerid, err = ipfs.GetPeerID(l)

	return l.peerid.String(), err
}

/// Metric Interface

func (l *Localipfs) GetMetricList() []string {
	return GetMetricList()
}

func (l *Localipfs) GetMetricDesc(attr string) (string, error) {
	return GetMetricDesc(attr)
}

func (l *Localipfs) Metric(metric string) (string, error) {
	return ipfs.GetMetric(l, metric)
}

func (l *Localipfs) Heartbeat() (map[string]string, error) {
	return nil, nil
}

func (l *Localipfs) Events() (io.ReadCloser, error) {
	return ipfs.ReadLogs(l)
}

func (l *Localipfs) Logs() (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Attribute Interface

func (l *Localipfs) GetAttrList() []string {
	return GetAttrList()
}

func (l *Localipfs) GetAttrDesc(attr string) (string, error) {
	return GetAttrDesc(attr)
}

func (l *Localipfs) GetAttr(attr string) (string, error) {
	return ipfs.GetAttr(l, attr)
}

func (l *Localipfs) SetAttr(string, string) error {
	return fmt.Errorf("no attribute to set")
}

func (l *Localipfs) StderrReader() (io.ReadCloser, error) {
	return l.readerFor("daemon.stderr")
}

func (l *Localipfs) StdoutReader() (io.ReadCloser, error) {
	return l.readerFor("daemon.stdout")
}

func (l *Localipfs) GetConfig() (interface{}, error) {
	return serial.Load(filepath.Join(l.dir, "config"))
}

func (l *Localipfs) WriteConfig(cfg interface{}) error {
	return serial.WriteConfigFile(filepath.Join(l.dir, "config"), cfg)
}

func (l *Localipfs) Type() string {
	return "ipfs"
}

func (l *Localipfs) Deployment() string {
	return "local"
}

func (l *Localipfs) readerFor(file string) (io.ReadCloser, error) {
	return os.OpenFile(filepath.Join(l.dir, file), os.O_RDONLY, 0)
}

func (l *Localipfs) signalAndWait(p *os.Process, waitch <-chan struct{}, signal os.Signal, t time.Duration) error {
	err := p.Signal(signal)
	if err != nil {
		return fmt.Errorf("error killing daemon %s: %s", l.dir, err)
	}

	select {
	case <-waitch:
		return nil
	case <-time.After(t):
		return errTimeout
	}
}

func (l *Localipfs) getPID() (int, error) {
	b, err := ioutil.ReadFile(filepath.Join(l.dir, "daemon.pid"))
	if err != nil {
		return -1, err
	}

	return strconv.Atoi(string(b))
}

func (l *Localipfs) isAlive() (bool, error) {
	pid, err := l.getPID()
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

func (l *Localipfs) env() ([]string, error) {
	envs := os.Environ()
	ipfspath := "IPFS_PATH=" + l.dir

	for i, e := range envs {
		if strings.HasPrefix(e, "IPFS_PATH=") {
			envs[i] = ipfspath
			return envs, nil
		}
	}
	return append(envs, ipfspath), nil
}
