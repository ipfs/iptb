all: ipfslocal ipfsdocker

deps:
	gx install

ipfslocal: deps
	gx-go rw
	(cd local; go build -buildmode=plugin -o ../../localipfs.so)
	gx-go uw
CLEAN += localipfs.so

ipfsdocker: deps
	gx-go rw
	(cd docker; go build -buildmode=plugin -o ../../dockeripfs.so)
	gx-go uw
CLEAN += dockeripfs.so

.PHONY: all ipfslocal ipfsdocker
