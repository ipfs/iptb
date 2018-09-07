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
		cli.BoolFlag{
			Name:   "terminator",
			Hidden: true,
		},
		cli.BoolFlag{
			Name:   "stdin",
			Hidden: true,
		},
	},
	Before: func(c *cli.Context) error {
		if c.NArg() == 0 {
			return c.Set("stdin", "true")
		}
		if present := isTerminatorPresent(c); present {
			return c.Set("terminator", "true")
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
		var terminatorPresent bool
		if c.IsSet("stdin") {
			terminatorPresent = false
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				tokens := strings.Fields(scanner.Text())
				args = append(args, tokens)
			}
		} else {
			terminatorPresent = c.IsSet("terminator")
			cArgsStr := make([]string, c.NArg())
			for i, arg := range c.Args() {
				cArgsStr[i] = arg
			}
			args = append(args, cArgsStr)
		}

		ranges := make([][]int, len(args))
		runCmds := make([]outputFunc, len(args))
		for i, cmd := range args {
			nodeRange, tokens := parseCommand(cmd, terminatorPresent)
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
