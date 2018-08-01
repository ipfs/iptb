package commands

import (
	"context"
	"path"

	cli "github.com/urfave/cli"

	"github.com/ipfs/iptb/testbed"
)

var BenchCmd = cli.Command{
	Name:  "bench",
	Usage: "create, remove, list bench setups",
	Subcommands: []cli.Command{
		BenchCreateCmd,
	},
}

var BenchCreateCmd = cli.Command{
	Name:      "create",
	Usage:     "create test bench",
	ArgsUsage: "--type <type>",
	Flags: []cli.Flag{
		cli.IntFlag{
			Name:  "count",
			Usage: "number of nodes to initialize",
			Value: 1,
		},
		cli.BoolFlag{
			Name:  "force",
			Usage: "force overwrite of existing nodespecs",
		},
		cli.StringFlag{
			Name:  "type",
			Usage: "kind of nodes to initialize",
			Value: "localipfs",
		},
		cli.StringSliceFlag{
			Name:  "attr",
			Usage: "specify addition attributes for nodespecs",
		},
		cli.BoolFlag{
			Name:  "init",
			Usage: "initialize after creation (like calling `init` after create)",
		},
	},
	Action: func(c *cli.Context) error {
		flagRoot := c.GlobalString("IPTB_ROOT")
		flagTestbed := c.GlobalString("bench")
		flagType := c.String("type")
		flagInit := c.Bool("init")
		flagCount := c.Int("count")
		flagForce := c.Bool("force")
		flagAttrs := c.StringSlice("attr")

		attrs := parseAttrSlice(flagAttrs)
		tb := testbed.NewTestbed(path.Join(flagRoot, "benches", flagTestbed))

		if err := testbed.AlreadyInitCheck(tb.Dir(), flagForce); err != nil {
			return err
		}

		specs, err := testbed.BuildSpecs(tb.Dir(), flagCount, flagType, attrs)
		if err != nil {
			return err
		}

		if err := testbed.WriteNodeSpecs(tb.Dir(), specs); err != nil {
			return err
		}

		if flagInit {
			nodes, err := tb.Nodes()
			if err != nil {
				return err
			}

			for _, n := range nodes {
				if _, err := n.Init(context.TODO()); err != nil {
					return err
				}
			}
		}

		return nil
	},
}
