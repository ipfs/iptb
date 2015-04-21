package main

import (
	"flag"
	"fmt"
	serial "github.com/ipfs/go-ipfs/repo/fsrepo/serialize"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"syscall"
)

// GetNumNodes returns the number of testbed nodes configured in the testbed directory
func GetNumNodes() int {
	for i := 0; i < 2000; i++ {
		_, err := os.Stat(IpfsDirN(i))
		if os.IsNotExist(err) {
			return i
		}
	}
	panic("i dont know whats going on")
}

func TestBedDir() string {
	tbd := os.Getenv("IPTB_ROOT")
	if len(tbd) != 0 {
		return tbd
	}

	home := os.Getenv("HOME")
	if len(home) == 0 {
		panic("could not find home")
	}

	return path.Join(home, "testbed")
}

func IpfsDirN(n int) string {
	return path.Join(TestBedDir(), fmt.Sprint(n))
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

func IpfsInit(n int) error {
	p := IpfsDirN(0)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		if !YesNoPrompt("testbed nodes already exist, overwrite? [y/n]") {
			return nil
		}
		err := os.RemoveAll(TestBedDir())
		if err != nil {
			return err
		}
	}
	wait := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wait.Add(1)
		go func(v int) {
			defer wait.Done()
			dir := IpfsDirN(v)
			err := os.MkdirAll(dir, 0777)
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

	// Now setup configs to bootstrap to eachother

	// '0' node is the bootstrap node
	cfgpath := path.Join(IpfsDirN(0), "config")
	bcfg, err := serial.Load(cfgpath)
	if err != nil {
		return err
	}
	bcfg.Bootstrap = nil
	bcfg.Addresses.Swarm = []string{"/ip4/127.0.0.1/tcp/4002"}
	bcfg.Addresses.API = "/ip4/127.0.0.1/tcp/5002"
	bcfg.Addresses.Gateway = ""
	err = serial.WriteConfigFile(cfgpath, bcfg)
	if err != nil {
		return err
	}

	for i := 1; i < n; i++ {
		cfgpath := path.Join(IpfsDirN(i), "config")
		cfg, err := serial.Load(cfgpath)
		if err != nil {
			return err
		}

		cfg.Bootstrap = []string{fmt.Sprintf("%s/ipfs/%s", bcfg.Addresses.Swarm[0], bcfg.Identity.PeerID)}
		cfg.Addresses.Gateway = ""
		cfg.Addresses.Swarm = []string{
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 4002+i),
		}
		cfg.Addresses.API = fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 5002+i)
		err = serial.WriteConfigFile(cfgpath, cfg)
		if err != nil {
			return err
		}
	}
	return nil
}

func IpfsPidOf(n int) (int, error) {
	dir := IpfsDirN(n)
	b, err := ioutil.ReadFile(path.Join(dir, "daemon.pid"))
	if err != nil {
		return -1, err
	}

	return strconv.Atoi(string(b))
}

func IpfsKill() error {
	n := GetNumNodes()
	for i := 0; i < n; i++ {
		pid, err := IpfsPidOf(i)
		if err != nil {
			return err
		}

		p, err := os.FindProcess(pid)
		if err != nil {
			fmt.Printf("error killing daemon %d: %s\n", i, err)
			continue
		}
		err = p.Kill()
		if err != nil {
			fmt.Printf("error killing daemon %d: %s\n", i, err)
			continue
		}

		p.Wait()
	}
	return nil
}

func IpfsStart() error {
	n := GetNumNodes()
	for i := 0; i < n; i++ {
		dir := IpfsDirN(i)
		cmd := exec.Command("ipfs", "daemon")
		cmd.Dir = dir
		cmd.Env = []string{"IPFS_PATH=" + dir}

		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

		err := cmd.Start()
		if err != nil {
			return err
		}
		pid := cmd.Process.Pid

		fmt.Printf("Started daemon %d, pid = %d\n", i, pid)
		err = ioutil.WriteFile(path.Join(dir, "daemon.pid"), []byte(fmt.Sprint(pid)), 0666)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetPeerID(n int) (string, error) {
	cfg, err := serial.Load(path.Join(IpfsDirN(n), "config"))
	if err != nil {
		return "", err
	}
	return cfg.Identity.PeerID, nil
}

func IpfsShell(n int) error {
	dir := IpfsDirN(n)
	nenvs := []string{"IPFS_PATH=" + dir}

	nnodes := GetNumNodes()
	for i := 0; i < nnodes; i++ {
		peerid, err := GetPeerID(i)
		if err != nil {
			return err
		}
		nenvs = append(nenvs, fmt.Sprintf("NODE%d=%s", i, peerid))
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		return fmt.Errorf("couldnt find shell!")
	}
	nenvs = append(os.Environ(), nenvs...)

	return syscall.Exec(shell, []string{shell}, nenvs)
}

var helptext = `
Ipfs Testbed

commands:
	init -n=[number of nodes]
	    creates and initializes 'n' repos
	start 
	    starts up all testbed nodes
	stop 
	    kills all testbed nodes
	restart
	    kills, then restarts all testbed nodes

	shell [n]
	    execs your shell with environment variables set as follows:
	        IPFS_PATH - set to testbed node n's IPFS_PATH
	        NODE[x] - set to the peer ID of node x
`

func main() {
	count := flag.Int("n", 0, "number of ipfs nodes to initialize")
	flag.Usage = func() {
		fmt.Println(helptext)
	}

	flag.Parse()

	switch flag.Arg(0) {
	case "init":
		if *count == 0 {
			fmt.Printf("please specify number of nodes: '%s -n=10 init'\n", os.Args[0])
			return
		}
		err := IpfsInit(*count)
		if err != nil {
			fmt.Println("ipfs init err: ", err)
		}
	case "start":
		err := IpfsStart()
		if err != nil {
			fmt.Println("ipfs start err: ", err)
		}
	case "stop", "kill":
		err := IpfsKill()
		if err != nil {
			fmt.Println("ipfs kill err: ", err)
		}
	case "restart":
		err := IpfsKill()
		if err != nil {
			fmt.Println("ipfs kill err: ", err)
			return
		}
		err = IpfsStart()
		if err != nil {
			fmt.Println("ipfs start err: ", err)
		}
	case "shell":
		if len(flag.Args()) < 2 {
			fmt.Println("please specify which node you want a shell for")
		}
		n, err := strconv.Atoi(flag.Arg(1))
		if err != nil {
			fmt.Println("parse err: ", err)
			return
		}

		err = IpfsShell(n)
		if err != nil {
			fmt.Println("ipfs shell err: ", err)
		}

	default:
		fmt.Println("unrecognized command: ", flag.Arg(0))
	}
}
