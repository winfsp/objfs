///usr/bin/env go run -tags debug objfs.go registry.go commands.go cache_commands.go "$@"; exit

/*
 * objfs.go
 *
 * Copyright 2018 Bill Zissimopoulos
 */
/*
 * This file is part of Objfs.
 *
 * You can redistribute it and/or modify it under the terms of the GNU
 * Affero General Public License version 3 as published by the Free
 * Software Foundation.
 *
 * Licensees holding a valid commercial license may use this file in
 * accordance with the commercial license agreement provided with the
 * software.
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/billziss-gh/golib/appdata"
	"github.com/billziss-gh/golib/cmd"
	"github.com/billziss-gh/golib/errors"
	"github.com/billziss-gh/golib/trace"
	"github.com/billziss-gh/objfs/auth"
	"github.com/billziss-gh/objfs/httputil"
	"github.com/billziss-gh/objfs/objio"
)

var (
	cachePath      string
	credentials    auth.CredentialMap
	storageName    string
	storageUri     string
	storage        objio.ObjectStorage
	acceptTlsCert  bool
	storageUriMap  = map[string]string{}
	authSessionMap = map[string]func(auth.CredentialMap) (auth.Session, error){}
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-options] command args...\n", filepath.Base(os.Args[0]))
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "commands:")
		cmd.PrintCmds()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "options:")
		flag.PrintDefaults()
	}

	flag.StringVar(&cachePath, "cache", "",
		"`path` to file system cache")
	flag.Var(&credentials, "credentials",
		"auth credentials `path` (keyring:service/user or /file/path)")
	flag.StringVar(&storageName, "storage", defaultStorageName,
		"storage `name` to access")
	flag.StringVar(&storageUri, "storage-uri", "",
		"storage `uri` to access")
	flag.BoolVar(&acceptTlsCert, "accept-tls-cert", false,
		"accept any TLS certificate presented by the server (insecure)")
	flag.BoolVar(&trace.Verbose, "v", false,
		"verbose")
}

func usage(cmd *cmd.Cmd) {
	if nil == cmd {
		flag.Usage()
	} else {
		cmd.Flag.Usage()
	}
	os.Exit(2)
}

func needvar(args ...interface{}) {
	if acceptTlsCert {
		httputil.SetInsecureSkipVerify(httputil.DefaultClient)
	}

	for _, a := range args {
		switch a {
		case &cachePath:
			if "" == cachePath && "" != storageName {
				dir, err := appdata.DataDir()
				if nil != err {
					fail(err)
				}
				cachePath = filepath.Join(dir, "objfs", storageName)
			}
			if "" == cachePath {
				warn(errors.New("unknown cache path; specify -cache in the command line"))
				usage(nil)
			}

		case &credentials:
			if nil == credentials && "" != storageName {
				credentials, _ = auth.ReadCredentials("keyring:objfs/" + storageName)
			}
			if nil == credentials {
				warn(errors.New("unknown credentials; specify -credentials in the command line"))
				usage(nil)
			}

		case &storageName:
			if _, ok := storageUriMap[storageName]; !ok {
				warn(errors.New("unknown storage " + storageName +
					"; specify -storage in the command line"))
				usage(nil)
			}

		case &storage:
			if "" == storageUri {
				storageUri = storageUriMap[storageName]
			}
			s, err := objio.Registry.NewObject(storageName, storageUri, credentials)
			if nil != err {
				fail(err)
			}
			storage = s.(objio.ObjectStorage)
			if trace.Verbose {
				storage = &objio.TraceObjectStorage{ObjectStorage: storage}
			}
		}
	}
}

func warn(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
}

func fail(err error) {
	warn(err)
	os.Exit(1)
}

func main() {
	for _, name := range cmd.DefaultCmdMap.GetNames() {
		cmd := cmd.DefaultCmdMap.Get(name)
		cmd.Flag.Usage = func() {
			fmt.Fprintf(os.Stderr, "usage: %s %s\n", filepath.Base(os.Args[0]), cmd.Use)
			cmd.Flag.PrintDefaults()
		}
	}

	cmd.Run()
}
