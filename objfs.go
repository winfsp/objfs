///usr/bin/env go run objfs.go registry.go commands.go "$@"; exit

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
	"crypto/rand"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/billziss-gh/golib/appdata"
	"github.com/billziss-gh/golib/cmd"
	"github.com/billziss-gh/golib/config"
	cflag "github.com/billziss-gh/golib/config/flag"
	"github.com/billziss-gh/golib/errors"
	"github.com/billziss-gh/golib/keyring"
	"github.com/billziss-gh/golib/trace"
	"github.com/billziss-gh/golib/util"
	"github.com/billziss-gh/objfs/auth"
	"github.com/billziss-gh/objfs/httputil"
	"github.com/billziss-gh/objfs/objio"
)

// Configuration variables. These variables control the overall operation of objfs.
//
// The logic of initializing these variables is rather complicated:
//
// - The configuration is determined by a combination of command-line parameters
// and a configuration file. When there is a conflict between the two, the
// command-line parameters take precendence.
//
// - The configuration file is named objfs.conf and placed in the appropriate
// directory for the underlying system, unless the -config command-line parameter
// is specified. The configuration file (if it exists) stores key/value pairs and
// may also have [sections].
//
// - The process starts by creating an empty "flag map" and proceeds by merging
// key/value pairs from the different sources.
//
// - If the configuration file exists it is read and the unnamed empty section ("")
// is merged into the flag map. Then any "-storage" command line parameter
// is merged into the flag map. Then if there is a configuration section with the
// name specified by "storage" that section is merged into the flag map.
//
// - The remaining command-line options (other than -storage) are merged
// into the flag map.
//
// - Finally the flag map is used to initialize the configuration variables.
//
// For the full logic see needvar.
var (
	configPath    string
	dataDir       string
	programConfig config.TypedConfig

	acceptTlsCert  bool
	authName       string
	authSession    auth.Session
	cachePath      string
	credentialPath string
	credentials    auth.CredentialMap
	storage        objio.ObjectStorage
	storageName    string
	storageUri     string
)

func init() {
	flag.Usage = cmd.UsageFunc()

	flag.StringVar(&configPath, "config", "",
		"`path` to configuration file")
	flag.String("datadir", "",
		"`path` to supporting data and caches")
	flag.BoolVar(&trace.Verbose, "v", false,
		"verbose")

	flag.Bool("accept-tls-cert", false,
		"accept any TLS certificate presented by the server (insecure)")
	flag.String("auth", "",
		"auth `name` to use")
	flag.String("credentials", "",
		"auth credentials `path` (keyring:service/user or /file/path)")
	flag.String("storage", defaultStorageName,
		"storage `name` to access")
	flag.String("storage-uri", "",
		"storage `uri` to access")
}

func usage(cmd *cmd.Cmd) {
	if nil == cmd {
		flag.Usage()
	} else {
		cmd.Flag.Usage()
	}
	os.Exit(2)
}

func initKeyring(path string) {
	var key []byte
	pass, err := keyring.Get("objfs", "keyring")
	if nil != err {
		key = make([]byte, 16)
		_, err = rand.Read(key)
		if nil != err {
			fail(err)
		}
		err = keyring.Set("objfs", "keyring", string(key))
		if nil != err {
			fail(err)
		}
	} else {
		key = []byte(pass)
	}

	keyring.DefaultKeyring = &keyring.OverlayKeyring{
		Keyrings: []keyring.Keyring{
			&keyring.FileKeyring{
				Path: filepath.Join(path, "keyring"),
				Key:  key,
			},
			keyring.DefaultKeyring,
		},
	}
}

var needvarOnce sync.Once

func needvar(args ...interface{}) {
	needvarOnce.Do(func() {
		if "" == configPath {
			dir, err := appdata.ConfigDir()
			if nil != err {
				fail(err)
			}

			configPath = filepath.Join(dir, "objfs.conf")
		}

		flagMap := config.TypedSection{}
		cflag.VisitAll(nil, flagMap,
			"accept-tls-cert",
			"auth",
			"credentials",
			"datadir",
			"storage",
			"storage-uri")

		c, err := util.ReadFunc(configPath, func(file *os.File) (interface{}, error) {
			return config.ReadTyped(file)
		})
		if nil == err {
			programConfig = c.(config.TypedConfig)

			for k, v := range programConfig[""] {
				flagMap[k] = v
			}

			cflag.Visit(nil, flagMap, "storage")

			for k, v := range programConfig[flagMap["storage"].(string)] {
				flagMap[k] = v
			}

			cflag.Visit(nil, flagMap,
				"accept-tls-cert",
				"auth",
				"credentials",
				"datadir",
				"storage-uri")
		} else {
			programConfig = config.TypedConfig{}
		}

		acceptTlsCert = flagMap["accept-tls-cert"].(bool)
		authName = flagMap["auth"].(string)
		credentialPath = flagMap["credentials"].(string)
		dataDir = flagMap["datadir"].(string)
		storageName = flagMap["storage"].(string)
		storageUri = flagMap["storage-uri"].(string)

		if "" == dataDir {
			dir, err := appdata.DataDir()
			if nil != err {
				fail(err)
			}

			dataDir = filepath.Join(dir, "objfs")
		}

		initKeyring(dataDir)

		if false {
			fmt.Printf("configPath=%#v\n", configPath)
			fmt.Printf("dataDir=%#v\n", dataDir)
			fmt.Println()
			fmt.Printf("acceptTlsCert=%#v\n", acceptTlsCert)
			fmt.Printf("authName=%#v\n", authName)
			fmt.Printf("credentialPath=%#v\n", credentialPath)
			fmt.Printf("storageName=%#v\n", storageName)
			fmt.Printf("storageUri=%#v\n", storageUri)
		}

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
			cachePath = filepath.Join(dataDir, storageName)

		case &credentialPath:
			if "" != credentialPath {
				continue
			}
			needvar(&storageName)
			credentialPath = "keyring:objfs/" + storageName

		case &credentials:
			if nil != credentials {
				continue
			}
			needvar(&credentialPath)
			credentials, _ = auth.ReadCredentials(credentialPath)
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
	cmd.Run()
}
