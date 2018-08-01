package commands

import (
	"context"
	"path"
	"time"

	"github.com/pkg/errors"
	cli "github.com/urfave/cli"

	"github.com/ipfs/iptb/testbed"
)

var ConnectCmd = cli.Command{
	Category:  "CORE",
	Name:      "connect",
	Usage:     "connect nodes together",
	ArgsUsage: "<nodes> <nodes>",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "timeout",
			Usage: "timeout on the command",
			Value: "30s",
		},
	},
	Action: func(c *cli.Context) error {
		flagRoot := c.GlobalString("IPTB_ROOT")
		flagTestbed := c.GlobalString("bench")
		flagTimeout := c.String("timeout")

		timeout, err := time.ParseDuration(flagTimeout)
		if err != nil {
			return err
		}

		if c.NArg() != 2 {
			return NewUsageError("connet accepts exactly 2 arguments")
		}

		tb := testbed.NewTestbed(path.Join(flagRoot, "benches", flagTestbed))
		args := c.Args()

		from, err := parseRange(args[0])
		if err != nil {
			return err
		}

		to, err := parseRange(args[1])
		if err != nil {
			return err
		}

		nodes, err := tb.Nodes()
		if err != nil {
			return err
		}

		var results []Result
		for _, f := range from {
			for _, t := range to {
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()

				err = nodes[f].Connect(ctx, nodes[t])

				results = append(results, Result{
					Node:   f,
					Output: nil,
					Error:  errors.Wrapf(err, "node[%d] => node[%d]", f, t),
				})
			}
		}

		return buildReport(results)
	},
}
