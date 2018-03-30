///usr/bin/env go run "$0" "$@"; exit
// +build tool

/*
 * listtool.go
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
	"bytes"
	"encoding/json"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"text/template"
)

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: listtool template-file packages`)
	os.Exit(2)
}

func main() {
	if 2 > len(os.Args) {
		usage()
	}

	templ, err := template.ParseFiles(os.Args[1])
	if nil != err {
		fail(err)
	}

	args := append([]string{"list", "-json"}, os.Args[2:]...)
	out, err := exec.Command("go", args...).Output()
	if nil != err {
		if e, ok := err.(*exec.ExitError); ok {
			fmt.Fprintln(os.Stderr, string(e.Stderr))
			os.Exit(1)
		} else {
			fail(err)
		}
	}

	var packages []build.Package
	for dec := json.NewDecoder(bytes.NewReader(out)); dec.More(); {
		var p build.Package
		err := dec.Decode(&p)
		if nil != err {
			fail(err)
		}

		packages = append(packages, p)
	}

	err = templ.Execute(os.Stdout, packages)
	if nil != err {
		fail(err)
	}
}
