/*
 * tracefs.go
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

type TraceFs struct {
	fuse.FileSystemInterface
}

func traceFuse(vals ...interface{}) func(vals ...interface{}) {
	uid, gid, _ := fuse.Getcontext()
	return trace.Trace(1, fmt.Sprintf("[uid=%v,gid=%v]", uid, gid), vals...)
}

func (self *TraceFs) Init() {
	defer traceFuse()()
	self.FileSystemInterface.Init()
}

func (self *TraceFs) Destroy() {
	defer traceFuse()()
	self.FileSystemInterface.Destroy()
}

func (self *TraceFs) Statfs(path string, stat *fuse.Statfs_t) (errc int) {
	defer traceFuse(path)(traceErrc{&errc}, stat)
	return self.FileSystemInterface.Statfs(path, stat)
}

func (self *TraceFs) Mknod(path string, mode uint32, dev uint64) (errc int) {
	defer traceFuse(path, mode, dev)(traceErrc{&errc})
	return self.FileSystemInterface.Mknod(path, mode, dev)
}

func (self *TraceFs) Mkdir(path string, mode uint32) (errc int) {
	defer traceFuse(path, mode)(traceErrc{&errc})
	return self.FileSystemInterface.Mkdir(path, mode)
}

func (self *TraceFs) Unlink(path string) (errc int) {
	defer traceFuse(path)(traceErrc{&errc})
	return self.FileSystemInterface.Unlink(path)
}

func (self *TraceFs) Rmdir(path string) (errc int) {
	defer traceFuse(path)(traceErrc{&errc})
	return self.FileSystemInterface.Rmdir(path)
}

func (self *TraceFs) Link(oldpath string, newpath string) (errc int) {
	defer traceFuse(oldpath, newpath)(traceErrc{&errc})
	return self.FileSystemInterface.Link(oldpath, newpath)
}

func (self *TraceFs) Symlink(target string, newpath string) (errc int) {
	defer traceFuse(target, newpath)(traceErrc{&errc})
	return self.FileSystemInterface.Symlink(target, newpath)
}

func (self *TraceFs) Readlink(path string) (errc int, target string) {
	defer traceFuse(path)(traceErrc{&errc}, &target)
	return self.FileSystemInterface.Readlink(path)
}

func (self *TraceFs) Rename(oldpath string, newpath string) (errc int) {
	defer traceFuse(oldpath, newpath)(traceErrc{&errc})
	return self.FileSystemInterface.Rename(oldpath, newpath)
}

func (self *TraceFs) Chmod(path string, mode uint32) (errc int) {
	defer traceFuse(path, mode)(traceErrc{&errc})
	return self.FileSystemInterface.Chmod(path, mode)
}

func (self *TraceFs) Chown(path string, uid uint32, gid uint32) (errc int) {
	defer traceFuse(path, uid, gid)(traceErrc{&errc})
	return self.FileSystemInterface.Chown(path, uid, gid)
}

func (self *TraceFs) Utimens(path string, tmsp []fuse.Timespec) (errc int) {
	defer traceFuse(path, tmsp)(traceErrc{&errc})
	return self.FileSystemInterface.Utimens(path, tmsp)
}

func (self *TraceFs) Access(path string, mask uint32) (errc int) {
	defer traceFuse(path, mask)(traceErrc{&errc})
	return self.FileSystemInterface.Access(path, mask)
}

func (self *TraceFs) Create(path string, flags int, mode uint32) (errc int, fh uint64) {
	defer traceFuse(path, flags, mode)(traceErrc{&errc}, &fh)
	return self.FileSystemInterface.Create(path, flags, mode)
}

func (self *TraceFs) Open(path string, flags int) (errc int, fh uint64) {
	defer traceFuse(path, flags)(traceErrc{&errc}, &fh)
	return self.FileSystemInterface.Open(path, flags)
}

func (self *TraceFs) Getattr(path string, stat *fuse.Stat_t, fh uint64) (errc int) {
	defer traceFuse(path, fh)(traceErrc{&errc}, stat)
	return self.FileSystemInterface.Getattr(path, stat, fh)
}

func (self *TraceFs) Truncate(path string, size int64, fh uint64) (errc int) {
	defer traceFuse(path, size, fh)(traceErrc{&errc})
	return self.FileSystemInterface.Truncate(path, size, fh)
}

func (self *TraceFs) Read(path string, buff []byte, ofst int64, fh uint64) (n int) {
	defer traceFuse(path, ofst, fh)(&n)
	return self.FileSystemInterface.Read(path, buff, ofst, fh)
}

func (self *TraceFs) Write(path string, buff []byte, ofst int64, fh uint64) (n int) {
	defer traceFuse(path, ofst, fh)(&n)
	return self.FileSystemInterface.Write(path, buff, ofst, fh)
}

func (self *TraceFs) Flush(path string, fh uint64) (errc int) {
	defer traceFuse(path, fh)(traceErrc{&errc})
	return self.FileSystemInterface.Flush(path, fh)
}

func (self *TraceFs) Release(path string, fh uint64) (errc int) {
	defer traceFuse(path, fh)(traceErrc{&errc})
	return self.FileSystemInterface.Release(path, fh)
}

func (self *TraceFs) Fsync(path string, datasync bool, fh uint64) (errc int) {
	defer traceFuse(path, datasync, fh)(traceErrc{&errc})
	return self.FileSystemInterface.Fsync(path, datasync, fh)
}

func (self *TraceFs) Opendir(path string) (errc int, fh uint64) {
	defer traceFuse(path)(traceErrc{&errc}, &fh)
	return self.FileSystemInterface.Opendir(path)
}

func (self *TraceFs) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64) (errc int) {
	defer traceFuse(path, ofst, fh)(traceErrc{&errc})
	return self.FileSystemInterface.Readdir(path, fill, ofst, fh)
}

func (self *TraceFs) Releasedir(path string, fh uint64) (errc int) {
	defer traceFuse(path, fh)(traceErrc{&errc})
	return self.FileSystemInterface.Releasedir(path, fh)
}

func (self *TraceFs) Fsyncdir(path string, datasync bool, fh uint64) (errc int) {
	defer traceFuse(path, datasync, fh)(traceErrc{&errc})
	return self.FileSystemInterface.Fsyncdir(path, datasync, fh)
}

func (self *TraceFs) Setxattr(path string, name string, value []byte, flags int) (errc int) {
	defer traceFuse(path, name, value, flags)(traceErrc{&errc})
	return self.FileSystemInterface.Setxattr(path, name, value, flags)
}

func (self *TraceFs) Getxattr(path string, name string) (errc int, xatr []byte) {
	defer traceFuse(path, name)(traceErrc{&errc}, &xatr)
	return self.FileSystemInterface.Getxattr(path, name)
}

func (self *TraceFs) Removexattr(path string, name string) (errc int) {
	defer traceFuse(path, name)(traceErrc{&errc})
	return self.FileSystemInterface.Removexattr(path, name)
}

func (self *TraceFs) Listxattr(path string, fill func(name string) bool) (errc int) {
	defer traceFuse(path)(traceErrc{&errc})
	return self.FileSystemInterface.Listxattr(path, fill)
}

type traceErrc struct {
	v *int
}

func (t traceErrc) GoString() string {
	return fuse.Error(*t.v).Error()
}
