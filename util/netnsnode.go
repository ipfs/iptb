package iptbutil

import (
	"os/exec"
	"runtime"

	"github.com/vishvananda/netns"
)

type NetnsNode struct {
	LocalNode
}

var _ IpfsNode = &NetnsNode{}

func (dn *NetnsNode) Start() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	orginal, err := netns.Get()
	if err != nil {
		return err
	}
	defer netns.Set(orginal)

	ns, err := netns.New()
	if err != nil {
		return err
	}
	err = netns.Set(ns)
	if err != nil {
		return err
	}

	err = exec.Command("ip", "link", "set", "up", "dev", "lo").Run()
	if err != nil {
		return err
	}

	return dn.LocalNode.Start()
}

func (dn *NetnsNode) APIAddr() (string, error) {
	return dn.LocalNode.APIAddr()
}

func (dn *NetnsNode) Kill() error {
	return dn.LocalNode.Kill()
}

func (dn *NetnsNode) String() string {
	return "netns:" + dn.PeerID
}

func (nn *NetnsNode) InNS(f func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	pid, err := nn.LocalNode.getPID()
	if err != nil {
		return err
	}
	orginal, err := netns.Get()
	if err != nil {
		return err
	}
	defer netns.Set(orginal)

	nodens, err := netns.GetFromPid(pid)
	if err != nil {
		return err
	}

	err = netns.Set(nodens)
	if err != nil {
		return err
	}
	return f()
}

func (nn *NetnsNode) RunCmd(args ...string) (string, error) {
	res := ""
	err := nn.InNS(func() error {
		var err error
		res, err = nn.LocalNode.RunCmd(args...)
		return err
	})
	return res, err
}

func (nn *NetnsNode) Shell() error {
	return nn.InNS(nn.LocalNode.Shell)
}

func (dn *NetnsNode) GetAttr(name string) (string, error) {
	return dn.LocalNode.GetAttr(name)
}

func (dn *NetnsNode) SetAttr(name, val string) error {
	return dn.LocalNode.SetAttr(name, val)
}
