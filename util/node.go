package iptbutil

import (
	"github.com/ipfs/go-ipfs/repo/config"
)

type IpfsNode interface {
	Init() error
	Kill() error
	Start() error
	APIAddr() (string, error)
	GetPeerID() string
	RunCmd(args ...string) (string, error)
	Shell() error
	String() string

	GetConfig() (*config.Config, error)
	WriteConfig(*config.Config) error
}
