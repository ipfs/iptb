package router

import (
	"os"
	"path"
	"path/filepath"

	lock "github.com/nightlyone/lockfile"
	iptbu "github.com/whyrusleeping/iptb/util"
)

func routerDir() (string, error) {
	res, err := iptbu.TestBedDir()
	if err != nil {
		return "", err
	}
	return path.Join(res, "router"), err
}

func lockFile() (lock.Lockfile, error) {
	dir, err := routerDir()
	os.Mkdir(dir, 0777)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path.Join(dir, "lock.pid"))
	if err != nil {
		return "", err
	}
	return lock.New(abs)
}
