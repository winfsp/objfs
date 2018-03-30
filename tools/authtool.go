///usr/bin/env go run "$0" "$@"; exit
// +build tool

/*
 * authtool.go
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
	"fmt"
	"os"

	"github.com/billziss-gh/objfs.pkg/auth/oauth2"
	"github.com/billziss-gh/objfs/auth"
)

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: authtool kind args... ipath opath`)
	os.Exit(2)
}

func main() {
	if 5 > len(os.Args) {
		usage()
	}

	kind := os.Args[1]
	args := []interface{}{}
	for _, a := range os.Args[2 : len(os.Args)-2] {
		args = append(args, a)
	}
	ipath := os.Args[len(os.Args)-2]
	opath := os.Args[len(os.Args)-1]

	cmap, err := auth.ReadCredentials(ipath)
	if nil != err {
		fail(err)
	}

	a, err := auth.Registry.NewObject(kind, args...)
	if nil != err {
		fail(err)
	}

	sess, err := a.(auth.Auth).Session(cmap)
	if nil != err {
		fail(err)
	}

	cmap = sess.Credentials()
	err = auth.WriteCredentials(opath, cmap)
	if nil != err {
		fail(err)
	}
}

func init() {
	oauth2.Load()
}
