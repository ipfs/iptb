package commands

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	cli "github.com/urfave/cli"

	"github.com/ipfs/iptb/testbed"
	"github.com/ipfs/iptb/testbed/interfaces"
	"github.com/mattn/go-shellwords"
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

		var reader io.Reader
		if c.IsSet("stdin") {
			reader = bufio.NewReader(os.Stdin)
		} else {
			var builder strings.Builder
			if c.IsSet("terminator") {
				builder.WriteString("-- ")
			}
			for i, arg := range c.Args() {
				builder.WriteString(strconv.Quote(arg))
				if i != c.NArg()-1 {
					builder.WriteString(" ")
				}
			}
			reader = strings.NewReader(builder.String())
		}

		var args [][]string
		scanner := bufio.NewScanner(reader)
		line := 1
		for scanner.Scan() {
			tokens, err := shellwords.Parse(scanner.Text())
			if err != nil {
				return fmt.Errorf("parser error on line %d: %s", line, err)
			}
			if strings.HasPrefix(tokens[0], "#") {
				continue
			}
			args = append(args, tokens)
			line++
		}

		ranges := make([][]int, len(args))
		runCmds := make([]outputFunc, len(args))
		for i, cmd := range args {
			nodeRange, tokens := parseCommand(cmd, false)
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
