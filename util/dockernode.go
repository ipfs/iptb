package iptbutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/whyrusleeping/stump"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type DockerNode struct {
	ImageName string
	ID        string

	apiAddr string

	LocalNode
}

var _ IpfsNode = &DockerNode{}

func (dn *DockerNode) Start() error {
	cmd := exec.Command("docker", "run", "-d", "-v", dn.Dir+":/data/ipfs", dn.ImageName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(out))
	}

	id := bytes.TrimSpace(out)
	idfile := filepath.Join(dn.Dir, "dockerID")
	err = ioutil.WriteFile(idfile, id, 0664)
	if err != nil {
		return err
	}

	dn.ID = string(id)

	err = waitOnAPI(dn)
	if err != nil {
		return err
	}

	return nil
}

func (dn *DockerNode) setAPIAddr() error {
	internal, err := dn.LocalNode.APIAddr()
	if err != nil {
		return err
	}

	port := strings.Split(internal, ":")[1]

	dip, err := dn.getDockerIP()
	if err != nil {
		return err
	}

	dn.apiAddr = dip + ":" + port
	return nil
}

func (dn *DockerNode) APIAddr() (string, error) {
	if dn.apiAddr == "" {
		if err := dn.setAPIAddr(); err != nil {
			return "", err
		}
	}

	return dn.apiAddr, nil
}

func (dn *DockerNode) getDockerIP() (string, error) {
	cmd := exec.Command("docker", "inspect", dn.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, string(out))
	}

	var info []interface{}
	if err := json.Unmarshal(out, &info); err != nil {
		return "", err
	}

	if len(info) == 0 {
		return "", fmt.Errorf("got no inspect data")
	}

	cinfo := info[0].(map[string]interface{})
	netinfo := cinfo["NetworkSettings"].(map[string]interface{})
	return netinfo["IPAddress"].(string), nil
}

func (dn *DockerNode) Kill() error {
	out, err := exec.Command("docker", "kill", dn.ID).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(out))
	}

	return nil
}

func (dn *DockerNode) String() string {
	return "docker:" + dn.PeerID
}

func (dn *DockerNode) RunCmd(args ...string) (string, error) {
	if dn.ID == "" {
		return "", fmt.Errorf("no docker id set on node")
	}

	args = append([]string{"exec", "-ti", dn.ID}, args...)
	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin

	stump.VLog("running: ", cmd.Args)

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, string(out))
	}

	return string(out), nil
}

func (dn *DockerNode) Shell() error {
	nodes, err := LoadNodes()
	if err != nil {
		return err
	}

	nenvs := os.Environ()
	for i, n := range nodes {
		peerid := n.GetPeerID()
		if peerid == "" {
			return fmt.Errorf("failed to check peerID")
		}

		nenvs = append(nenvs, fmt.Sprintf("NODE%d=%s", i, peerid))
	}

	cmd := exec.Command("docker", "exec", "-ti", dn.ID, "/bin/bash")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
