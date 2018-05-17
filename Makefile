# Makefile

MyBuildNumber=$(shell date +%y%j)
MyVersion=0.9.$(MyBuildNumber)

Packages=\
	./vendor/github.com/billziss-gh/objfs.pkg/objio/onedrive\
	./vendor/github.com/billziss-gh/objfs.pkg/objio/dropbox

ifeq ($(OS),Windows_NT)
PathSep=\$(strip)
else
PathSep=/
endif

.PHONY: default
default: build

.PHONY: generate
generate:
	go generate ./...

.PHONY: build
build: registry.go
	go build -ldflags "-s -w -X \"main.MyVersion=$(MyVersion)\""

.PHONY: debug
debug: registry.go
	go build -race -tags debug -gcflags all="-N -l"

registry.go: registry.go.in Makefile
	go run _tools/listtool.go registry.go.in $(Packages) > registry.go

.PHONY: manpage
manpage: $(patsubst %.1.asciidoc,%.1,$(wildcard *.1.asciidoc))
%.1: %.1.asciidoc
	asciidoctor -b manpage *.1.asciidoc

.PHONY: test
test:
	_tools$(PathSep)run-tests -count=1 ./... ./vendor/github.com/billziss-gh/objfs.pkg/...
