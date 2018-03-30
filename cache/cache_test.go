/*
 * cache_test.go
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

package cache

import (
	"testing"
)

func TestPartialPaths(t *testing.T) {
	var paths []string

	paths = partialPaths("/")
	if 1 != len(paths) || "/" != paths[0] {
		t.Error()
	}

	paths = partialPaths("//foo")
	if 2 != len(paths) || "/" != paths[0] || "/foo" != paths[1] {
		t.Error()
	}

	paths = partialPaths("/foo/")
	if 2 != len(paths) || "/" != paths[0] || "/foo" != paths[1] {
		t.Error()
	}

	paths = partialPaths("/foo/bar")
	if 3 != len(paths) || "/" != paths[0] || "/foo" != paths[1] || "/foo/bar" != paths[2] {
		t.Error()
	}

	paths = partialPaths("/foo/bar/Δοκιμή")
	if 4 != len(paths) || "/" != paths[0] || "/foo" != paths[1] || "/foo/bar" != paths[2] ||
		"/foo/bar/Δοκιμή" != paths[3] {
		t.Error()
	}

	paths = partialPaths("/foo/bar/Δοκιμή/baz")
	if 5 != len(paths) || "/" != paths[0] || "/foo" != paths[1] || "/foo/bar" != paths[2] ||
		"/foo/bar/Δοκιμή" != paths[3] || "/foo/bar/Δοκιμή/baz" != paths[4] {
		t.Error()
	}

	paths = partialPaths("/foo/bar///Δοκιμή/////baz")
	if 5 != len(paths) || "/" != paths[0] || "/foo" != paths[1] || "/foo/bar" != paths[2] ||
		"/foo/bar/Δοκιμή" != paths[3] || "/foo/bar/Δοκιμή/baz" != paths[4] {
		t.Error()
	}
}

func TestNormalizeCase(t *testing.T) {
	s := ""

	s = normalizeCase("TEST test")
	if "TEST TEST" != s {
		t.Error(s)
	}

	s = normalizeCase("ΔΟΚΙΜΉ Δοκιμή")
	if "ΔΟΚͅµΉ ΔΟΚͅµΉ" != s {
		t.Error(s)
	}

	s = normalizeCase("ΣΊΣΥΦΟΣ Σίσυφος")
	if "ΣΊΣΥΦΟΣ ΣΊΣΥΦΟΣ" != s {
		t.Error(s)
	}

	s = normalizeCase("TSCHÜSS, TSCHÜẞ, tschüß, tschüss")
	if "TSCHÜSS, TSCHÜß, TSCHÜß, TSCHÜSS" != s {
		t.Error(s)
	}
}
