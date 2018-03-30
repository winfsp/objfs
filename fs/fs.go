/*
 * fs.go
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
	"os"

	"github.com/billziss-gh/cgofuse/fuse"
	"github.com/billziss-gh/golib/errors"
	"github.com/billziss-gh/objfs/errno"
	"github.com/billziss-gh/objfs/objio"
	"github.com/billziss-gh/objfs/objreg"
)

type FileSystemCaseInsensitive interface {
	IsCaseInsensitive() bool
}

// Mount mounts a file system to a mountpoint.
func Mount(fsys interface{}, mountpoint string, opts []string) error {
	fsif, ok := fsys.(fuse.FileSystemInterface)
	if !ok {
		return errors.New(": invalid argument", nil, errno.EINVAL)
	}

	cins := false
	if i, ok := fsys.(FileSystemCaseInsensitive); ok {
		cins = i.IsCaseInsensitive()
	}

	host := fuse.NewFileSystemHost(fsif)
	host.SetCapCaseInsensitive(cins)
	host.SetCapReaddirPlus(true)

	ok = host.Mount(mountpoint, opts)
	if !ok {
		return errors.New(": mount failed", nil, errno.EINVAL)
	}

	return nil
}

// Registry is the default file system factory registry.
var Registry = objreg.NewObjectFactoryRegistry()

func CopyFusestatfsFromStorageInfo(dst *fuse.Statfs_t, src objio.StorageInfo) {
	*dst = fuse.Statfs_t{}
	dst.Frsize = 4096
	dst.Bsize = dst.Frsize
	dst.Blocks = uint64(src.TotalSize()) / dst.Frsize
	dst.Bfree = uint64(src.FreeSize()) / dst.Frsize
	dst.Bavail = dst.Bfree
	dst.Namemax = uint64(src.MaxComponentLength())
}

var startUid = uint32(os.Geteuid())
var startGid = uint32(os.Getegid())

func CopyFusestatFromObjectInfo(dst *fuse.Stat_t, src objio.ObjectInfo) {
	*dst = fuse.Stat_t{}
	dst.Mode = fuse.S_IFREG | 0600
	if src.IsDir() {
		dst.Mode = fuse.S_IFDIR | 0700
	}
	dst.Nlink = 1
	dst.Uid = startUid
	dst.Gid = startGid
	dst.Size = src.Size()
	dst.Mtim = fuse.NewTimespec(src.Mtime())
	dst.Atim = dst.Mtim
	dst.Ctim = dst.Mtim
	dst.Birthtim = fuse.NewTimespec(src.Btime())
}
