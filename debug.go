// +build debug

/*
 * debug.go
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
	"net"
	"net/http"
	"os"

	_ "net/http/pprof"
)

func init() {
	listen, err := net.Listen("tcp", "localhost:0")
	if nil != err {
		fmt.Fprintf(os.Stderr, "debug: cannot listen: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "debug: listening on %v\n", listen.Addr())

	go http.Serve(listen, nil)
}
