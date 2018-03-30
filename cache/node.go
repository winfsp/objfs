/*
 * node.go
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
	"bytes"
	"os"
	"path"
	"time"

	"github.com/billziss-gh/objfs/errno"
	"github.com/billziss-gh/objfs/objio"
	"github.com/boltdb/bolt"
)

type nodetx_t struct {
	Tx  *bolt.Tx
	idx *bolt.Bucket
	cat *bolt.Bucket
}

func (tx *nodetx_t) Idx() *bolt.Bucket {
	if nil == tx.idx {
		tx.idx = tx.Tx.Bucket(idxname)
	}

	return tx.idx
}

func (tx *nodetx_t) Cat() *bolt.Bucket {
	if nil == tx.cat {
		tx.cat = tx.Tx.Bucket(catname)
	}

	return tx.cat
}

type node_t struct {
	// persistent
	Ino   uint64    // read-only after init
	Path  string    // guarded by lockPath/unlockPath
	Size  int64     //     -ditto-
	Btime time.Time //     -ditto-
	Mtime time.Time //     -ditto-
	IsDir bool      //     -ditto-
	Sig   string    //     -ditto-
	Hash  []byte    //     -ditto-

	// transient
	Valid   bool
	Deleted bool
	File    *os.File
	refcnt  int
}

func (node *node_t) Get(tx *nodetx_t, k []byte) (err error) {
	v := tx.Cat().Get(k)
	if nil == v || nil != node.Decode(v) {
		err = errno.ENOENT
	}

	return
}

func (node *node_t) GetWithIno(tx *nodetx_t, ino uint64) (err error) {
	var vbuf [8]byte
	v := vbuf[:]

	putUint64(v, 0, ino)
	k := tx.Idx().Get(v)
	if nil == k {
		err = errno.ENOENT
		return
	}

	err = node.Get(tx, k)

	return
}

func (node *node_t) NextIno(tx *nodetx_t) (ino uint64, err error) {
	if 0 == node.Ino {
		node.Ino, err = tx.Idx().NextSequence()
		if nil != err {
			return
		}
	}

	ino = node.Ino

	return
}

func (node *node_t) Put(tx *nodetx_t, k []byte) (err error) {
	if nil != node {
		v := make([]byte, node.EncodeLen())
		putUint64(v, 0, node.Ino)

		if k0 := tx.Idx().Get(v[:8]); !bytes.Equal(k0, k) {
			err = tx.Idx().Put(v[:8], k)
		}
		if nil == err {
			v = node.Encode(v)
			err = tx.Cat().Put(k, v)
		}
	} else {
		n := node_t{}
		err = n.Get(tx, k)
		if nil == err {
			var vbuf [8]byte
			v := vbuf[:]

			putUint64(v, 0, n.Ino)
			err = tx.Idx().Delete(v)
			if nil == err {
				err = tx.Cat().Delete(k)
			}
		}
		if errno.ENOENT == err {
			err = nil
		}
	}

	return
}

func (node *node_t) CopyStat(info objio.ObjectInfo) {
	node.Size = info.Size()
	node.Btime = info.Btime()
	node.Mtime = info.Mtime()
	node.IsDir = info.IsDir()
	node.Sig = info.Sig()
	node.Valid = true
}

func (node *node_t) Stat() (info objio.ObjectInfo, err error) {
	if 0 == node.Ino || "" == node.Path || !node.Valid || node.Deleted {
		panic(errno.EINVAL)
	}

	nodeinfo := nodeinfo_t{
		name:  path.Base(node.Path),
		size:  node.Size,
		btime: node.Btime,
		mtime: node.Mtime,
		isdir: node.IsDir,
		sig:   node.Sig,
	}

	if nil != node.File {
		var fileinfo os.FileInfo
		fileinfo, err = node.File.Stat()
		if nil != err {
			return
		}

		nodeinfo.size = fileinfo.Size()
		nodeinfo.mtime = fileinfo.ModTime()
	}

	info = &nodeinfo

	return
}

func (node *node_t) Reference() {
	node.refcnt++
}

func (node *node_t) Dereference() int {
	node.refcnt--
	return node.refcnt
}

func (node *node_t) EncodeLen() int {
	lp, ls, lh := len(node.Path), len(node.Sig), len(node.Hash)
	return 8 + 8 + 8 + 8 + 2 + 2 + 1 + 1 + lp + ls + lh
}

func (node *node_t) Encode(b []byte) []byte {
	// encode order: uint64*, uint32*, uint16*, uint8*
	// Ino, Size, Btime, Mtime, len(Path), len(Sig), IsDir, len(Hash), Path, Sig, Hash

	if 0 == node.Ino || "" == node.Path || !node.Valid || node.Deleted {
		panic(errno.EINVAL)
	}

	isdir := uint8(0)
	if node.IsDir {
		isdir = uint8(1)
	}
	lp, ls, lh := len(node.Path), len(node.Sig), len(node.Hash)

	i := 0
	i = putUint64(b, i, node.Ino)
	i = putUint64(b, i, uint64(node.Size))
	i = putTime(b, i, node.Btime)
	i = putTime(b, i, node.Mtime)
	i = putUint16(b, i, uint16(lp))
	i = putUint16(b, i, uint16(ls))
	i = putUint8(b, i, isdir)
	i = putUint8(b, i, uint8(lh))
	i = putString(b, i, node.Path, 1<<16-1)
	i = putString(b, i, node.Sig, 1<<16-1)
	i = putBytes(b, i, node.Hash, 1<<8-1)
	return b[:i]
}

func (node *node_t) Decode(b []byte) (err error) {
	// encode order: uint64*, uint32*, uint16*, uint8*
	// Ino, Size, Btime, Mtime, len(Path), len(Sig), IsDir, len(Hash), Path, Sig, Hash

	defer func() {
		if r := recover(); nil != r {
			err = errno.EIO
		}
	}()

	i := 0
	i, ino := getUint64(b, i)
	i, size := getUint64(b, i)
	i, btime := getTime(b, i)
	i, mtime := getTime(b, i)
	i, lp := getUint16(b, i)
	i, ls := getUint16(b, i)
	i, isdir := getUint8(b, i)
	i, lh := getUint8(b, i)
	i, path := getString(b, i, int(lp))
	i, sig := getString(b, i, int(ls))
	i, hash := getBytes(b, i, int(lh))

	node.Ino = ino
	node.Size = int64(size)
	node.Btime = btime
	node.Mtime = mtime
	node.IsDir = 0 != isdir
	node.Path = path
	node.Sig = sig
	node.Hash = hash
	node.Valid = true

	return nil
}

type nodeinfo_t struct {
	name  string
	size  int64
	btime time.Time
	mtime time.Time
	isdir bool
	sig   string
}

func (info *nodeinfo_t) Name() string {
	return info.name
}

func (info *nodeinfo_t) Size() int64 {
	return info.size
}

func (info *nodeinfo_t) Btime() time.Time {
	return info.btime
}

func (info *nodeinfo_t) Mtime() time.Time {
	return info.mtime
}

func (info *nodeinfo_t) IsDir() bool {
	return info.isdir
}

func (info *nodeinfo_t) Sig() string {
	return info.sig
}

func putUint8(b []byte, i int, v uint8) int {
	b[i] = byte(v)
	return i + 1
}

func putUint16(b []byte, i int, v uint16) int {
	b[i+0] = byte(v >> 8)
	b[i+1] = byte(v)
	return i + 2
}

func putUint32(b []byte, i int, v uint32) int {
	b[i+0] = byte(v >> 24)
	b[i+1] = byte(v >> 16)
	b[i+2] = byte(v >> 8)
	b[i+3] = byte(v)
	return i + 4
}

func putUint64(b []byte, i int, v uint64) int {
	b[i+0] = byte(v >> 56)
	b[i+1] = byte(v >> 48)
	b[i+2] = byte(v >> 40)
	b[i+3] = byte(v >> 32)
	b[i+4] = byte(v >> 24)
	b[i+5] = byte(v >> 16)
	b[i+6] = byte(v >> 8)
	b[i+7] = byte(v)
	return i + 8
}

func putTime(b []byte, i int, v time.Time) int {
	return putUint64(b, i, uint64(v.UnixNano()))
}

func putString(b []byte, i int, v string, maxlen int) int {
	if maxlen > len(v) {
		maxlen = len(v)
	}
	return i + copy(b[i:], ([]byte)(v[:maxlen]))
}

func putBytes(b []byte, i int, v []byte, maxlen int) int {
	if maxlen > len(v) {
		maxlen = len(v)
	}
	return i + copy(b[i:], v[:maxlen])
}

func getUint8(b []byte, i int) (int, uint8) {
	return i + 1, uint8(b[i])
}

func getUint16(b []byte, i int) (int, uint16) {
	return i + 2, uint16(b[i])<<8 | uint16(b[i+1])
}

func getUint32(b []byte, i int) (int, uint32) {
	return i + 4, uint32(b[i])<<24 | uint32(b[i+1])<<16 | uint32(b[i+2])<<8 | uint32(b[i+3])
}

func getUint64(b []byte, i int) (int, uint64) {
	return i + 8,
		uint64(b[i])<<56 | uint64(b[i+1])<<48 | uint64(b[i+2])<<40 | uint64(b[i+3])<<32 |
			uint64(b[i+4])<<24 | uint64(b[i+5])<<16 | uint64(b[i+6])<<8 | uint64(b[i+7])
}

func getTime(b []byte, i int) (int, time.Time) {
	i, v := getUint64(b, i)
	return i, time.Unix(0, int64(v)).UTC()
}

func getString(b []byte, i int, l int) (int, string) {
	return i + l, string(b[i : i+l])
}

func getBytes(b []byte, i int, l int) (int, []byte) {
	return i + l, b[i : i+l]
}

var (
	idxname = []byte("i")
	catname = []byte("c")
)
