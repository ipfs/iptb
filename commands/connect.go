package commands

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/pkg/errors"
	cli "github.com/urfave/cli"

	"github.com/ipfs/iptb/testbed"
	"github.com/ipfs/iptb/testbed/interfaces"
)

var ConnectCmd = cli.Command{
	Category:  "CORE",
	Name:      "connect",
	Usage:     "connect sets of nodes together (or all)",
	ArgsUsage: "[nodes] [nodes]",
	Description: `
The connect command allows for connecting sets of nodes together.

Every node listed in the first set, will try to connect to every node
listed in the second set.

There are three variants of the command. It can accept no arugments,
a single argument, or two arguments. The no argument and single argument
expands out to the two argument usage.

$ iptb connect             => iptb connect [0-C] [0-C]
$ iptb connect [n-m]       => iptb connect [n-m] [n-m]
$ iptb connect [n-m] [i-k]

Sets of nodes can be expressed in the following ways

INPUT         EXPANDED
0             0
[0]           0
[0-4]         0,1,2,3,4
[0,2-4]       0,2,3,4
[2-4,0]       2,3,4,0
[0,2,4]       0,2,4
`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "timeout",
			Usage: "timeout on the command",
			Value: "30s",
		},
	},
	Action: func(c *cli.Context) error {
		flagRoot := c.GlobalString("IPTB_ROOT")
		flagTestbed := c.GlobalString("testbed")
		flagTimeout := c.String("timeout")

		timeout, err := time.ParseDuration(flagTimeout)
		if err != nil {
			return err
		}

		tb := testbed.NewTestbed(path.Join(flagRoot, "testbeds", flagTestbed))
		var results []Result

		args := c.Args()

		switch c.NArg() {
		case 0:
			nodes, err := tb.Nodes()
			if err != nil {
				return err
			}

			fromto, err := parseRange(fmt.Sprintf("[0-%d]", len(nodes)-1))
			if err != nil {
				return err
			}

			results, err = connectNodes(tb, fromto, fromto, timeout)
			if err != nil {
				return err
			}
		case 1:
			fromto, err := parseRange(args[0])
			if err != nil {
				return err
			}

			results, err = connectNodes(tb, fromto, fromto, timeout)
			if err != nil {
				return err
			}
		case 2:
			from, err := parseRange(args[0])
			if err != nil {
				return err
			}

			to, err := parseRange(args[1])
			if err != nil {
				return err
			}

			results, err = connectNodes(tb, from, to, timeout)
			if err != nil {
				return err
			}
		default:
			return NewUsageError("connet accepts between 0 and 2 arguments")
		}

		return buildReport(results)
	},
}

func connectNodes(tb testbed.BasicTestbed, from, to []int, timeout time.Duration) ([]Result, error) {
	var results []Result
	nodes, err := tb.Nodes()

	if err != nil {
		return results, err
	}
	// synchronization variables
	var wg sync.WaitGroup
	var lk sync.Mutex

	// check if the list `to` is a valid range
	if err := validRange(to, len(nodes)); err != nil {
		return results, err
	}
	// check if the list `from` is a valid range
	if err := validRange(from, len(nodes)); err != nil {
		return results, err
	}

	for _, f := range from {
		for _, t := range to {
			if f == t {
				//Skip connecting a node with itself
				continue
			}
			wg.Add(1)
			go func(from, to int, nodeFrom, nodeTo testbedi.Core) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()

				err := nodeFrom.Connect(ctx, nodeTo)

				lk.Lock()
				defer lk.Unlock()
				results = append(results, Result{
					Node:   from,
					Output: nil,
					Error:  errors.Wrapf(err, "node[%d] => node[%d]", from, to),
				})

			}(f, t, nodes[f], nodes[t])
		}
	}

	wg.Wait()
	return results, nil
}
