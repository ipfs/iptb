#!/bin/sh

test_description="iptb --version"

. ./lib/sharness/sharness.sh

test_expect_success "ipfs-update binary is here" '
	test -f ../bin/iptb
'

test_expect_success "'iptb --version' works" '
	iptb --version >actual
'

test_expect_success "'iptb --version' output looks good" '
	egrep "^iptb version [0-9]+.[0-9]+.[0-9]+$" actual
'

test_done
