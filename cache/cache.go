/*
 * cache.go
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
	"unsafe"

	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/billziss-gh/golib/errors"
	"github.com/billziss-gh/objfs/errno"
	"github.com/billziss-gh/objfs/objio"
	"github.com/boltdb/bolt"
)

const Version = 1 // bump version when database format changes

const (
	DefaultNegPathTimeout  = time.Second * 3
	DefaultNegPathMaxCount = 100
	DefaultLoopPeriod      = time.Second * 10
	DefaultUploadDelay     = DefaultLoopPeriod
	DefaultEvictDelay      = DefaultLoopPeriod * 3
)

type Config struct {
	NegPathTimeout  time.Duration
	NegPathMaxCount int
	LoopPeriod      time.Duration
	UploadDelay     time.Duration
	EvictDelay      time.Duration
}

const (
	OpenIfExists = 0
	Open         = 1
	Activate     = 2
)

type Cache struct {
	path      string
	database  *bolt.DB
	storage   objio.ObjectStorage
	config    Config
	isCaseIns bool
	infomux   sync.Mutex
	info      objio.StorageInfo
	infotime  time.Time
	pathmux   sync.Mutex
	pathmap   map[string]*pathmux_t
	openmux   sync.Mutex
	openmap   map[uint64]*node_t
	negmux    sync.Mutex
	negmap    map[string]*negitem_t
	neglst    link_t
	lrumux    sync.Mutex
	rwmap     map[uint64]*lruitem_t
	rwlst     link_t
	romap     map[uint64]*lruitem_t
	rolst     link_t
	done      chan struct{}
	wg        sync.WaitGroup
}

// Cache Locking
//
// Pathlock:
//     /C1/C2/.../Cn: rlock(/) -> rlock(C1) -> rclock(C2) -> ... -> rwlock(Cn)
//
// Hierarchy:
//     - pathlock
//     - [pathlock ->] openmux
//     - [pathlock ->] lrumux

func OpenCache(
	path string, storage objio.ObjectStorage, config *Config, flag int) (
	self *Cache, err error) {

	path, err = filepath.Abs(path)
	if nil != err {
		err = errors.New("", err)
		return
	}

	idxpath := filepath.Join(path, "index")

	if OpenIfExists == flag {
		_, err = os.Stat(idxpath)
		if nil != err {
			err = errors.New("", err)
			return
		}
	}

	info, err := storage.Info(false)
	if nil != err {
		err = errors.New("", err)
		return
	}
	isCaseIns := info.IsCaseInsensitive()

	err = os.MkdirAll(path, 0700)
	if nil != err {
		err = errors.New("", err)
		return
	}

	database, err := bolt.Open(idxpath, 0600, nil)
	if nil != err {
		err = errors.New("", err)
		return
	}

	err = database.Update(func(tx *bolt.Tx) (err error) {
		metaname := []byte("m")
		vername := []byte("version")

		var verbuf [8]byte
		putUint64(verbuf[:], 0, Version)

		var version []byte
		meta := tx.Bucket(metaname)
		if nil != meta {
			version = meta.Get(vername)
		} else if nil == tx.Bucket(idxname) {
			meta, err = tx.CreateBucket(metaname)
			if nil != err {
				return
			}

			version = verbuf[:]
			err = meta.Put(vername, version)
			if nil != err {
				return
			}
		}

		if !bytes.Equal(verbuf[:], version) {
			err = errors.New("incorrect database version")
			return
		}

		_, err = tx.CreateBucketIfNotExists(idxname)
		if nil == err {
			_, err = tx.CreateBucketIfNotExists(catname)
		}
		return
	})
	if nil != err {
		err = errors.New("", err)
		return
	}

	self = &Cache{
		path:      path,
		database:  database,
		storage:   storage,
		isCaseIns: isCaseIns,
		pathmap:   map[string]*pathmux_t{},
		openmap:   map[uint64]*node_t{},
		negmap:    map[string]*negitem_t{},
		rwmap:     map[uint64]*lruitem_t{},
		romap:     map[uint64]*lruitem_t{},
		wg:        sync.WaitGroup{},
	}
	self.neglst.Init()
	self.rwlst.Init()
	self.rolst.Init()

	if nil != config {
		self.config = *config
	}
	if 0 >= self.config.NegPathTimeout {
		self.config.NegPathTimeout = DefaultNegPathTimeout
	}
	if 0 >= self.config.NegPathMaxCount {
		self.config.NegPathMaxCount = DefaultNegPathMaxCount
	}
	if 0 >= self.config.LoopPeriod {
		self.config.LoopPeriod = DefaultLoopPeriod
	}
	if 0 >= self.config.UploadDelay {
		self.config.UploadDelay = DefaultUploadDelay
	}
	if 0 >= self.config.EvictDelay {
		self.config.EvictDelay = DefaultEvictDelay
	}

	self.database.View(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		filepath.Walk(self.path, func(path string, stat os.FileInfo, err error) error {
			if nil != err || stat.IsDir() {
				return nil
			}

			ino, err := self.parseIno(path)
			if nil != err {
				return nil
			}

			n := node_t{}
			err = n.GetWithIno(&ntx, ino)
			if errno.ENOENT == err {
				os.Remove(path)
			}

			return nil
		})

		return
	})

	past := time.Now().Add(-self.config.UploadDelay - self.config.EvictDelay).UnixNano()
	self.database.View(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		cursor := ntx.Cat().Cursor()
		for k, v := cursor.First(); nil != k; k, v = cursor.Next() {
			n := node_t{}
			if nil != n.Decode(v) {
				continue
			}

			item := &lruitem_t{ino: n.Ino, atime: past}
			hash, err0 := hashFile(self.filePath(n.Ino))
			if nil != err0 || bytes.Equal(n.Hash, hash) {
				self.romap[n.Ino] = item
				item.InsertTail(&self.rolst)
			} else {
				self.rwmap[n.Ino] = item
				item.InsertTail(&self.rwlst)
			}
		}

		return
	})

	if Activate == flag {
		_ = self.resetCache(true, nil)

		self.done = make(chan struct{})
		self.wg.Add(1)
		go self.loop()
	}

	return
}

func (self *Cache) Storage() objio.ObjectStorage {
	return self.storage
}

func (self *Cache) ListCache() (paths []string) {
	self.database.View(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		cursor := ntx.Cat().Cursor()
		for k, v := cursor.First(); nil != k; k, v = cursor.Next() {
			n := node_t{}
			if nil != n.Decode(v) {
				continue
			}

			path := n.Path

			hash, err0 := hashFile(self.filePath(n.Ino))
			if nil != err0 {
				path = "-" + path
			} else if bytes.Equal(n.Hash, hash) {
				path = "=" + path
			} else {
				path = "+" + path
			}

			paths = append(paths, path)
		}

		return
	})
	return
}

func (self *Cache) ResetCache(progress func(path string)) (err error) {
	return self.resetCache(true, progress)
}

func (self *Cache) CloseCache() (err error) {
	if nil != self.done {
		close(self.done)
		self.wg.Wait()

		err = self.resetCache(true, nil)
	}

	err0 := self.database.Close()
	if nil == err {
		err = err0
	}

	if nil != err {
		err = errors.New("", err)
	}

	return
}

func (self *Cache) Statfs() (info objio.StorageInfo, err error) {
	now := time.Now()

	self.infomux.Lock()
	info = self.info
	if nil == info || self.infotime.Add(self.config.EvictDelay).Before(now) {
		info, err = self.storage.Info(true)
		if nil == err {
			self.info = info
			self.infotime = now
		}
	}
	self.infomux.Unlock()

	return
}

func (self *Cache) Open(path string) (ino uint64, err error) {
	node, err := self.openNode(path)
	if nil != err {
		ino = ^uint64(0)
		return
	}

	ino = node.Ino

	self.openmux.Lock()
	n := self.openmap[ino]
	if nil == n {
		n = node
		self.openmap[ino] = n
	}
	n.Reference()
	self.openmux.Unlock()

	return
}

func (self *Cache) Make(ino uint64, dir bool) (err error) {
	node, err := self.getOpenNode(ino)
	if nil == err {
		err = self.makeNode(node, dir)
	}

	return
}

func (self *Cache) Remove(ino uint64, dir bool) (err error) {
	node, err := self.getOpenNode(ino)
	if nil == err {
		err = self.removeNode(node, dir)
	}

	return
}

func (self *Cache) Rename(ino uint64, newpath string) (err error) {
	node, err := self.getOpenNode(ino)
	if nil == err {
		err = self.renameNode(node, newpath)
	}

	return
}

func (self *Cache) Stat(ino uint64) (info objio.ObjectInfo, err error) {
	node, err := self.getOpenNode(ino)
	if nil == err {
		info, err = self.statNode(node)
	}

	return
}

func (self *Cache) Chtime(ino uint64, mtime time.Time) (err error) {
	node, err := self.getOpenNode(ino)
	if nil == err {
		err = self.chtimeNode(node, mtime)
	}

	return
}

func (self *Cache) Readdir(ino uint64, maxcount int) (infos []objio.ObjectInfo, err error) {
	node, err := self.getOpenNode(ino)
	if nil == err {
		infos, err = self.readdirNode(node, maxcount)
	}

	return
}

func (self *Cache) ReadAt(ino uint64, buf []byte, off int64) (n int, err error) {
	node, err := self.getOpenNode(ino)
	if nil == err {
		err = self.performFileIoOnNode(node, true, -1, func(file *os.File) (err error) {
			n, err = file.ReadAt(buf, off)
			return
		})
	}

	return
}

func (self *Cache) WriteAt(ino uint64, buf []byte, off int64) (n int, err error) {
	node, err := self.getOpenNode(ino)
	if nil == err {
		err = self.performFileIoOnNode(node, true, -1, func(file *os.File) (err error) {
			n, err = file.WriteAt(buf, off)
			if nil == err {
				self.touchIno(node.Ino, true)
			}
			return
		})
	}

	return
}

func (self *Cache) Truncate(ino uint64, size int64) (err error) {
	node, err := self.getOpenNode(ino)
	if nil == err {
		err = self.performFileIoOnNode(node, true, size, func(file *os.File) (err error) {
			err = file.Truncate(size)
			if nil == err {
				self.touchIno(node.Ino, true)
			}
			return
		})
	}

	return
}

func (self *Cache) Sync(ino uint64) (err error) {
	node, err := self.getOpenNode(ino)
	if nil == err {
		err = self.performFileIoOnNode(node, false, -1, func(file *os.File) (err error) {
			err = file.Sync()
			return
		})
	}

	return
}

func (self *Cache) Close(ino uint64) (err error) {
	closefile := false

	self.openmux.Lock()
	n := self.openmap[ino]
	if nil == n {
		err = errno.EBADF
	} else {
		if 0 == n.Dereference() {
			closefile = true
			delete(self.openmap, ino)
		}
	}
	self.openmux.Unlock()

	if closefile {
		_ = self.closeNode(n)
	}

	return
}

func (self *Cache) getOpenNode(ino uint64) (node *node_t, err error) {
	self.openmux.Lock()
	n := self.openmap[ino]
	if nil == n {
		err = errno.EBADF
	} else {
		node = n
	}
	self.openmux.Unlock()

	return
}

func (self *Cache) openNode(path string) (node *node_t, err error) {
	if !strings.HasPrefix(path, "/") {
		err = errno.ENOENT
		return
	}

	pathKey := self.pathKey(path)
	self.lockPath(pathKey)
	defer self.unlockPath(pathKey)

	k := []byte(pathKey)
	n := node_t{}
	err = self.database.View(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = n.Get(&ntx, k)
		return
	})

	if nil != err {
		if errno.ENOENT != err {
			return
		}

		err = self.database.Update(func(tx *bolt.Tx) (err error) {
			ntx := nodetx_t{Tx: tx}
			_, err = n.NextIno(&ntx)
			return
		})
		if nil != err {
			return
		}

		n.Path = path
	}

	node = &n

	return
}

func (self *Cache) makeNode(node *node_t, dir bool) (err error) {
	pathKey := self.pathKey(node.Path)
	self.lockPath(pathKey)
	defer self.unlockPath(pathKey)

	if node.Deleted {
		err = errno.EPERM
		return
	}

	_ = self.statNodeNoLock(node, pathKey)

	if node.Valid {
		err = errno.EEXIST
		return
	}

	var info objio.ObjectInfo
	if dir {
		info, err = self.storage.Mkdir(node.Path)
	} else {
		var writer objio.WriteWaiter
		writer, err = self.storage.OpenWrite(node.Path, 0)
		if nil != err {
			return
		}
		defer writer.Close()

		info, err = writer.Wait()
	}

	if nil != err {
		return
	}

	n := *node
	n.CopyStat(info)

	k := []byte(pathKey)
	err = self.database.Update(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = n.Put(&ntx, k)
		return
	})
	if nil != err {
		return
	}

	node.CopyStat(info)

	self.removeNegPath(pathKey)

	return
}

func (self *Cache) removeNode(node *node_t, dir bool) (err error) {
	pathKey := self.pathKey(node.Path)
	self.lockPath(pathKey)
	defer self.unlockPath(pathKey)

	if node.Deleted {
		err = errno.EPERM
		return
	}

	k := []byte(pathKey)

	if node.Valid {
		if dir && !node.IsDir {
			err = errno.ENOTDIR
			return
		} else if !dir && node.IsDir {
			err = errno.EISDIR
			return
		}

		nodecnt := 0
		err = self.database.View(func(tx *bolt.Tx) (err error) {
			ntx := nodetx_t{Tx: tx}

			cursor := ntx.Cat().Cursor()
			for i, _ := cursor.Seek(k); nil != i && 1 >= nodecnt; i, _ = cursor.Next() {
				if !pathKeyHasPrefix(i, k) {
					break
				}
				nodecnt++
			}

			return
		})
		if 1 < nodecnt {
			err = errno.ENOTEMPTY
			return
		}
	}

	if dir {
		err = self.storage.Rmdir(node.Path)
	} else {
		err = self.storage.Remove(node.Path)
	}
	if nil != err {
		if errors.HasAttachment(err, errno.ENOENT) {
			// Our view of the file system namespace is inconsistent with the one
			// in the object storage.
			//
			// For Remove we will process ENOENT as if nothing has happened, but
			// we will remember to return ENOENT in the end.
		} else {
			return
		}
	}

	err0 := self.database.Update(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = (*node_t)(nil).Put(&ntx, k)
		return
	})

	if nil == err0 {
		node.Deleted = true
		self.addNegPath(pathKey)
	}
	if nil == err {
		err = err0
	}

	return
}

func (self *Cache) renameNode(node *node_t, newpath string) (err error) {
	oldpath := node.Path

	pathKey := self.pathKey(oldpath)
	newpathKey := self.pathKey(newpath)

	k := []byte(pathKey)
	newk := []byte(newpathKey)

	if pathKey != newpathKey {
		if pathKeyHasPrefix(k, newk) || pathKeyHasPrefix(newk, k) {
			// guard against directory loop creation
			return errno.EINVAL
		}
	}

	self.lockPath(pathKey)
	defer self.unlockPath(pathKey)

	if node.Deleted {
		err = errno.EPERM
		return
	}

	if pathKey != newpathKey {
		// to avoid deadlock only lock when oldpath != newpath;
		// (this can happen during case-sensitivity rename (file->FILE))
		self.lockPath(newpathKey)
		defer self.unlockPath(newpathKey)
	}

	err = self.storage.Rename(oldpath, newpath)
	if nil != err {
		if errors.HasAttachment(err, errno.ENOENT) {
			// Our view of the file system namespace is inconsistent with the one
			// in the object storage.
			//
			// For Rename we will translate ENOENT to EPERM. This is to minimize
			// confusion to the user if we returned ENOENT and then the user saw
			// that the file/directory is still there!
			err = errno.EPERM
		}

		return
	}

	keys := make([][]byte, 0, 128)
	inos := make([]uint64, 0, 128)
	err = self.database.Update(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}

		cursor := ntx.Cat().Cursor()
		for i, _ := cursor.Seek(k); nil != i; i, _ = cursor.Next() {
			if !pathKeyHasPrefix(i, k) {
				break
			}

			keys = append(keys, i)
		}

		var ibuf []byte
		for _, i := range keys {
			n := node_t{}
			err = n.Get(&ntx, i)
			if nil != err {
				return
			}

			// delete old key
			err = (*node_t)(nil).Put(&ntx, i)
			if nil != err {
				return
			}

			l := len(newk) + len(i) - len(k)
			if len(ibuf) < l {
				ibuf = make([]byte, l)
			}
			copy(ibuf, newk)
			copy(ibuf[len(newk):], i[len(k):])
			i = ibuf[:l]

			// delete new key
			err = (*node_t)(nil).Put(&ntx, i)
			if nil != err {
				return
			}

			n.Path = newpath + n.Path[len(oldpath):]

			// place new key
			err = n.Put(&ntx, i)
			if nil != err {
				return
			}

			inos = append(inos, n.Ino)
		}

		return
	})

	if nil != err {
		return
	}

	self.openmux.Lock()
	for _, ino := range inos {
		n := self.openmap[ino]
		if nil != n {
			n.Path = newpath + n.Path[len(oldpath):]
		}
	}
	self.openmux.Unlock()

	if pathKey != newpathKey {
		self.removeAllNegPath(newpathKey)
		self.addNegPath(pathKey)
	}

	return
}

func (self *Cache) statNode(node *node_t) (info objio.ObjectInfo, err error) {
	pathKey := self.pathKey(node.Path)
	self.lockPath(pathKey)
	defer self.unlockPath(pathKey)

	if !node.Valid {
		if node.Deleted {
			err = errno.EPERM
			return
		}

		err = self.statNodeNoLock(node, pathKey)
		if nil != err {
			return
		}
	}

	info, err = node.Stat()

	return
}

func (self *Cache) statNodeNoLock(node *node_t, pathKey string) (err error) {
	if self.isNegPath(pathKey) {
		err = errno.ENOENT
		return
	}

	var i objio.ObjectInfo
	i, err = self.storage.Stat(node.Path)
	if nil != err {
		if errors.HasAttachment(err, errno.ENOENT) {
			self.addNegPath(pathKey)
		}

		return
	}

	n := *node
	n.CopyStat(i)

	k := []byte(pathKey)
	err = self.database.Update(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = n.Put(&ntx, k)
		return
	})
	if nil != err {
		return
	}

	node.CopyStat(i)

	self.removeNegPath(pathKey)

	return
}

func (self *Cache) chtimeNode(node *node_t, mtime time.Time) (err error) {
	err = errno.ENOSYS
	return
}

func (self *Cache) readdirNode(node *node_t, maxcount int) (infos []objio.ObjectInfo, err error) {
	pathKey := self.pathKey(node.Path)
	self.lockPath(pathKey)
	defer self.unlockPath(pathKey)

	if node.Deleted {
		err = errno.EPERM
		return
	}

	count := maxcount
	marker := ""
	for {
		var i []objio.ObjectInfo
		marker, i, err = self.storage.List(node.Path, marker, count)
		if nil != err {
			return
		}

		infos = append(infos, i...)

		if "" == marker {
			break
		}

		if 0 < maxcount {
			count -= len(i)
			if 0 >= count {
				break
			}
		}
	}

	// cache these infos as we may get a flurry of Stat calls
	inos := make([]uint64, 0, len(infos))
	err = self.database.Update(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}

		for _, i := range infos {
			name := i.Name()
			k := []byte(path.Join(pathKey, self.pathKey(name)))

			n := node_t{}
			err = n.Get(&ntx, k)
			if nil == err {
				continue
			}

			var ino uint64
			ino, err = n.NextIno(&ntx)
			if nil != err {
				return
			}

			n.Path = path.Join(node.Path, name)
			n.CopyStat(i)

			err = n.Put(&ntx, k)
			if nil != err {
				return
			}

			inos = append(inos, ino)
		}

		return
	})

	if nil != err {
		infos = nil
		return
	}

	for _, ino := range inos {
		self.touchIno(ino, false)
	}

	self.removeNegPath(pathKey)

	return
}

func (self *Cache) performFileIoOnNode(
	node *node_t, ensure bool, size int64, fn func(file *os.File) error) (
	err error) {

	pathKey := self.pathKey(node.Path)
	self.lockPath(pathKey)
	defer self.unlockPath(pathKey)

	if ensure && nil == node.File {
		if node.Deleted {
			err = errno.EPERM
			return
		}

		var info objio.ObjectInfo
		var hash []byte
		var file *os.File
		info, hash, file, err = self.readNodeFromStorage(node, size)
		if nil != err {
			return
		}

		if nil != info {
			n := *node
			n.CopyStat(info)
			n.Hash = hash

			k := []byte(pathKey)
			err = self.database.Update(func(tx *bolt.Tx) (err error) {
				ntx := nodetx_t{Tx: tx}
				err = n.Put(&ntx, k)
				return
			})
			if nil != err {
				file.Close()
				return
			}

			node.CopyStat(info)
			node.Hash = hash
		}

		node.File = file

		self.removeNegPath(pathKey)
	}

	if nil != node.File {
		err = fn(node.File)
	}

	return
}

func (self *Cache) readNodeFromStorage(
	node *node_t, size int64) (
	info objio.ObjectInfo, hash []byte, file *os.File, err error) {

	filePath := self.filePath(node.Ino)
	sig := node.Sig

	f, err := openFile(filePath, os.O_RDWR, 0600)
	if nil != err {
		sig = ""

		err = os.MkdirAll(filepath.Dir(filePath), 0700)
		if nil != err {
			return
		}

		f, err = openFile(filePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if nil != err {
			return
		}

		defer func() {
			if nil != err {
				os.Remove(filePath)
			}
		}()
	}

	defer func() {
		if nil != err {
			f.Close()
		}
	}()

	i, reader, err := self.storage.OpenRead(node.Path, sig)
	if nil != err {
		return
	}

	if nil != reader {
		defer reader.Close()

		h := sha256.New()
		reader := io.TeeReader(reader, h)

		if -1 == size {
			_, err = io.Copy(f, reader)
		} else {
			_, err = io.CopyN(f, reader, size)
			if io.EOF == err {
				err = nil
			}
		}
		if nil != err {
			return
		}

		info = i
		hash = h.Sum(nil)
	}

	file = f

	return
}

func (self *Cache) closeNode(node *node_t) (err error) {
	if nil != node.File {
		if node.Valid && !node.Deleted {
			err = self.closeAndUpdateNode(node)
		}

		node.File.Close()
		node.File = nil
	}

	if node.Valid {
		self.touchIno(node.Ino, false)
	}

	return
}

func (self *Cache) closeAndUpdateNode(node *node_t) (err error) {
	pathKey := self.pathKey(node.Path)
	self.lockPath(pathKey)
	defer self.unlockPath(pathKey)

	var fileinfo os.FileInfo
	fileinfo, err = node.File.Stat()
	if nil != err {
		return
	}

	n := *node
	n.Size = fileinfo.Size()
	n.Mtime = fileinfo.ModTime()

	k := []byte(pathKey)
	err = self.database.Update(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = n.Put(&ntx, k)
		return
	})
	if nil != err {
		return
	}

	node.Size = n.Size
	node.Mtime = n.Mtime

	return
}

func (self *Cache) isNegPath(pathKey string) (ok bool) {
	self.negmux.Lock()

	item, ok := self.negmap[pathKey]
	if ok {
		if item.atime+int64(self.config.NegPathTimeout) < time.Now().UnixNano() {
			item.Remove()
			delete(self.negmap, pathKey)
			ok = false
		}
	}

	self.negmux.Unlock()

	return
}

func (self *Cache) addNegPath(pathKey string) {
	self.negmux.Lock()

	item := self.negmap[pathKey]
	if nil != item {
		item.Remove()
	} else {
		if self.config.NegPathMaxCount <= len(self.negmap) {
			link := self.neglst.next
			if &self.neglst != link {
				var i *negitem_t
				i = (*negitem_t)(containerOf(unsafe.Pointer(link), unsafe.Offsetof(i.link_t)))
				i.Remove()
				delete(self.negmap, i.pathKey)
			}
		}

		item = &negitem_t{pathKey: pathKey}
		self.negmap[item.pathKey] = item
	}

	item.atime = time.Now().UnixNano()
	item.InsertTail(&self.neglst)

	self.negmux.Unlock()
}

func (self *Cache) removeNegPath(pathKey string) {
	self.negmux.Lock()

	item, ok := self.negmap[pathKey]
	if ok {
		item.Remove()
		delete(self.negmap, pathKey)
	}

	self.negmux.Unlock()
}

func (self *Cache) removeAllNegPath(pathKey string) {
	self.negmux.Lock()

	// just throw away the whole map for now!
	self.negmap = map[string]*negitem_t{}
	self.neglst.Init()

	self.negmux.Unlock()
}

func (self *Cache) touchIno(ino uint64, rw bool) {
	self.lrumux.Lock()

	var thismap, thatmap map[uint64]*lruitem_t
	var thislst *link_t

	if rw {
		thismap, thatmap = self.rwmap, self.romap
		thislst = &self.rwlst
	} else {
		thismap, thatmap = self.romap, self.rwmap
		thislst = &self.rolst
	}

	item := thismap[ino]
	if nil != item {
		item.Remove()
	} else {
		item = thatmap[ino]
		if nil != item {
			if !rw {
				// The ino is a candidate for upload; we cannot evict it.
				self.lrumux.Unlock()
				return
			}
			item.Remove()
			delete(thatmap, ino)
		} else {
			item = &lruitem_t{ino: ino}
		}
		thismap[ino] = item
	}

	item.atime = time.Now().UnixNano()
	item.InsertTail(thislst)

	self.lrumux.Unlock()
}

func (self *Cache) uploadOne(force bool, progress func(path string)) (err error) {
	self.lrumux.Lock()

	var item *lruitem_t
	link := self.rwlst.next
	if &self.rwlst != link {
		var i *lruitem_t
		i = (*lruitem_t)(containerOf(unsafe.Pointer(link), unsafe.Offsetof(i.link_t)))

		if force || i.atime+int64(self.config.UploadDelay) < time.Now().UnixNano() {
			i.Remove()
			delete(self.rwmap, i.ino)
			item = i
		}
	}

	self.lrumux.Unlock()

	if nil == item {
		err = errNoItem
		return
	}

	touch := true
	defer func() {
		if touch {
			// If there was an error enter ino in rwlst to retry the upload.
			// If there was no error enter ino in rolst to evict it.
			self.touchIno(item.ino, nil != err)
		}
	}()

	n, pathKey, err := self.getLockedNodeWithIno(item.ino)
	if nil != err {
		if errno.ENOENT == err {
			err = nil
		}
		return
	}
	defer self.unlockPath(pathKey)

	if n.Deleted {
		// bail if the node is Deleted
		touch = false
		return
	}

	self.lrumux.Lock()
	_, ok := self.rwmap[item.ino]
	self.lrumux.Unlock()
	if ok {
		// bail if the ino snuck back in
		touch = false
		return
	}

	filePath := self.filePath(item.ino)

	file, err := openFile(filePath, os.O_RDONLY, 0)
	if nil != err {
		if os.IsNotExist(err) {
			err = nil
		}
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if nil != err {
		return
	}

	writer, err := self.storage.OpenWrite(n.Path, stat.Size())
	if nil != err {
		return
	}
	defer writer.Close()

	h := sha256.New()
	reader := io.TeeReader(file, h)

	_, err = io.CopyN(writer, reader, stat.Size())
	if nil != err {
		return
	}

	info, err := writer.Wait()
	if nil != err {
		return
	}

	mtime := info.Mtime()
	err = os.Chtimes(filePath, mtime, mtime)
	if nil != err {
		return
	}

	n.CopyStat(info)
	n.Hash = h.Sum(nil)

	k := []byte(pathKey)
	err = self.database.Update(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = n.Put(&ntx, k)
		return
	})

	if nil != err {
		return
	}

	self.openmux.Lock()
	node := self.openmap[item.ino]
	if nil != node {
		node.CopyStat(info)
		node.Hash = n.Hash
	}
	self.openmux.Unlock()

	if nil != progress {
		progress("+" + n.Path)
	}

	return
}

func (self *Cache) uploadAll(force bool, progress func(path string)) (err error) {
	for {
		err = self.uploadOne(force, progress)
		if nil != err {
			break
		}
	}

	if errNoItem == err {
		err = nil
	}

	return
}

func (self *Cache) evictOne(force bool, progress func(path string)) (err error) {
	self.lrumux.Lock()

	var item *lruitem_t
	link := self.rolst.next
	if &self.rolst != link {
		var i *lruitem_t
		i = (*lruitem_t)(containerOf(unsafe.Pointer(link), unsafe.Offsetof(i.link_t)))

		if force || i.atime+int64(self.config.EvictDelay) < time.Now().UnixNano() {
			i.Remove()
			delete(self.romap, i.ino)
			item = i
		}
	}

	self.lrumux.Unlock()

	if nil == item {
		err = errNoItem
		return
	}

	defer func() {
		// If there was an error enter ino in rolst to retry the evict.
		if nil != err {
			self.touchIno(item.ino, false)
		}
	}()

	n, pathKey, err := self.getLockedNodeWithIno(item.ino)
	if nil != err {
		if errno.ENOENT != err {
			return
		}

		err = nil
		os.Remove(self.filePath(item.ino))

		return
	}
	defer self.unlockPath(pathKey)

	self.lrumux.Lock()
	_, ok1 := self.rwmap[item.ino]
	_, ok2 := self.romap[item.ino]
	self.lrumux.Unlock()
	if ok1 || ok2 {
		// bail if the ino snuck back in
		return
	}

	self.openmux.Lock()
	_, ok1 = self.openmap[item.ino]
	self.openmux.Unlock()
	if ok1 {
		// bail if the ino is still open
		//self.touchIno(item.ino, false)
		return
	}

	k := []byte(pathKey)
	err = self.database.Update(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = ((*node_t)(nil)).Put(&ntx, k)
		return
	})

	if nil != err {
		return
	}

	os.Remove(self.filePath(item.ino))

	if nil != progress {
		progress("-" + n.Path)
	}

	return
}

func (self *Cache) evictAll(force bool, progress func(path string)) (err error) {
	for {
		err = self.evictOne(force, progress)
		if nil != err {
			break
		}
	}

	if errNoItem == err {
		err = nil
	}

	return
}

// Get and lock node from ino; ensure that proper node.Path gets locked!
func (self *Cache) getLockedNodeWithIno(ino uint64) (node *node_t, pathKey string, err error) {
	pk0 := ""
	for {
		n := node_t{}
		err = self.database.View(func(tx *bolt.Tx) (err error) {
			ntx := nodetx_t{Tx: tx}
			err = n.GetWithIno(&ntx, ino)
			return
		})

		if nil != err {
			if "" != pk0 {
				self.unlockPath(pk0)
			}
			return
		}

		pk := self.pathKey(n.Path)
		if pk0 == pk {
			node = &n
			pathKey = pk
			return
		}

		if "" != pk0 {
			self.unlockPath(pk0)
		}
		self.lockPath(pk)

		pk0 = pk
	}
}

func (self *Cache) resetCache(force bool, progress func(path string)) (err error) {
	err = self.uploadAll(force, progress)
	if nil == err {
		err = self.evictAll(force, progress)
	}

	if nil != err {
		err = errors.New("", err)
	}

	return
}

func (self *Cache) loop() {
	defer self.wg.Done()

	ticker := time.NewTicker(self.config.LoopPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			self.resetCache(false, nil)
		case <-self.done:
			return
		}
	}
}

func (self *Cache) filePath(ino uint64) string {
	return filepath.Join(self.path, fmt.Sprintf("%02x%c%014x", ino&0xff, os.PathSeparator, ino>>8))
}

func (self *Cache) parseIno(path string) (ino uint64, err error) {
	pathsep := string(os.PathSeparator)

	if strings.HasPrefix(path, self.path) {
		path = path[len(self.path):]
		path = strings.TrimPrefix(path, pathsep)
	} else {
		path = "INV"
	}

	dir, base := filepath.Split(path)
	dir = strings.TrimSuffix(dir, pathsep)
	ino, err = strconv.ParseUint(base+dir, 16, 64)
	return
}

func (self *Cache) pathKey(path string) string {
	if self.isCaseIns {
		return normalizeCase(path)
	} else {
		return path
	}
}

func (self *Cache) lockPath(path string) {
	paths := partialPaths(path)

	for i := 0; len(paths) > i; i++ {
		partial := paths[i]

		self.pathmux.Lock()
		pathmux := self.pathmap[partial]
		if nil == pathmux {
			pathmux = &pathmux_t{}
			self.pathmap[partial] = pathmux
		}
		pathmux.refcnt++
		self.pathmux.Unlock()

		if i == len(paths)-1 {
			pathmux.mux.Lock()
		} else {
			pathmux.mux.RLock()
		}
	}
}

func (self *Cache) unlockPath(path string) {
	paths := partialPaths(path)

	for i := len(paths) - 1; 0 <= i; i-- {
		partial := paths[i]

		self.pathmux.Lock()
		pathmux := self.pathmap[partial]
		pathmux.refcnt--
		if 0 == pathmux.refcnt {
			delete(self.pathmap, partial)
		}
		self.pathmux.Unlock()

		if i == len(paths)-1 {
			pathmux.mux.Unlock()
		} else {
			pathmux.mux.RUnlock()
		}
	}
}

func partialPaths(path string) []string {
	paths := make([]string, 0, 16)
	paths = append(paths, "/")

	partial := ""
	i, comp := 0, ""
	for ; len(path) > i && '/' == path[i]; i++ {
	}
	for {
		j := i
		for ; len(path) > i && '/' != path[i]; i++ {
		}
		comp = path[j:i]
		for ; len(path) > i && '/' == path[i]; i++ {
		}

		if "" == comp {
			break
		}

		partial += "/" + comp
		paths = append(paths, partial)
	}

	return paths
}

func pathKeyHasPrefix(a []byte, b []byte) bool {
	alen := len(a)
	blen := len(b)
	return alen >= blen && bytes.Equal(a[:blen], b) &&
		(alen == blen || (1 == blen && '/' == b[0]) || '/' == a[blen])
}

func hashFile(path string) (hash []byte, err error) {
	file, err := openFile(path, os.O_RDONLY, 0)
	if nil != err {
		return
	}
	defer file.Close()

	h := sha256.New()
	_, err = io.Copy(h, file)
	if nil != err {
		return
	}

	hash = h.Sum(nil)

	return
}

func normalizeCase(s string) string {
	b := strings.Builder{}
	b.Grow(len(s))
	for _, r := range s {
		if 'a' <= r && r <= 'z' {
			r += 'A' - 'a'
		} else if r < 128 {
		} else {
			f := unicode.SimpleFold(r)
			for f > r {
				r = f
				f = unicode.SimpleFold(r)
			}
			r = f
		}
		b.WriteRune(r)
	}
	return b.String()
}

type pathmux_t struct {
	mux    sync.RWMutex
	refcnt int
}

type negitem_t struct {
	link_t
	pathKey string
	atime   int64
}

type lruitem_t struct {
	link_t
	ino   uint64
	atime int64
}

var errNoItem = errors.New("")
