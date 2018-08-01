package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	cli "github.com/urfave/cli"

	"github.com/ipfs/iptb/commands"
	"github.com/ipfs/iptb/plugins/ipfs/docker"
	"github.com/ipfs/iptb/plugins/ipfs/local"
	"github.com/ipfs/iptb/testbed"
)

func init() {
	plglocalipfs := testbed.IptbPlugin{
		From:        "<builtin>",
		NewNode:     pluginlocalipfs.NewNode,
		PluginName:  pluginlocalipfs.PluginName,
		GetAttrList: pluginlocalipfs.GetAttrList,
		GetAttrDesc: pluginlocalipfs.GetAttrDesc,
		BuiltIn:     true,
	}

	plgdockeripfs := testbed.IptbPlugin{
		From:        "<builtin>",
		NewNode:     plugindockeripfs.NewNode,
		PluginName:  plugindockeripfs.PluginName,
		GetAttrList: plugindockeripfs.GetAttrList,
		GetAttrDesc: plugindockeripfs.GetAttrDesc,
		BuiltIn:     true,
	}

	testbed.RegisterPlugin(plgdockeripfs, false)
	testbed.RegisterPlugin(plglocalipfs, false)
}

func loadPlugins(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	plugs, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, f := range plugs {
		plg, err := testbed.LoadPlugin(path.Join(dir, f.Name()))

		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			continue
		}

		overloaded, err := testbed.RegisterPlugin(*plg, false)
		if overloaded {
			fmt.Fprintf(os.Stderr, "overriding built in plugin %s with %s\n", plg.PluginName, path.Join(dir, f.Name()))
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Usage = "iptb is a tool for managing test clusters of ipfs nodes"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "bench",
			Value:  "default",
			EnvVar: "IPTB_BENCH",
			Usage:  "Name of bench to use under IPTB_ROOT",
		},
		cli.StringFlag{
			Name:   "IPTB_ROOT",
			EnvVar: "IPTB_ROOT",
			Hidden: true,
		},
	}
	app.Before = func(c *cli.Context) error {
		flagRoot := c.GlobalString("IPTB_ROOT")

		if len(flagRoot) != 0 {
			return nil
		}

		home := os.Getenv("HOME")
		if len(home) == 0 {
			return fmt.Errorf("environment variable HOME not set")
		}

		root := path.Join(home, "testbench")

		c.Set("IPTB_ROOT", root)

		return loadPlugins(path.Join(root, "plugins"))
	}
	app.Commands = []cli.Command{
		commands.FastCmd,
		commands.BenchCmd,

		commands.InitCmd,
		commands.StartCmd,
		commands.StopCmd,
		commands.RestartCmd,
		commands.RunCmd,
		commands.ConnectCmd,
		commands.ShellCmd,

		commands.AttrCmd,

		commands.LogsCmd,
		commands.EventsCmd,
		commands.MetricCmd,
	}

	// https://github.com/urfave/cli/issues/736
	// Currently unreleased
	/*
		app.ExitErrHandler = func(c *cli.Context, err error) {
			switch err.(type) {
			case *commands.UsageError:
				fmt.Fprintf(c.App.ErrWriter, "%s\n\n", err)
				cli.ShowCommandHelpAndExit(c, c.Command.Name, 1)
			default:
				cli.HandleExitCoder(err)
			}
		}
	*/

	app.ErrWriter = os.Stderr
	app.Writer = os.Stdout

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(app.ErrWriter, "%s\n", err)
		os.Exit(1)
	}
}
