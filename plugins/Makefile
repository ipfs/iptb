IPTB_ROOT ?= $(HOME)/testbed

all: ipfs

install:
	mkdir -p $(IPTB_ROOT)/plugins
	cp *.so $(IPTB_ROOT)/plugins

ipfs:
	make -C ipfs all

clean:
	rm *.so

.PHONY: all ipfs clean
