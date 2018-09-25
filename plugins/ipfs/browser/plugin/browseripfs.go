package main

import (
	plugin "github.com/ipfs/iptb/plugins/ipfs/browser"
	testbedi "github.com/ipfs/iptb/testbed/interfaces"
)

var PluginName string
var NewNode testbedi.NewNodeFunc

func init() {
	PluginName = plugin.PluginName
	NewNode = plugin.NewNode
}
