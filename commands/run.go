package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	cli "github.com/urfave/cli"

	"github.com/ipfs/iptb/testbed"
	"github.com/ipfs/iptb/testbed/interfaces"
)

var RunCmd = cli.Command{
	Category:  "CORE",
	Name:      "run",
	Usage:     "run command on specified nodes (or all)",
	ArgsUsage: "[nodes] -- <command...>",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "cmdFile",
			Usage: "File containing list of commands to run asynchronously on nodes.",
		},
		cli.BoolFlag{
			Name:   "terminator",
			Hidden: true,
		},
	},
	Before: func(c *cli.Context) error {
		if !c.IsSet("cmdFile") {
			if present := isTerminatorPresent(c); present {
				return c.Set("terminator", "true")
			}
		}
		return nil
	},
	Action: func(c *cli.Context) error {
		flagRoot := c.GlobalString("IPTB_ROOT")
		flagTestbed := c.GlobalString("testbed")

		tb := testbed.NewTestbed(path.Join(flagRoot, "testbeds", flagTestbed))
		nodes, err := tb.Nodes()
		if err != nil {
			return err
		}

		var args [][]string
		var terminatorPresent []bool
		if c.IsSet("cmdFile") {
			cmdFile, err := os.Open(c.String("cmdFile"))
			if err != nil {
				return err
			}
			defer cmdFile.Close()

			scanner := bufio.NewScanner(cmdFile)
			for scanner.Scan() {
				tokens := strings.Fields(scanner.Text())
				term := tokens[0] == "--"
				if term {
					tokens = tokens[1:]
				}
				terminatorPresent = append(terminatorPresent, term)
				args = append(args, tokens)
			}
		} else {
			cArgsStr := make([]string, c.NArg())
			for i, arg := range c.Args() {
				cArgsStr[i] = arg
			}
			args = append(args, cArgsStr)
			terminatorPresent = append(terminatorPresent, c.IsSet("terminator"))
		}

		ranges := make([][]int, len(args))
		runCmds := make([]outputFunc, len(args))
		for i, cmd := range args {
			nodeRange, tokens := parseCommand(cmd, terminatorPresent[i])
			if nodeRange == "" {
				nodeRange = fmt.Sprintf("[0-%d]", len(nodes)-1)
			}
			list, err := parseRange(nodeRange)
			if err != nil {
				return fmt.Errorf("could not parse node range %s", nodeRange)
			}
			ranges[i] = list

			runCmd := func(node testbedi.Core) (testbedi.Output, error) {
				return node.RunCmd(context.Background(), nil, tokens...)
			}
			runCmds[i] = runCmd
		}

		results, err := mapListWithOutput(ranges, nodes, runCmds)
		return buildReport(results)
	},
}
