#!/bin/sh

test_description="iptb init --profile"

. lib/test-lib.sh

IPTB_ROOT=.

test_expect_success "iptb init works" '
	../bin/iptb init -n 3 --profile=badgerds
'

for i in {0..2}; do
	test_expect_success "node '$i' has badger datastore" '
		IPATH=$(iptb get path '$i')

		test -d "${IPATH}/badgerds"
	'
done


test_done
