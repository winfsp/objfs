/*
 * tracestg.go
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

package objio

import (
	"fmt"
	"io"

	"github.com/billziss-gh/golib/trace"
)

// TraceObjectStorage wraps a storage and traces calls to it.
type TraceObjectStorage struct {
	ObjectStorage
}

func traceStg(val0 interface{}, vals ...interface{}) func(vals ...interface{}) {
	return trace.Trace(1, fmt.Sprintf("{{yellow}}%T{{off}}", val0), vals...)
}

func (self *TraceObjectStorage) Info(getsize bool) (info StorageInfo, err error) {
	defer traceStg(self.ObjectStorage, getsize)(traceWrap{&info}, traceWrap{&err})
	return self.ObjectStorage.Info(getsize)
}

func (self *TraceObjectStorage) List(
	prefix string, imarker string, maxcount int) (
	omarker string, infos []ObjectInfo, err error) {
	defer traceStg(
		self.ObjectStorage, prefix, imarker, maxcount)(
		&omarker, traceWrap{&infos}, traceWrap{&err})
	return self.ObjectStorage.List(prefix, imarker, maxcount)
}

func (self *TraceObjectStorage) Stat(name string) (info ObjectInfo, err error) {
	defer traceStg(self.ObjectStorage, name)(traceWrap{&info}, traceWrap{&err})
	return self.ObjectStorage.Stat(name)
}

func (self *TraceObjectStorage) Mkdir(prefix string) (info ObjectInfo, err error) {
	defer traceStg(self.ObjectStorage, prefix)(traceWrap{&info}, traceWrap{&err})
	return self.ObjectStorage.Mkdir(prefix)
}

func (self *TraceObjectStorage) Rmdir(prefix string) (err error) {
	defer traceStg(self.ObjectStorage, prefix)(traceWrap{&err})
	return self.ObjectStorage.Rmdir(prefix)
}

func (self *TraceObjectStorage) Remove(name string) (err error) {
	defer traceStg(self.ObjectStorage, name)(traceWrap{&err})
	return self.ObjectStorage.Remove(name)
}

func (self *TraceObjectStorage) Rename(oldname string, newname string) (err error) {
	defer traceStg(self.ObjectStorage, oldname, newname)(traceWrap{&err})
	return self.ObjectStorage.Rename(oldname, newname)
}

func (self *TraceObjectStorage) OpenRead(
	name string, sig string) (
	info ObjectInfo, reader io.ReadCloser, err error) {
	defer traceStg(self.ObjectStorage, name, sig)(traceWrap{&info}, traceWrap{&err})
	return self.ObjectStorage.OpenRead(name, sig)
}

func (self *TraceObjectStorage) OpenWrite(
	name string, size int64) (
	writer WriteWaiter, err error) {
	defer traceStg(self.ObjectStorage, name, size)(traceWrap{&err})
	writer, err = self.ObjectStorage.OpenWrite(name, size)
	if nil == err {
		writer = &traceWriteWaiter{writer}
	}
	return
}

type traceWriteWaiter struct {
	WriteWaiter
}

func (self *traceWriteWaiter) Wait() (info ObjectInfo, err error) {
	defer traceStg(self.WriteWaiter)(traceWrap{&info}, traceWrap{&err})
	return self.WriteWaiter.Wait()
}

type traceWrap struct {
	v interface{}
}

func (t traceWrap) GoString() string {
	switch i := t.v.(type) {
	case *error:
		if nil == *i {
			return "{{bold green}}OK{{off}}"
		}
		return fmt.Sprintf("{{bold red}}error(\"%v\"){{off}}", *i)
	case *StorageInfo:
		return fmt.Sprintf("%#v", *i)
	case *ObjectInfo:
		return fmt.Sprintf("%#v", *i)
	case *[]ObjectInfo:
		return fmt.Sprintf("%T (len=%d)", i, len(*i))
	default:
		return fmt.Sprintf("%#v", t.v)
	}
}
