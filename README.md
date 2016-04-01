# IPTB
iptb is a program used to manage a cluster of ipfs nodes locally on your
computer. It allows the creation of up to 1000 or more nodes! Allows for
various setup options to be selected such as different bootstrapping patterns.
iptb makes testing networks in ipfs easy!

### Example

```
$ iptb init -n 5

$ iptb start
Started daemon 0, pid = 12396
Started daemon 1, pid = 12406
Started daemon 2, pid = 12415
Started daemon 3, pid = 12424
Started daemon 4, pid = 12434

$ iptb shell 0
$ echo $IPFS_PATH
/home/noffle/testbed/0

$ echo 'hey!' | ipfs add -q
QmNqugRcYjwh9pEQUK7MLuxvLjxDNZL1DH8PJJgWtQXxuF

$ iptb shell 4
$ ipfs cat QmNqugRcYjwh9pEQUK7MLuxvLjxDNZL1DH8PJJgWtQXxuF
hey!
```

### Usage
```
NAME:
	iptb - A new cli application

USAGE:
	iptb [global options] command [command options] [arguments...]

COMMANDS:
	init		create and initialize testbed nodes
	start	starts up all testbed nodes
	kill, stop	kill a given node (or all nodes if none specified)
	restart	kill all nodes, then restart
	shell	execs your shell with certain environment variables set
	get		get an attribute of the given node
	connect	connect two nodes together
	dump-stack	get a stack dump from the given daemon
	help, h	Shows a list of commands or help for one command

GLOBAL OPTIONS:
	--help, -h		show help
	--version, -v	print the version
```



### Configuration
By default, iptb uses `$HOME/testbed` to store created nodes. This path is
configurable via the environment variables `IPTB_ROOT`. 



