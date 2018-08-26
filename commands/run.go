package commands

import (
	"context"
	"fmt"
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
			Name:  "stats",
			Usage: "Output statistics on the command execution",
		},
	},
	Before: func(c *cli.Context) error {
		if present := isTerminatorPresent(c); present {
			return c.Set("terminator", "true")
		}

		return nil
	},
	Action: func(c *cli.Context) error {
		flagRoot := c.GlobalString("IPTB_ROOT")
		flagTestbed := c.GlobalString("testbed")
		flagStats := c.Bool("stats")

		tb := testbed.NewTestbed(path.Join(flagRoot, "testbeds", flagTestbed))
		nodes, err := tb.Nodes()
		if err != nil {
			return err
		}

		nodeRange, args := parseCommand(c.Args(), c.IsSet("terminator"))

		if nodeRange == "" {
			nodeRange = fmt.Sprintf("[0-%d]", len(nodes)-1)
		}

		list, err := parseRange(nodeRange)
		if err != nil {
			return fmt.Errorf("could not parse node range %s", nodeRange)
		}

		runCmd := func(node testbedi.Core) (testbedi.Output, error) {
			return node.RunCmd(context.Background(), nil, args...)
		}

		results, err := mapWithOutput(list, nodes, runCmd)
		if err != nil {
			return err
		}

		return buildReport(results, strings.Join(args[:], " "), flagStats)
	},
}
