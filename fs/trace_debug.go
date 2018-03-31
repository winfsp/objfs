// +build debug

/*
 * trace.go
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

package fs

import (
	"fmt"

	"github.com/billziss-gh/cgofuse/fuse"
	"github.com/billziss-gh/golib/trace"
)

func Trace(vals ...interface{}) func(vals ...interface{}) {
	uid, gid, _ := fuse.Getcontext()
	return trace.Trace(1, fmt.Sprintf("[uid=%v,gid=%v]", uid, gid), vals...)
}

func Tracef(form string, vals ...interface{}) {
	trace.Tracef(1, form, vals...)
}
