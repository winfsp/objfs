/*
 * objio.go
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
	"time"

	"github.com/billziss-gh/objfs/objreg"
)

// StorageInfo contains information about the storage.
type StorageInfo interface {
	// determines if the storage is case-insensitive
	IsCaseInsensitive() bool

	// determines if the storage is read-only
	IsReadOnly() bool

	// maximum object name component length
	MaxComponentLength() int

	// total storage size
	TotalSize() int64

	// free storage size
	FreeSize() int64
}

// ObjectInfo contains information about an object.
type ObjectInfo interface {
	// object name (no path)
	Name() string

	// object size
	Size() int64

	// object birth time
	Btime() time.Time

	// object modification time
	Mtime() time.Time

	// isdir flag
	IsDir() bool

	// object signature
	Sig() string
}

// WriteWaiter wraps a WriteCloser and a Wait method that waits until
// all transfers are complete. After Wait has been called no further
// Write's are possible and Close must be called. Calling Close without
// Wait cancels any pending tranfers.
type WriteWaiter interface {
	io.WriteCloser
	Wait() (ObjectInfo, error)
}

// ObjectStorage is the interface that an object storage must implement.
// It provides methods to list, open and manipulate objects.
type ObjectStorage interface {
	// Info gets storage information. The getsize parameter instructs the
	// implementation to also contact the object storage for size information.
	Info(getsize bool) (StorageInfo, error)

	// List lists all objects that have names with the specified prefix.
	// A marker can be used to continue a paginated listing. The listing
	// will contain up to maxcount items; a 0 specifies no limit (but the
	// underlying storage may still limit the number of items returned).
	// List returns an (optionally empty) marker and a slice of ObjectInfo.
	List(prefix string, marker string, maxcount int) (string, []ObjectInfo, error)

	// Stat gets object information.
	Stat(name string) (ObjectInfo, error)

	// Mkdir makes an object directory if the storage supports it.
	Mkdir(prefix string) (ObjectInfo, error)

	// Rmdir removes an object directory if the storage supports it.
	Rmdir(prefix string) error

	// Remove deletes an object from storage.
	Remove(name string) error

	// Rename renames an object.
	Rename(oldname string, newname string) error

	// OpenRead opens an object for reading. If sig is not empty, OpenRead
	// opens the object only if its current signature is different from sig.
	// It returns the current object info and an io.ReadCloser or any error;
	// if the object is not opened because of a matching non-empty sig, a nil
	// io.ReadCloser and nil error are returned.
	//
	// The returned io.ReadCloser may also support the io.ReaderAt interface.
	OpenRead(name string, sig string) (ObjectInfo, io.ReadCloser, error)

	// OpenWrite opens an object for writing. The parameter size specifies
	// the size that the written object will have.
	OpenWrite(name string, size int64) (WriteWaiter, error)
}

// Registry is the default object storage factory registry.
var Registry = objreg.NewObjectFactoryRegistry()
