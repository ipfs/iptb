# IPTB Changelog

## 2.0.0 2018-11-02

The **IPTB v2** release is a complete rewrite for the most part that shifts IPTB to a more general tool that many projects outside IPFS can use independently.

### Highlights
The key differences from IPTB v1 are changes to the [Core Interface](#core-interface) to define functionality and make IPTB easier to extend, use of [plugins](#plugins) to load functionality, and updates to the [CLI](#cli) to make it easier to use independently.

### Core Interface
IPTB previously had some built in assumptions, largely around go-ipfs. We now have a set of interfaces that define all of IPTBs functionality and make it easier to extend, and support other projects.

- Core
- Metrics
- Attributes

See https://github.com/ipfs/iptb/blob/master/testbed/interfaces/node.go

### Plugins
IPTB uses plugins to load functionality. Plugins can be loaded from disk by placing them under $IPTB_ROOT/plugins, or by building them into the IPTB binary and registering them.

Current plugins written for the IPFS project can be found @ https://github.com/ipfs/iptb-plugins

### CLI
Due to the large changes under the hood to IPTB, we wanted to also take the time to update the cli. All commands now use the same order, accepting the node ID / range as the first argument. The CLI closely maps to the IPTB interfaces.

See README https://github.com/ipfs/iptb#usage

The CLI is also now a package itself, which makes it really easy to roll-your-own-iptb by registering plugins. This makes it easy to build a custom IPTB with built in plugins and not having to worry as much with moving plugins around on disk.

See https://github.com/ipfs/iptb-plugins/blob/master/iptb/iptb.go
