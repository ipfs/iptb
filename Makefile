CLEAN =

all: iptb

deps:
	gx install

iptb: deps
	go build

plugins: deps
	make -C plugins all

install_plugins:
	make -C plugins install

CLEAN += iptb

install:
	go install

test:
	make -C sharness all

clean:
	rm $(CLEAN)

.PHONY: all test iptb install plugins clean
