// +build !debug

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

func traceIgnore(...interface{}) {
}

func Trace(vals ...interface{}) func(vals ...interface{}) {
	return traceIgnore
}
