package router

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/codegangsta/cli"
)

func CStart(c *cli.Context) error {
	cmd := exec.Command(os.Args[0], "router", "run")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Setpgid = true

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Start()
}
func CStop(c *cli.Context) error {
	l, err := lockFile()
	if err != nil {
		return err
	}
	p, err := l.GetOwner()
	fmt.Printf("Killing %d\n", p.Pid)

	if err != nil {
		return err
	}

	return p.Kill()
}
func CRun(c *cli.Context) error {
	l, err := lockFile()
	if err != nil {
		return err
	}
	err = l.TryLock()
	if err != nil {
		return err
	}

	time.Sleep(100000 * time.Second)
	return nil

}
func CLink(c *cli.Context) error {
	return nil

}
