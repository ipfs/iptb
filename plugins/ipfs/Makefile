all: ipfslocal ipfsdocker ipfsbrowser

deps:
	gx install

ipfslocal: deps
	gx-go rw
	(cd local/plugin; go build -buildmode=plugin -o ../../../localipfs.so)
	gx-go uw
CLEAN += localipfs.so

ipfsdocker: deps
	gx-go rw
	(cd docker/plugin; go build -buildmode=plugin -o ../../../dockeripfs.so)
	gx-go uw
CLEAN += dockeripfs.so

ipfsbrowser:
	gx-go rw
	(cd browser/plugin; go build -buildmode=plugin -o ../../../browseripfs.so)
	gx-go uw
CLEAN += browseripfs.so

.PHONY: all ipfslocal ipfsdocker ipfsbrowser
