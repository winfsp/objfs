/*
 * commands.go
 * Main objfs commands.
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
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/billziss-gh/golib/cmd"
	"github.com/billziss-gh/golib/errors"
	"github.com/billziss-gh/objfs/auth"
	"github.com/billziss-gh/objfs/cache"
	"github.com/billziss-gh/objfs/fs"
	"github.com/billziss-gh/objfs/objio"
)

type mntopts []string

// String implements flag.Value.String.
func (opts *mntopts) String() string {
	return ""
}

// Set implements flag.Value.Set.
func (opts *mntopts) Set(s string) error {
	*opts = append(*opts, s)
	return nil
}

// Get implements flag.Getter.Get.
func (opts *mntopts) Get() interface{} {
	return *opts
}

func init() {
	var c *cmd.Cmd
	cmd.Add("auth output-credentials\nperform authentication/authorization", Auth)
	c = cmd.Add("mount [-o option...] mountpoint\nmount file system", Mount)
	c.Flag.Var(new(mntopts), "o", "FUSE mount `option`")
	cmd.Add("statfs\nget storage information", Statfs)
	c = cmd.Add("ls [-l][-n count] path...\nlist files", Ls)
	c.Flag.Bool("l", false, "long format")
	c.Flag.Int("n", 0, "max `count` of list entries")
	c = cmd.Add("stat [-l] path...\ndisplay file information", Stat)
	c.Flag.Bool("l", false, "long format")
	cmd.Add("mkdir path...\nmake directories", Mkdir)
	cmd.Add("rmdir path...\nremove directories", Rmdir)
	cmd.Add("rm path...\nremove files", Rm)
	cmd.Add("mv oldpath newpath\nmove (rename) files", Mv)
	c = cmd.Add("get [-r range][-s signature] path [local-path]\nget (download) files", Get)
	c.Flag.String("r", "", "`range` to request (startpos-endpos)")
	c.Flag.String("s", "", "only get file if it does not match `signature`")
	cmd.Add("put [local-path] path\nput (upload) files", Put)
	cmd.Add("cache-pending\nlist pending cache files", CachePending)
	cmd.Add("cache-reset\nreset cache (upload and evict files)", CacheReset)
}

func Auth(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName)

	cmd.Flag.Parse(args)
	opath := cmd.Flag.Arg(0)
	if "" == opath {
		usage(cmd)
	}

	session, err := authSessionMap[storageName](credentials)
	if nil != err {
		fail(errors.New("auth", err))
	}

	err = auth.WriteCredentials(opath, session.Credentials())
	if nil != err {
		fail(errors.New("auth", err))
	}
}

func Mount(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage, &cachePath)

	cmd.Flag.Parse(args)
	opts := cmd.GetFlag("o").(mntopts)
	mountpoint := cmd.Flag.Arg(0)
	if "" == mountpoint {
		usage(cmd)
	}

	for i := range opts {
		opts[i] = "-o" + opts[i]
	}

	c, err := openCache(cache.Activate)
	if nil != err {
		fail(errors.New("mount", err))
	}
	defer c.CloseCache()

	fsys, err := fs.Registry.NewObject("objfs", c)
	if nil != err {
		fail(errors.New("mount", err))
	}

	err = fs.Mount(fsys, cmd.Flag.Arg(0), opts)
	if nil != err {
		fail(errors.New("mount", err))
	}
}

func Statfs(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage)

	cmd.Flag.Parse(args)

	if 0 != cmd.Flag.NArg() {
		usage(cmd)
	}

	info, err := storage.Info(true)
	if nil != err {
		fail(errors.New("statfs", err))
	}

	printStorageInfo(info)
}

func Ls(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage)

	cmd.Flag.Parse(args)
	long := cmd.GetFlag("l").(bool)
	maxcount := cmd.GetFlag("n").(int)
	count := maxcount

	failed := false

	for _, path := range cmd.Flag.Args() {
		marker := ""
		infos := ([]objio.ObjectInfo)(nil)
		for {
			var err error
			marker, infos, err = storage.List(path, marker, count)
			if nil != err {
				failed = true
				warn(errors.New("ls "+path, err))
				break
			}

			for _, info := range infos {
				printObjectInfo(info, long)
			}

			if "" == marker {
				break
			}
			if 0 < maxcount {
				count -= len(infos)
				if 0 >= count {
					break
				}
			}
		}
	}

	if failed {
		os.Exit(1)
	}
}

func Stat(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage)

	cmd.Flag.Parse(args)
	long := cmd.GetFlag("l").(bool)

	failed := false

	for _, path := range cmd.Flag.Args() {
		info, err := storage.Stat(path)
		if nil != err {
			failed = true
			warn(errors.New("stat "+path, err))
			continue
		}

		printObjectInfo(info, long)
	}

	if failed {
		os.Exit(1)
	}
}

func Mkdir(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage)

	cmd.Flag.Parse(args)

	failed := false

	for _, path := range cmd.Flag.Args() {
		info, err := storage.Mkdir(path)
		if nil != err {
			failed = true
			warn(errors.New("mkdir "+path, err))
			continue
		}

		printObjectInfo(info, false)
	}

	if failed {
		os.Exit(1)
	}
}

func Rmdir(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage)

	cmd.Flag.Parse(args)

	failed := false

	for _, path := range cmd.Flag.Args() {
		err := storage.Rmdir(path)
		if nil != err {
			failed = true
			warn(errors.New("rmdir "+path, err))
			continue
		}
	}

	if failed {
		os.Exit(1)
	}
}

func Rm(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage)

	cmd.Flag.Parse(args)

	failed := false

	for _, path := range cmd.Flag.Args() {
		err := storage.Remove(path)
		if nil != err {
			failed = true
			warn(errors.New("rm "+path, err))
			continue
		}
	}

	if failed {
		os.Exit(1)
	}
}

func Mv(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage)

	cmd.Flag.Parse(args)

	if 1 > cmd.Flag.NArg() || 2 < cmd.Flag.NArg() {
		usage(cmd)
	}

	oldpath := cmd.Flag.Arg(0)
	newpath := cmd.Flag.Arg(1)

	err := storage.Rename(oldpath, newpath)
	if nil != err {
		fail(errors.New("mv "+oldpath, err))
	}
}

func Get(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage)

	cmd.Flag.Parse(args)
	rng := cmd.GetFlag("r").(string)
	sig := cmd.GetFlag("s").(string)

	var off, n int64
	if "" != rng {
		_, err := fmt.Sscanf(rng, "%d-%d", &off, &n)
		n -= off
		if nil != err || 0 >= n {
			usage(cmd)
		}
	}

	if 1 > cmd.Flag.NArg() || 2 < cmd.Flag.NArg() {
		usage(cmd)
	}

	ipath := cmd.Flag.Arg(0)
	opath := cmd.Flag.Arg(1)
	if "" == opath {
		opath = path.Base(ipath)
	}

	info, reader, err := storage.OpenRead(ipath, sig)
	if nil != err {
		fail(errors.New("get "+ipath, err))
	}
	if nil == reader {
		fmt.Printf("%s matches sig %s; not downloaded\n", ipath, sig)
		return
	}
	defer reader.Close()

	if "" != rng {
		readat, ok := reader.(io.ReaderAt)
		if !ok {
			fail(errors.New("get " + ipath + "; storage does not implement GET with Range"))
		}
		reader = ioutil.NopCloser(io.NewSectionReader(readat, off, n))
	}

	writer, err := os.OpenFile(opath, os.O_CREATE|os.O_WRONLY, 0666)
	if nil != err {
		fail(errors.New("get "+ipath, err))
	}
	defer writer.Close()

	if "" != rng {
		_, err = writer.Seek(off, io.SeekStart)
		if nil != err {
			fail(errors.New("get "+ipath, err))
		}
	}

	_, err = io.Copy(writer, reader)
	if nil != err {
		fail(errors.New("get "+ipath, err))
	}

	printObjectInfo(info, false)
}

func Put(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage)

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

	reader, err := os.OpenFile(ipath, os.O_RDONLY, 0)
	if nil != err {
		fail(errors.New("put "+opath, err))
	}
	defer reader.Close()

	stat, err := reader.Stat()
	if nil != err {
		fail(errors.New("put "+opath, err))
	}

	writer, err := storage.OpenWrite(opath, stat.Size())
	if nil != err {
		fail(errors.New("put "+opath, err))
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	if nil != err {
		fail(errors.New("put "+opath, err))
	}

	info, err := writer.Wait()
	if nil != err {
		fail(errors.New("put "+opath, err))
	}

	printObjectInfo(info, false)
}

func CachePending(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage, &cachePath)

	cmd.Flag.Parse(args)

	if 0 != cmd.Flag.NArg() {
		usage(cmd)
	}

	fmt.Printf("%s:\n", cachePath)

	c, err := openCache(cache.OpenIfExists)
	if nil != err {
		fail(errors.New("cache-pending", err))
	}
	defer c.CloseCache()

	for _, p := range c.ListCache() {
		fmt.Printf("\t%s\n", p)
	}
}

func CacheReset(cmd *cmd.Cmd, args []string) {
	needvar(&credentials, &storageName, &storage, &cachePath)

	cmd.Flag.Parse(args)

	if 0 != cmd.Flag.NArg() {
		usage(cmd)
	}

	fmt.Printf("%s:\n", cachePath)

	c, err := openCache(cache.Open)
	if nil != err {
		fail(errors.New("cache-reset", err))
	}
	defer c.CloseCache()

	err = c.ResetCache(func(path string) {
		fmt.Printf("\t%s\n", path)
	})
	if nil != err {
		fail(errors.New("cache-reset", err))
	}
}

func printStorageInfo(info objio.StorageInfo) {
	fmt.Printf(`IsCaseInsensitive = %v
IsReadOnly = %v
MaxComponentLength = %v
TotalSize = %v
FreeSize = %v
`,
		info.IsCaseInsensitive(),
		info.IsReadOnly(),
		info.MaxComponentLength(),
		info.TotalSize(),
		info.FreeSize())
}

var dtype = map[bool]string{
	true:  "d",
	false: "-",
}

func printObjectInfo(info objio.ObjectInfo, long bool) {
	if long {
		fmt.Printf("%s %10d %s %s %s %s\n",
			dtype[info.IsDir()],
			info.Size(),
			info.Mtime().Format(time.RFC3339),
			info.Btime().Format(time.RFC3339),
			info.Sig(),
			info.Name())
	} else {
		fmt.Printf("%s %10d %s %s\n",
			dtype[info.IsDir()],
			info.Size(),
			info.Mtime().Format(time.RFC3339),
			info.Name())
	}
}

func openCache(flag int) (*cache.Cache, error) {
	return cache.OpenCache(cachePath, storage, nil, flag)
}
