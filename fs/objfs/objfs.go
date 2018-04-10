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

package objfs

import (
	"io"
	"runtime"

	"github.com/billziss-gh/cgofuse/fuse"
	"github.com/billziss-gh/golib/errors"
	"github.com/billziss-gh/golib/trace"
	"github.com/billziss-gh/objfs/cache"
	"github.com/billziss-gh/objfs/errno"
	"github.com/billziss-gh/objfs/fs"
)

type objfs struct {
	fuse.FileSystemBase
	cache *cache.Cache
}

func (self *objfs) Statfs(path string, stat *fuse.Statfs_t) (errc int) {
	if trace.Verbose {
		defer fs.Trace(path)(&errc, stat)
	}

	info, err := self.cache.Statfs()
	if nil != err {
		return fs.FuseErrc(err)
	}

	fs.CopyFusestatfsFromStorageInfo(stat, info)

	return 0
}

func (self *objfs) Mkdir(path string, mode uint32) (errc int) {
	if trace.Verbose {
		defer fs.Trace(path, mode)(&errc)
	}

	ino, err := self.cache.Open(path)
	if nil != err {
		return fs.FuseErrc(err)
	}
	defer self.cache.Close(ino)

	err = self.cache.Make(ino, true)
	if nil != err {
		return fs.FuseErrc(err)
	}

	return 0
}

func (self *objfs) Unlink(path string) (errc int) {
	if trace.Verbose {
		defer fs.Trace(path)(&errc)
	}

	ino, err := self.cache.Open(path)
	if nil != err {
		return fs.FuseErrc(err)
	}
	defer self.cache.Close(ino)

	err = self.cache.Remove(ino, false)
	if nil != err {
		return fs.FuseErrc(err)
	}

	return 0
}

func (self *objfs) Rmdir(path string) (errc int) {
	if trace.Verbose {
		defer fs.Trace(path)(&errc)
	}

	ino, err := self.cache.Open(path)
	if nil != err {
		return fs.FuseErrc(err)
	}
	defer self.cache.Close(ino)

	err = self.cache.Remove(ino, true)
	if nil != err {
		return fs.FuseErrc(err)
	}

	return 0
}

func (self *objfs) Rename(oldpath string, newpath string) (errc int) {
	if trace.Verbose {
		defer fs.Trace(oldpath, newpath)(&errc)
	}

	ino, err := self.cache.Open(oldpath)
	if nil != err {
		return fs.FuseErrc(err)
	}
	defer self.cache.Close(ino)

	err = self.cache.Rename(ino, newpath)
	if nil != err {
		return fs.FuseErrc(err)
	}

	return 0
}

func (self *objfs) Utimens(path string, tmsp []fuse.Timespec) (errc int) {
	if trace.Verbose {
		defer fs.Trace(path, tmsp)(&errc)
	}

	return -fuse.ENOSYS
}

func (self *objfs) Create(path string, flags int, mode uint32) (errc int, ino uint64) {
	if trace.Verbose {
		defer fs.Trace(path, flags, mode)(&errc, &ino)
	}

	ino, err := self.cache.Open(path)
	if nil != err {
		return fs.FuseErrc(err), ^uint64(0)
	}

	err = self.cache.Make(ino, false)
	if nil != err {
		self.cache.Close(ino)
		return fs.FuseErrc(err), ^uint64(0)
	}

	return 0, ino
}

func (self *objfs) Open(path string, flags int) (errc int, ino uint64) {
	if trace.Verbose {
		defer fs.Trace(path, flags)(&errc, &ino)
	}

	ino, err := self.cache.Open(path)
	if nil != err {
		return fs.FuseErrc(err), ^uint64(0)
	}

	_, err = self.cache.Stat(ino)
	if nil != err {
		self.cache.Close(ino)
		return fs.FuseErrc(err), ^uint64(0)
	}

	return 0, ino
}

func (self *objfs) Getattr(path string, stat *fuse.Stat_t, ino uint64) (errc int) {
	if trace.Verbose {
		defer fs.Trace(path, ino)(&errc, stat)
	}

	if ^uint64(0) == ino {
		var err error
		ino, err = self.cache.Open(path)
		if nil != err {
			return fs.FuseErrc(err)
		}
		defer self.cache.Close(ino)
	}

	info, err := self.cache.Stat(ino)
	if nil != err {
		return fs.FuseErrc(err)
	}

	fs.CopyFusestatFromObjectInfo(stat, info)

	return 0
}

func (self *objfs) Truncate(path string, size int64, ino uint64) (errc int) {
	if trace.Verbose {
		defer fs.Trace(path, size, ino)(&errc)
	}

	if ^uint64(0) == ino {
		var err error
		ino, err = self.cache.Open(path)
		if nil != err {
			return fs.FuseErrc(err)
		}
		defer self.cache.Close(ino)
	}

	err := self.cache.Truncate(ino, size)
	if nil != err {
		return fs.FuseErrc(err)
	}

	return 0
}

func (self *objfs) Read(path string, buff []byte, ofst int64, ino uint64) (n int) {
	if trace.Verbose {
		defer fs.Trace(path, ofst, ino)(&n)
	}

	n, err := self.cache.ReadAt(ino, buff, ofst)
	if nil != err && io.EOF != err {
		return fs.FuseErrc(err)
	}

	return n
}

func (self *objfs) Write(path string, buff []byte, ofst int64, ino uint64) (n int) {
	if trace.Verbose {
		defer fs.Trace(path, ofst, ino)(&n)
	}

	n, err := self.cache.WriteAt(ino, buff, ofst)
	if nil != err {
		return fs.FuseErrc(err)
	}

	return n
}

func (self *objfs) Release(path string, ino uint64) (errc int) {
	if trace.Verbose {
		defer fs.Trace(path, ino)(&errc)
	}

	err := self.cache.Close(ino)
	if nil != err {
		return fs.FuseErrc(err)
	}

	return 0
}

func (self *objfs) Fsync(path string, datasync bool, ino uint64) (errc int) {
	if trace.Verbose {
		defer fs.Trace(path, datasync, ino)(&errc)
	}

	err := self.cache.Sync(ino)
	if nil != err {
		return fs.FuseErrc(err)
	}

	return 0
}

func (self *objfs) Opendir(path string) (errc int, ino uint64) {
	if trace.Verbose {
		defer fs.Trace(path)(&errc, &ino)
	}

	ino, err := self.cache.Open(path)
	if nil != err {
		return fs.FuseErrc(err), ^uint64(0)
	}

	_, err = self.cache.Stat(ino)
	if nil != err {
		self.cache.Close(ino)
		return fs.FuseErrc(err), ^uint64(0)
	}

	return 0, ino
}

func (self *objfs) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	ino uint64) (errc int) {
	if trace.Verbose {
		defer fs.Trace(path, ofst, ino)(&errc)
	}

	infos, err := self.cache.Readdir(ino, 0)
	if nil != err {
		return fs.FuseErrc(err)
	}

	// on Windows only add dot entries for non-root
	if !onWindows || "/" != path {
		fill(".", nil, 0)
		fill("..", nil, 0)
	}

	for _, info := range infos {
		stat := fuse.Stat_t{}

		fs.CopyFusestatFromObjectInfo(&stat, info)

		if !fill(info.Name(), &stat, 0) {
			break
		}
	}

	return 0
}

func (self *objfs) Releasedir(path string, ino uint64) (errc int) {
	if trace.Verbose {
		defer fs.Trace(path, ino)(&errc)
	}

	err := self.cache.Close(ino)
	if nil != err {
		return fs.FuseErrc(err)
	}

	return 0
}

func (self *objfs) Setxattr(path string, name string, value []byte, flags int) int {
	// return EPERM to completely disable the xattr mechanism on Darwin
	return -fuse.EPERM
}

func (self *objfs) Getxattr(path string, name string) (int, []byte) {
	// return EPERM to completely disable the xattr mechanism on Darwin
	return -fuse.EPERM, nil
}

func (self *objfs) Removexattr(path string, name string) int {
	// return EPERM to completely disable the xattr mechanism on Darwin
	return -fuse.EPERM
}

func (self *objfs) Listxattr(path string, fill func(name string) bool) int {
	// return EPERM to completely disable the xattr mechanism on Darwin
	return -fuse.EPERM
}

func (self *objfs) IsCaseInsensitive() (res bool) {
	res = false
	info, err := self.cache.Storage().Info(false)
	if nil == err {
		res = info.IsCaseInsensitive()
	}
	return
}

func New(args ...interface{}) (interface{}, error) {
	var c *cache.Cache
	for _, arg := range args {
		switch a := arg.(type) {
		case *cache.Cache:
			c = a
		}
	}

	if nil == c {
		return nil, errors.New(": missing cache", nil, errno.EINVAL)
	}

	self := &objfs{
		cache: c,
	}

	return self, nil
}

var _ fuse.FileSystemInterface = (*objfs)(nil)
var _ fs.FileSystemCaseInsensitive = (*objfs)(nil)

var onWindows = "windows" == runtime.GOOS

// Load is used to ensure that this package is linked.
func Load() {
}

func init() {
	fs.Registry.RegisterFactory("objfs", New)
}
