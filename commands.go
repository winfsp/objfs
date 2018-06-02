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
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/billziss-gh/golib/cmd"
	"github.com/billziss-gh/golib/config"
	"github.com/billziss-gh/golib/errors"
	"github.com/billziss-gh/golib/keyring"
	"github.com/billziss-gh/golib/terminal"
	"github.com/billziss-gh/golib/terminal/editor"
	"github.com/billziss-gh/golib/util"
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
	initCommands(cmd.DefaultCmdMap)
	addcmd(cmd.DefaultCmdMap, "shell\ninteractive shell", Shell)
}

func initCommands(cmdmap *cmd.CmdMap) {
	var c *cmd.Cmd
	addcmd(cmdmap, "version\nget current version information", Version)
	addcmd(cmdmap,
		"config {get|set|delete} [section.]name [value]\nget or set configuration options",
		Config)
	addcmd(cmdmap, "keyring {get|set|delete} service/user\nget or set keys",
		Keyring)
	addcmd(cmdmap, "auth output-credentials\nperform authentication/authorization", Auth)
	c = addcmd(cmdmap, "mount [-o option...] mountpoint\nmount file system", Mount)
	c.Flag.Var(new(mntopts), "o", "FUSE mount `option`")
	addcmd(cmdmap, "statfs\nget storage information", Statfs)
	c = addcmd(cmdmap, "ls [-l][-n count] path...\nlist files", Ls)
	c.Flag.Bool("l", false, "long format")
	c.Flag.Int("n", 0, "max `count` of list entries")
	c = addcmd(cmdmap, "stat [-l] path...\ndisplay file information", Stat)
	c.Flag.Bool("l", false, "long format")
	addcmd(cmdmap, "mkdir path...\nmake directories", Mkdir)
	addcmd(cmdmap, "rmdir path...\nremove directories", Rmdir)
	addcmd(cmdmap, "rm path...\nremove files", Rm)
	addcmd(cmdmap, "mv oldpath newpath\nmove (rename) files", Mv)
	c = addcmd(cmdmap, "get [-r range][-s signature] path [local-path]\nget (download) files", Get)
	c.Flag.String("r", "", "`range` to request (startpos-endpos)")
	c.Flag.String("s", "", "only get file if it does not match `signature`")
	addcmd(cmdmap, "put [local-path] path\nput (upload) files", Put)
	addcmd(cmdmap, "cache-pending\nlist pending cache files", CachePending)
	addcmd(cmdmap, "cache-reset\nreset cache (upload and evict files)", CacheReset)
}
func Version(cmd *cmd.Cmd, args []string) {
	cmd.Flag.Parse(args)

	if 0 != cmd.Flag.NArg() {
		usage(cmd)
	}

	curryear := time.Now().Year()
	scanyear := 0
	fmt.Sscan(MyCopyright, &scanyear)
	copyright := MyCopyright
	if curryear != scanyear {
		copyright = strings.Replace(MyCopyright,
			fmt.Sprint(scanyear), fmt.Sprintf("%d-%d", scanyear, curryear), 1)
	}

	fmt.Printf("%s - %s; version %s\ncopyright %s\n",
		MyProductName, MyDescription, MyVersion, copyright)
	if "" != MyRepository {
		fmt.Printf("%s\n", MyRepository)
	}

	fmt.Printf("\nsupported storages:\n")
	names := objio.Registry.GetNames()
	sort.Sort(sort.StringSlice(names))
	for _, n := range names {
		if n == defaultStorageName {
			fmt.Printf("  %s (default)\n", n)
		} else {
			fmt.Printf("  %s\n", n)
		}
	}
}

func Config(c *cmd.Cmd, args []string) {
	cmdmap := cmd.NewCmdMap()
	c.Flag.Usage = cmd.UsageFunc(c, cmdmap)

	addcmd(cmdmap, "config.get [section.]name", ConfigGet)
	addcmd(cmdmap, "config.set [section.]name value", ConfigSet)
	addcmd(cmdmap, "config.delete [section.]name", ConfigDelete)

	run(cmdmap, c.Flag, args)
}

func ConfigGet(cmd *cmd.Cmd, args []string) {
	needvar()

	cmd.Flag.Parse(args)
	k := cmd.Flag.Arg(0)
	if "" == k {
		usage(cmd)
	}

	v := programConfig.Get(k)
	if nil != v {
		fmt.Printf("%v\n", v)
	}
}

func ConfigSet(cmd *cmd.Cmd, args []string) {
	needvar()

	cmd.Flag.Parse(args)
	k := cmd.Flag.Arg(0)
	v := cmd.Flag.Arg(1)
	if "" == v {
		if i := strings.IndexByte(k, '='); -1 != i {
			v = k[i+1:]
			k = k[:i]
		}
	}
	if "" == k || "" == v {
		usage(cmd)
	}

	programConfig.Set(k, v)

	util.WriteFunc(configPath, 0600, func(file *os.File) error {
		return config.WriteTyped(file, programConfig)
	})
}

func ConfigDelete(cmd *cmd.Cmd, args []string) {
	needvar()

	cmd.Flag.Parse(args)
	k := cmd.Flag.Arg(0)
	if "" == k {
		usage(cmd)
	}

	programConfig.Delete(k)

	util.WriteFunc(configPath, 0600, func(file *os.File) error {
		return config.WriteTyped(file, programConfig)
	})
}

func Keyring(c *cmd.Cmd, args []string) {
	cmdmap := cmd.NewCmdMap()
	c.Flag.Usage = cmd.UsageFunc(c, cmdmap)

	var c1 *cmd.Cmd
	addcmd(cmdmap, "keyring.get service/user", KeyringGet)
	c1 = addcmd(cmdmap, "keyring.set [-k] service/user", KeyringSet)
	c1.Flag.Bool("k", false, "keep terminating newline when on a terminal")
	addcmd(cmdmap, "keyring.delete service/user", KeyringDelete)

	run(cmdmap, c.Flag, args)
}

func KeyringGet(cmd *cmd.Cmd, args []string) {
	needvar()

	cmd.Flag.Parse(args)

	service := cmd.Flag.Arg(0)
	user := cmd.Flag.Arg(1)
	if "" == user {
		if i := strings.IndexByte(service, '/'); -1 != i {
			user = service[i+1:]
			service = service[:i]
		}
	}
	if "" == service || "" == user {
		usage(cmd)
	}

	pass, err := keyring.Get(service, user)
	if nil != err {
		fail(err)
	}

	os.Stdout.WriteString(pass)
	if !strings.HasSuffix(pass, "\n") && terminal.IsTerminal(os.Stdout.Fd()) {
		os.Stdout.WriteString("\n")
	}
}

func KeyringSet(cmd *cmd.Cmd, args []string) {
	needvar()

	cmd.Flag.Parse(args)
	keep := cmd.GetFlag("k").(bool)

	service := cmd.Flag.Arg(0)
	user := cmd.Flag.Arg(1)
	if "" == user {
		if i := strings.IndexByte(service, '/'); -1 != i {
			user = service[i+1:]
			service = service[:i]
		}
	}
	if "" == service || "" == user {
		usage(cmd)
	}

	pass, err := ioutil.ReadAll(os.Stdin)
	if nil != err {
		fail(err)
	}

	p := string(pass)
	if !keep && terminal.IsTerminal(os.Stdin.Fd()) {
		p = strings.TrimSuffix(p, "\n")
	}

	err = keyring.Set(service, user, p)
	if nil != err {
		fail(err)
	}
}

func KeyringDelete(cmd *cmd.Cmd, args []string) {
	needvar()

	cmd.Flag.Parse(args)

	service := cmd.Flag.Arg(0)
	user := cmd.Flag.Arg(1)
	if "" == user {
		if i := strings.IndexByte(service, '/'); -1 != i {
			user = service[i+1:]
			service = service[:i]
		}
	}
	if "" == service || "" == user {
		usage(cmd)
	}

	err := keyring.Delete(service, user)
	if nil != err {
		fail(err)
	}
}

func Auth(cmd *cmd.Cmd, args []string) {
	needvar(&authSession)

	cmd.Flag.Parse(args)
	opath := cmd.Flag.Arg(0)
	if "" == opath {
		usage(cmd)
	}

	err := auth.WriteCredentials(opath, authSession.Credentials())
	if nil != err {
		fail(errors.New("auth", err))
	}
}

func Mount(cmd *cmd.Cmd, args []string) {
	needvar(&storage, &cachePath)

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
	needvar(&storage)

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
	needvar(&storage)

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
		exit(1)
	}
}

func Stat(cmd *cmd.Cmd, args []string) {
	needvar(&storage)

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
		exit(1)
	}
}

func Mkdir(cmd *cmd.Cmd, args []string) {
	needvar(&storage)

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
		exit(1)
	}
}

func Rmdir(cmd *cmd.Cmd, args []string) {
	needvar(&storage)

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
		exit(1)
	}
}

func Rm(cmd *cmd.Cmd, args []string) {
	needvar(&storage)

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
		exit(1)
	}
}

func Mv(cmd *cmd.Cmd, args []string) {
	needvar(&storage)

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
	needvar(&storage)

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
	needvar(&storage)

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
	needvar(&storage, &cachePath)

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
	needvar(&storage, &cachePath)

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

var shellCommands = []func(cmdmap *cmd.CmdMap){
	initCommands,
}

func Shell(c *cmd.Cmd, args []string) {
	editor.DefaultEditor.History().SetCap(100)
	split_re, _ := regexp.Compile(`\s+`)

	if "windows" == runtime.GOOS {
		fmt.Println("Type \"help\" for help. Type ^Z to quit.")
	} else {
		fmt.Println("Type \"help\" for help. Type ^D to quit.")
	}

	for {
		line, err := editor.DefaultEditor.GetLine("> ")
		if nil != err {
			if io.EOF == err {
				fmt.Println("QUIT")
				return
			}
			fail(err)
		}

		line = strings.TrimSpace(line)
		if "" == line {
			continue
		}
		args = split_re.Split(line, -1)

		cmdmap := cmd.NewCmdMap()
		for _, fn := range shellCommands {
			fn(cmdmap)
		}

		flagSet := flag.NewFlagSet("shell", flag.PanicOnError)
		flagSet.Usage = func() {
			fmt.Fprintln(os.Stderr, "commands:")
			cmdmap.PrintCmds()
		}

		ec := run(cmdmap, flagSet, args)
		if 2 != ec {
			editor.DefaultEditor.History().Add(line)
		}
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
