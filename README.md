# IPTB

`iptb` is a program used to create and manage a cluster of sandboxed nodes
locally on your computer. Spin up 1000s of nodes! Using `iptb` makes testing
libp2p networks easy!

### Example

```
$ iptb auto -count 5 -type <plugin_name> 

$ iptb start

$ iptb shell 0
$ echo $IPFS_PATH
/home/iptb/testbed/testbeds/default/0

$ echo 'hey!' | ipfs add -q
QmNqugRcYjwh9pEQUK7MLuxvLjxDNZL1DH8PJJgWtQXxuF

$ exit

$ iptb connect 0 4

$ iptb shell 4
$ ipfs cat QmNqugRcYjwh9pEQUK7MLuxvLjxDNZL1DH8PJJgWtQXxuF
hey!
```
Available plugins now: Local IPFS node (plugin_name: localipfs), Docker IPFS node (plugin_name: dockeripfs)
### Usage
```
NAME:
   iptb - iptb is a tool for managing test clusters of libp2p nodes

USAGE:
   iptb [global options] command [command options] [arguments...]

VERSION:
   0.0.0

COMMANDS:
     auto     create default testbed and initialize
     testbed  manage testbeds
     help, h  Shows a list of commands or help for one command
   ATTRIBUTES:
     attr  get, set, list attributes
   CORE:
     init     initialize specified nodes (or all)
     start    start specified nodes (or all)
     stop     stop specified nodes (or all)
     restart  restart specified nodes (or all)
     run      run command on specified nodes (or all)
     connect  connect sets of nodes together (or all)
     shell    starts a shell within the context of node
   METRICS:
     logs    show logs from specified nodes (or all)
     events  stream events from specified nodes (or all)
     metric  get metric from node

GLOBAL OPTIONS:
   --testbed value  Name of testbed to use under IPTB_ROOT (default: "default") [$IPTB_TESTBED]
   --help, -h       show help
   --version, -v    print the version
```

### Install

```
$ go get github.com/ipfs/iptb
```

### Configuration

By default, `iptb` uses `$HOME/testbed` to store created nodes. This path is configurable via the environment variables `IPTB_ROOT`.

### License

MIT
