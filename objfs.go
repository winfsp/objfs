///usr/bin/env go run -tags debug objfs.go registry.go commands.go cache_commands.go debug.go "$@"; exit

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
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/billziss-gh/golib/appdata"
	"github.com/billziss-gh/golib/cmd"
	"github.com/billziss-gh/golib/errors"
	"github.com/billziss-gh/golib/trace"
	"github.com/billziss-gh/objfs/auth"
	"github.com/billziss-gh/objfs/httputil"
	"github.com/billziss-gh/objfs/objio"
)

var (
	cachePath     string
	authName      string
	credentials   auth.CredentialMap
	authSession   auth.Session
	storageName   string
	storageUri    string
	storage       objio.ObjectStorage
	acceptTlsCert bool
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
	flag.StringVar(&authName, "auth", "",
		"auth `name` to use")
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

var needvarOnce sync.Once

func needvar(args ...interface{}) {
	needvarOnce.Do(func() {
		if acceptTlsCert {
			httputil.DefaultTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
	})

	for _, a := range args {
		switch a {
		case &authName:
			if "" != authName {
				continue
			}
			needvar(&storageName)
			authName = storageName

		case &authSession:
			if nil != authSession {
				continue
			}
			needvar(&authName, &credentials)
			a, err := auth.Registry.NewObject(authName)
			if nil != err {
				warn(errors.New("unknown auth; specify -auth in the command line"))
				usage(nil)
			}
			s, err := a.(auth.Auth).Session(credentials)
			if nil != err {
				fail(err)
			}
			authSession = s

		case &cachePath:
			if "" != cachePath {
				continue
			}
			needvar(&storageName)
			dir, err := appdata.DataDir()
			if nil != err {
				fail(err)
			}
			cachePath = filepath.Join(dir, "objfs", storageName)

		case &credentials:
			if nil != credentials {
				continue
			}
			needvar(&storageName)
			credentials, _ = auth.ReadCredentials("keyring:objfs/" + storageName)
			if nil == credentials {
				warn(errors.New("unknown credentials; specify -credentials in the command line"))
				usage(nil)
			}

		case &storageName:
			if "" != storageName {
				continue
			}
			warn(errors.New("unknown storage; specify -storage in the command line"))
			usage(nil)

		case &storage:
			if nil != storage {
				continue
			}
			var creds interface{}
			if "" != authName {
				needvar(&authSession, &storageName)
				creds = authSession
			} else {
				needvar(&credentials, &storageName)
				creds = credentials
			}
			s, err := objio.Registry.NewObject(storageName, storageUri, creds)
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
