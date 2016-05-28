package iptbutil

type IpfsNode interface {
	Kill() error
	Start() error
	APIAddr() (string, error)
	GetPeerID() string
	RunCmd(args ...string) (string, error)
	Shell() error
	String() string
}
