/*
 * assets.go
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

package assets

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var basedir string

func init() {
	exe, err := os.Executable()
	if nil == err && !strings.HasSuffix(exe, ".test") && !strings.HasSuffix(exe, ".test.exe") {
		basedir = filepath.Join(filepath.Dir(exe), "assets")
		if info, err := os.Stat(filepath.Join(basedir, "sys")); nil == err && info.IsDir() {
			return
		}
	}

	basedir = "\000"
}

// GetPath returns the full path for an asset.
func GetPath(subdir string, name string) string {
	dir := basedir
	if "\000" == dir {
		_, file, _, ok := runtime.Caller(1)
		if ok {
			testdir := filepath.Join(filepath.Dir(file), "assets")
			if info, err := os.Stat(filepath.Join(testdir, "sys")); nil == err && info.IsDir() {
				dir = testdir
			}
		}
	}

	if "sys" == subdir {
		return filepath.Join(dir, "sys", runtime.GOOS+"_"+runtime.GOARCH, name)
	} else {
		return filepath.Join(dir, subdir, name)
	}
}
