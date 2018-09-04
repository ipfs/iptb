package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
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

		nodeRange, args := parseCommand(c.Args(), c.IsSet("terminator"))

		if nodeRange == "" {
			nodeRange = fmt.Sprintf("[0-%d]", len(nodes)-1)
		}

		list, err := parseRange(nodeRange)
		if err != nil {
			return fmt.Errorf("could not parse node range %s", nodeRange)
		}

		if c.IsSet("cmdFile") {
			cmdFile, err := os.Open(c.String("cmdFile"))
			if err != nil {
				return err
			}
			defer cmdFile.Close()

			reg := regexp.MustCompile(`(?:(?P<nodeRange>.*?): *)(?P<cmd>[^\\z]+)`)
			scanner := bufio.NewScanner(cmdFile)
			line := 0
			var ranges [][]int
			var runCmds []outputFunc
			for scanner.Scan() {
				submatch := reg.FindStringSubmatch(scanner.Text())
				if len(submatch) != 3 {
					return fmt.Errorf("could not parse line %d of input file", line)
				}
				matches := make(map[string]string)
				for i, name := range reg.SubexpNames() {
					if i != 0 && name != "" {
						matches[name] = submatch[i]
					}
				}

				nodeRange := matches["nodeRange"]
				if nodeRange == "" {
					nodeRange = fmt.Sprintf("[0-%d]", len(nodes)-1)
				}

				list, err := parseRange(nodeRange)
				if err != nil {
					return fmt.Errorf("parse error on line %d: %s", line, err)
				}
				ranges = append(ranges, list)

				tokens := strings.Fields(matches["cmd"])
				runCmd := func(node testbedi.Core) (testbedi.Output, error) {
					return node.RunCmd(context.Background(), nil, tokens...)
				}
				runCmds = append(runCmds, runCmd)

				line++
			}

			results, err := mapListWithOutput(ranges, nodes, runCmds)
			if err != nil {
				return err
			}
			return buildReport(results)
		}

		runCmd := func(node testbedi.Core) (testbedi.Output, error) {
			return node.RunCmd(context.Background(), nil, args...)
		}

		results, err := mapWithOutput(list, nodes, runCmd)
		if err != nil {
			return err
		}

		return buildReport(results)
	},
}
