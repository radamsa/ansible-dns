OWNER=radamsa
PROJECT=ansible-dns
BRANCH=master
SRC=src
DIST=dist
B=$(shell git rev-parse --abbrev-ref HEAD)
BRANCH=$(subst /,-,$(B))
GITREV=$(shell git describe --abbrev=7 --always --tags)
REV=$(GITREV)-$(BRANCH)-$(shell date +%Y%m%d-%H%M%S)

docker:
	docker build -t $(OWNER)/$(PROJECT):$(BRANCH) --progress=plain .

race_test:
	cd $(SRC) && go test -race -mod=vendor -timeout=60s -count 1 ./...

build: info
	- cd $(SRC) && CGO_ENABLED=0 go build -ldflags "-X main.project=$(PROJECT) -X main.revision=$(REV) -s -w" -o ../$(DIST)/$(PROJECT)

install: build
	- cp $(DIST)/$(PROJECT) $(HOME)/bin

info:
	- @echo "$(PROJECT) revision $(REV)"

.PHONY: docker race_test bin info
