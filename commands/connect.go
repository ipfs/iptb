package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	cli "github.com/urfave/cli"

	"github.com/ipfs/iptb/testbed"
)

// TODO:Add explanation in the description for topology flag
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
		cli.StringFlag{
			Name:  "topology",
			Usage: "specify a network topology file",
		},
	},
	Action: func(c *cli.Context) error {
		flagRoot := c.GlobalString("IPTB_ROOT")
		flagTestbed := c.GlobalString("testbed")
		flagTimeout := c.String("timeout")
		flagTopology := c.String("topology")

		timeout, err := time.ParseDuration(flagTimeout)
		if err != nil {
			return err
		}
		tb := testbed.NewTestbed(path.Join(flagRoot, "testbeds", flagTestbed))

		// Case Topoloogy is specified
		if len(flagTopology) != 0 {
			nodes, err := tb.Nodes()
			if err != nil {
				return err
			}
			topologyGraph, err := parseTopology(flagRoot, len(nodes))
			if err != nil {
				return err
			}
			for _, connectionRow := range topologyGraph {
				from := connectionRow[0]
				to := connectionRow[1:]
				err = connectNodes(tb, []int{from}, to, timeout)
				if err != nil {
					return err
				}
			}
			return nil
		}

		// Case range is specified
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

			return connectNodes(tb, fromto, fromto, timeout)
		case 1:
			fromto, err := parseRange(args[0])
			if err != nil {
				return err
			}

			return connectNodes(tb, fromto, fromto, timeout)
		case 2:
			from, err := parseRange(args[0])
			if err != nil {
				return err
			}

			to, err := parseRange(args[1])
			if err != nil {
				return err
			}

			return connectNodes(tb, from, to, timeout)
		default:
			return NewUsageError("connet accepts between 0 and 2 arguments")
		}
	},
}

func connectNodes(tb testbed.BasicTestbed, from, to []int, timeout time.Duration) error {
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
}

func parseTopology(fileDir string, numberOfNodes int) ([][]int, error) {

	// Scan Input file Line by Line //
	inFile, err := os.Open(fileDir)
	defer inFile.Close()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)

	// Store number of line to produce meaningful errors
	lineNumber := 1
	// Topology graph implemented as an Adjacency Matrix DS
	// This intermediate variable it could be terminated to increase peformance
	// This would decrease code readability.
	var topology [][]int

	for scanner.Scan() {
		var destinations []string
		var lineTokenized []string
		line := scanner.Text()
		// Check if the line is a comment or empty and skip it//
		if len(line) == 0 || line[0] == '#' {
			lineNumber++
			continue
		} else {
			lineTokenized = strings.Split(line, ":")
			// Check if the format is correct
			if len(lineTokenized) == 1 {
				return nil, errors.New("Line " + strconv.Itoa(lineNumber) + " does not follow the correct format")
			}
			destinations = strings.Split(lineTokenized[1], ",")
		}
		// Declare the topology in that line, the first element is the origin
		var topologyLine []int
		// Parse origin in the line
		origin, err := strconv.Atoi(lineTokenized[0])
		// Check if it can be casted to integer
		if err != nil {
			return nil, errors.New("Line: " + strconv.Itoa(lineNumber) + " of connection graph, could not be parsed")
		}
		// Check if the node is out of range
		if origin >= numberOfNodes {
			return nil, errors.New("Node origin in line: " + strconv.Itoa(lineNumber) + " out of range")
		}
		topologyLine = append(topologyLine, origin)
		for _, destination := range destinations {
			// Check if it can be casted to integer
			target, err := strconv.Atoi(destination)
			if err != nil {
				return nil, errors.New("Check line: " + strconv.Itoa(lineNumber) + " of connection graph, could not be parsed")
			}
			// Check if the node is out of range
			if target >= numberOfNodes {
				return nil, errors.New("Node target in line: " + strconv.Itoa(lineNumber) + " out of range")
			}
			// Append destination to graph
			topologyLine = append(topologyLine, target)
		}
		lineNumber++
		topology = append(topology, topologyLine)
	}
	return topology, nil
}
