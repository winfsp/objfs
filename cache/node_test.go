/*
 * node_test.go
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
	"path/filepath"
	"testing"
	"time"

	"github.com/billziss-gh/objfs/errno"
	"github.com/boltdb/bolt"
)

func TestStat(t *testing.T) {
	now := time.Now().UTC()
	i := nodeinfo_t{
		size:  0x5152535455565758,
		btime: now,
		mtime: now,
		isdir: true,
		sig:   "fortytwo",
	}

	n := node_t{
		Ino:  0x4142434445464748,
		Path: "/foo/Δοκιμή/bar",
	}

	n.CopyStat(&i)

	info, err := n.Stat()
	if nil != err {
		t.Error(err)
	}

	if path.Base(n.Path) != info.Name() {
		t.Error()
	}

	if n.Size != info.Size() {
		t.Error()
	}

	if !n.Btime.Equal(info.Btime()) || n.Btime.String() != info.Btime().String() {
		t.Error()
	}

	if !n.Mtime.Equal(info.Mtime()) || n.Mtime.String() != info.Mtime().String() {
		t.Error()
	}

	if n.IsDir != info.IsDir() {
		t.Error()
	}

	if n.Sig != info.Sig() {
		t.Error()
	}
}

func TestEncodeDecode(t *testing.T) {
	now := time.Now().UTC()
	n := node_t{
		Ino:   0x4142434445464748,
		Path:  "/foo/Δοκιμή/bar",
		Size:  0x5152535455565758,
		Btime: now,
		Mtime: now,
		IsDir: true,
		Sig:   "fortytwo",
		Hash:  []byte{41, 42, 43, 44},
		Valid: true,
	}

	b := make([]byte, n.EncodeLen())
	b = n.Encode(b)
	if len(b) != n.EncodeLen() {
		t.Error()
	}

	n2 := node_t{}
	err := n2.Decode(b)
	if nil != err {
		t.Error(err)
	}

	if n.Ino != n2.Ino {
		t.Error()
	}

	if n.Path != n2.Path {
		t.Error()
	}

	if n.Size != n2.Size {
		t.Error()
	}

	if !n.Btime.Equal(n2.Btime) || n.Btime.String() != n2.Btime.String() {
		t.Error()
	}

	if !n.Mtime.Equal(n2.Mtime) || n.Mtime.String() != n2.Mtime.String() {
		t.Error()
	}

	if n.IsDir != n2.IsDir {
		t.Error()
	}

	if n.Sig != n2.Sig {
		t.Error()
	}

	if !bytes.Equal(n.Hash, n2.Hash) {
		t.Error()
	}

	if true != n2.Valid {
		t.Error()
	}
}

func TestPutGetDelete(t *testing.T) {
	path := filepath.Join(os.TempDir(), "cache_node_test")
	os.Remove(path)
	defer os.Remove(path)

	now := time.Now().UTC()
	n := node_t{
		Ino:   0x4142434445464748,
		Path:  "/foo/Δοκιμή/bar",
		Size:  0x5152535455565758,
		Btime: now,
		Mtime: now,
		IsDir: true,
		Sig:   "fortytwo",
		Hash:  []byte{41, 42, 43, 44},
		Valid: true,
	}

	db, err := bolt.Open(path, 0600, nil)
	if nil != err {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) (err error) {
		_, err = tx.CreateBucketIfNotExists(idxname)
		if nil == err {
			_, err = tx.CreateBucketIfNotExists(catname)
		}
		return
	})
	if nil != err {
		t.Fatal(err)
	}

	key := []byte("KEY")

	err = db.Update(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = n.Put(&ntx, key)
		return
	})
	if nil != err {
		t.Error(err)
	}

	n2 := node_t{}
	err = db.View(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = n2.Get(&ntx, key)
		return
	})
	if nil != err {
		t.Error(err)
	}

	if n.Ino != n2.Ino {
		t.Error()
	}

	if n.Path != n2.Path {
		t.Error()
	}

	if n.Size != n2.Size {
		t.Error()
	}

	if !n.Btime.Equal(n2.Btime) || n.Btime.String() != n2.Btime.String() {
		t.Error()
	}

	if !n.Mtime.Equal(n2.Mtime) || n.Mtime.String() != n2.Mtime.String() {
		t.Error()
	}

	if n.IsDir != n2.IsDir {
		t.Error()
	}

	if n.Sig != n2.Sig {
		t.Error()
	}

	if !bytes.Equal(n.Hash, n2.Hash) {
		t.Error()
	}

	if true != n2.Valid {
		t.Error()
	}

	n2 = node_t{}
	err = db.View(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = n2.GetWithIno(&ntx, 0x4142434445464748)
		return
	})
	if nil != err {
		t.Error(err)
	}

	if n.Ino != n2.Ino {
		t.Error()
	}

	if n.Path != n2.Path {
		t.Error()
	}

	if n.Size != n2.Size {
		t.Error()
	}

	if !n.Btime.Equal(n2.Btime) || n.Btime.String() != n2.Btime.String() {
		t.Error()
	}

	if !n.Mtime.Equal(n2.Mtime) || n.Mtime.String() != n2.Mtime.String() {
		t.Error()
	}

	if n.IsDir != n2.IsDir {
		t.Error()
	}

	if n.Sig != n2.Sig {
		t.Error()
	}

	if !bytes.Equal(n.Hash, n2.Hash) {
		t.Error()
	}

	if true != n2.Valid {
		t.Error()
	}

	err = db.Update(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = (*node_t)(nil).Put(&ntx, key)
		return
	})
	if nil != err {
		t.Error(err)
	}

	n2 = node_t{}
	err = db.View(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = n2.Get(&ntx, key)
		return
	})
	if errno.ENOENT != err {
		t.Error(err)
	}

	n2 = node_t{}
	err = db.View(func(tx *bolt.Tx) (err error) {
		ntx := nodetx_t{Tx: tx}
		err = n2.GetWithIno(&ntx, 0x4142434445464748)
		return
	})
	if errno.ENOENT != err {
		t.Error(err)
	}
}
