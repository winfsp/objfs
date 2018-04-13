// +build debug

/*
 * cache_commands.go
 * Additional objfs commands used for testing.
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
	"io"
	"os"
	"path"

	"github.com/billziss-gh/golib/cmd"
	"github.com/billziss-gh/golib/errors"
	"github.com/billziss-gh/objfs/cache"
	"github.com/billziss-gh/objfs/objio"
)

func init() {
	var c *cmd.Cmd
	cmd.Add("cache-statfs\nget storage information", CacheStatfs)
	c = cmd.Add("cache-ls [-l][-n count] path...\nlist files", CacheLs)
	c.Flag.Bool("l", false, "long format")
	c = cmd.Add("cache-stat [-l] path...\ndisplay file information", CacheStat)
	c.Flag.Bool("l", false, "long format")
	cmd.Add("cache-mkdir path...\nmake directories", CacheMkdir)
	cmd.Add("cache-rmdir path...\nremove directories", CacheRmdir)
	cmd.Add("cache-rm path...\nremove files", CacheRm)
	cmd.Add("cache-mv oldpath newpath\nmove (rename) files", CacheMv)
	cmd.Add("cache-get path [local-path]\nget (download) files", CacheGet)
	cmd.Add("cache-put [local-path] path\nput (upload) files", CachePut)
}

func CacheStatfs(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage, &cachePath)

	cmd.Flag.Parse(args)

	if 0 != cmd.Flag.NArg() {
		usage(cmd)
	}

	c, err := openCache(cache.Open)
	if nil != err {
		fail(errors.New("cache-statfs", err))
	}
	defer c.CloseCache()

	info, err := c.Statfs()
	if nil != err {
		fail(errors.New("cache-statfs", err))
	}

	printStorageInfo(info)
}

func CacheLs(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage, &cachePath)

	cmd.Flag.Parse(args)
	long := cmd.GetFlag("l").(bool)

	c, err := openCache(cache.Open)
	if nil != err {
		fail(errors.New("cache-ls", err))
	}
	defer c.CloseCache()

	failed := false

	for _, path := range cmd.Flag.Args() {
		ino, err := c.Open(path)
		if nil == err {
			var infos []objio.ObjectInfo
			infos, err = c.Readdir(ino, 0)
			if nil == err {
				for _, info := range infos {
					printObjectInfo(info, long)
				}
			}

			c.Close(ino)
		}

		if nil != err {
			failed = true
			warn(errors.New("cache-ls "+path, err))
			continue
		}
	}

	if failed {
		os.Exit(1)
	}
}

func CacheStat(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage, &cachePath)

	cmd.Flag.Parse(args)
	long := cmd.GetFlag("l").(bool)

	c, err := openCache(cache.Open)
	if nil != err {
		fail(errors.New("cache-stat", err))
	}
	defer c.CloseCache()

	failed := false

	for _, path := range cmd.Flag.Args() {
		ino, err := c.Open(path)
		if nil == err {
			var info objio.ObjectInfo
			info, err = c.Stat(ino)
			if nil == err {
				printObjectInfo(info, long)
			}

			c.Close(ino)
		}

		if nil != err {
			failed = true
			warn(errors.New("cache-stat "+path, err))
			continue
		}
	}

	if failed {
		os.Exit(1)
	}
}

func CacheMkdir(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage, &cachePath)

	cmd.Flag.Parse(args)

	c, err := openCache(cache.Open)
	if nil != err {
		fail(errors.New("cache-mkdir", err))
	}
	defer c.CloseCache()

	failed := false

	for _, path := range cmd.Flag.Args() {
		ino, err := c.Open(path)
		if nil == err {
			err = c.Make(ino, true)
			c.Close(ino)
		}

		if nil != err {
			failed = true
			warn(errors.New("cache-mkdir "+path, err))
			continue
		}
	}

	if failed {
		os.Exit(1)
	}
}

func CacheRmdir(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage, &cachePath)

	cmd.Flag.Parse(args)

	c, err := openCache(cache.Open)
	if nil != err {
		fail(errors.New("cache-rmdir", err))
	}
	defer c.CloseCache()

	failed := false

	for _, path := range cmd.Flag.Args() {
		ino, err := c.Open(path)
		if nil == err {
			err = c.Remove(ino, true)
			c.Close(ino)
		}

		if nil != err {
			failed = true
			warn(errors.New("cache-rmdir "+path, err))
			continue
		}
	}

	if failed {
		os.Exit(1)
	}
}

func CacheRm(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage, &cachePath)

	cmd.Flag.Parse(args)

	c, err := openCache(cache.Open)
	if nil != err {
		fail(errors.New("cache-rm", err))
	}
	defer c.CloseCache()

	failed := false

	for _, path := range cmd.Flag.Args() {
		ino, err := c.Open(path)
		if nil == err {
			err = c.Remove(ino, false)
			c.Close(ino)
		}

		if nil != err {
			failed = true
			warn(errors.New("cache-rm "+path, err))
			continue
		}
	}

	if failed {
		os.Exit(1)
	}
}

func CacheMv(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage, &cachePath)

	cmd.Flag.Parse(args)

	if 1 > cmd.Flag.NArg() || 2 < cmd.Flag.NArg() {
		usage(cmd)
	}

	oldpath := cmd.Flag.Arg(0)
	newpath := cmd.Flag.Arg(1)

	c, err := openCache(cache.Open)
	if nil != err {
		fail(errors.New("cache-mv", err))
	}
	defer c.CloseCache()

	ino, err := c.Open(oldpath)
	if nil == err {
		err = c.Rename(ino, newpath)
		c.Close(ino)
	}
	if nil != err {
		fail(errors.New("cache-mv "+oldpath, err))
	}
}

func CacheGet(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage, &cachePath)

	cmd.Flag.Parse(args)

	if 1 > cmd.Flag.NArg() || 2 < cmd.Flag.NArg() {
		usage(cmd)
	}

	ipath := cmd.Flag.Arg(0)
	opath := cmd.Flag.Arg(1)
	if "" == opath {
		opath = path.Base(ipath)
	}

	c, err := openCache(cache.Open)
	if nil != err {
		fail(errors.New("cache-get", err))
	}
	defer c.CloseCache()

	ino, err := c.Open(ipath)
	if nil != err {
		fail(errors.New("cache-get "+ipath, err))
	}
	defer c.Close(ino)

	writer, err := os.OpenFile(opath, os.O_CREATE|os.O_WRONLY, 0666)
	if nil != err {
		fail(errors.New("cache-get "+ipath, err))
	}
	defer writer.Close()

	reader := &cacheReader{cache: c, ino: ino}
	_, err = io.Copy(writer, reader)
	if nil != err {
		fail(errors.New("cache-get "+ipath, err))
	}
}

func CachePut(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage, &cachePath)

	cmd.Flag.Parse(args)

	if 1 > cmd.Flag.NArg() || 2 < cmd.Flag.NArg() {
		usage(cmd)
	}

	ipath := cmd.Flag.Arg(0)
	opath := cmd.Flag.Arg(1)
	if "" == opath {
		opath = ipath
		ipath = path.Base(ipath)
	}

	c, err := openCache(cache.Open)
	if nil != err {
		fail(errors.New("cache-put", err))
	}
	defer c.CloseCache()

	reader, err := os.OpenFile(ipath, os.O_RDONLY, 0)
	if nil != err {
		fail(errors.New("cache-put "+opath, err))
	}
	defer reader.Close()

	ino, err := c.Open(opath)
	if nil != err {
		fail(errors.New("cache-put "+opath, err))
	}
	defer c.Close(ino)

	err = c.Make(ino, false)
	if nil != err {
		fail(errors.New("cache-put "+opath, err))
	}

	writer := &cacheWriter{cache: c, ino: ino}
	_, err = io.Copy(writer, reader)
	if nil != err {
		fail(errors.New("cache-put "+opath, err))
	}
}

type cacheReader struct {
	cache *cache.Cache
	ino   uint64
	off   int64
}

func (self *cacheReader) Read(p []byte) (n int, err error) {
	n, err = self.cache.ReadAt(self.ino, p, self.off)
	self.off += int64(n)
	return
}

type cacheWriter struct {
	cache *cache.Cache
	ino   uint64
	off   int64
}

func (self *cacheWriter) Write(p []byte) (n int, err error) {
	n, err = self.cache.WriteAt(self.ino, p, self.off)
	self.off += int64(n)
	return
}
