/*
 * readonly.go
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
	"github.com/billziss-gh/objfs/errno"
)

type readOnlyStorageInfo struct {
	StorageInfo
}

func (self *readOnlyStorageInfo) IsReadOnly() bool {
	return true
}

// ReadOnlyStorage wraps a storage and provides read-only access to it.
type ReadOnlyStorage struct {
	ObjectStorage
}

// Info gets storage information. The getsize parameter instructs the
// implementation to also contact the object storage for size information.
//
// The ReadOnly storage implementation returns true for IsReadOnly.
func (self *ReadOnlyStorage) Info(getsize bool) (info StorageInfo, err error) {
	info, err = self.Info(getsize)
	if nil == err {
		info = &readOnlyStorageInfo{info}
	}
	return
}

// Mkdir makes an object directory if the storage supports it.
//
// The ReadOnly storage implementation returns errno.EROFS.
func (self *ReadOnlyStorage) Mkdir(prefix string) (ObjectInfo, error) {
	return nil, errno.EROFS
}

// Rmdir removes an object directory if the storage supports it.
//
// The ReadOnly storage implementation returns errno.EROFS.
func (self *ReadOnlyStorage) Rmdir(prefix string) error {
	return errno.EROFS
}

// Remove deletes an object from storage.
//
// The ReadOnly storage implementation returns errno.EROFS.
func (self *ReadOnlyStorage) Remove(name string) error {
	return errno.EROFS
}

// Rename renames an object.
//
// The ReadOnly storage implementation returns errno.EROFS.
func (self *ReadOnlyStorage) Rename(oldname string, newname string) error {
	return errno.EROFS
}

// OpenWrite opens an object for writing. The parameter size specifies
// the size that the written object will have.
//
// The ReadOnly storage implementation returns errno.EROFS.
func (self *ReadOnlyStorage) OpenWrite(name string, size int64) (WriteWaiter, error) {
	return nil, errno.EROFS
}

var _ ObjectStorage = (*ReadOnlyStorage)(nil)
