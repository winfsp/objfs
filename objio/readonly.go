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
	"io"

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
	Storage ObjectStorage
}

// Info gets storage information. The getsize parameter instructs the
// implementation to also contact the object storage for size information.
//
// The ReadOnly storage implementation returns true for IsReadOnly.
func (self *ReadOnlyStorage) Info(getsize bool) (info StorageInfo, err error) {
	info, err = self.Storage.Info(getsize)
	if nil == err {
		info = &readOnlyStorageInfo{info}
	}
	return
}

// IsReadOnly determines if the storage is read-only.
//
// The ReadOnly storage implementation returns true.
func (self *ReadOnlyStorage) IsReadOnly() bool {
	return true
}

// List lists all objects that have names with the specified prefix.
// A marker can be used to continue a paginated listing. The listing
// will contain up to maxcount items; a 0 specifies no limit (but the
// underlying storage may still limit the number of items returned).
// List returns an (optionally empty) marker and a slice of ObjectInfo.
func (self *ReadOnlyStorage) List(
	prefix string, marker string, maxcount int) (string, []ObjectInfo, error) {
	return self.Storage.List(prefix, marker, maxcount)
}

// Stat gets object information.
func (self *ReadOnlyStorage) Stat(name string) (ObjectInfo, error) {
	return self.Storage.Stat(name)
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

// OpenRead opens an object for reading. If sig is not empty, OpenRead
// opens the object only if its current signature is different from sig.
// It returns the current object sig and an io.ReadCloser or any error;
// if the object is not opened because of a matching non-empty sig, a nil
// io.ReadCloser and nil error are returned.
//
// The returned io.ReadCloser may also support the io.ReaderAt interface.
func (self *ReadOnlyStorage) OpenRead(name string, sig string) (ObjectInfo, io.ReadCloser, error) {
	return self.Storage.OpenRead(name, sig)
}

// OpenWrite opens an object for writing. The parameter size specifies
// the size that the written object will have.
//
// The ReadOnly storage implementation returns errno.EROFS.
func (self *ReadOnlyStorage) OpenWrite(name string, size int64) (WriteWaiter, error) {
	return nil, errno.EROFS
}

var _ ObjectStorage = (*ReadOnlyStorage)(nil)
