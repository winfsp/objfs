/*
 * errno.go
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

package errno

import (
	"github.com/billziss-gh/golib/errors"
)

//go:generate $GOPATH/bin/stringer -type=Errno $GOFILE

// Errno contains an error number (code) and implements error.
type Errno int

// Generic errno constants.
const (
	_ Errno = iota
	E2BIG
	EACCES
	EADDRINUSE
	EADDRNOTAVAIL
	EAFNOSUPPORT
	EAGAIN
	EALREADY
	EBADF
	EBADMSG
	EBUSY
	ECANCELED
	ECHILD
	ECONNABORTED
	ECONNREFUSED
	ECONNRESET
	EDEADLK
	EDESTADDRREQ
	EDOM
	EEXIST
	EFAULT
	EFBIG
	EHOSTUNREACH
	EIDRM
	EILSEQ
	EINPROGRESS
	EINTR
	EINVAL
	EIO
	EISCONN
	EISDIR
	ELOOP
	EMFILE
	EMLINK
	EMSGSIZE
	ENAMETOOLONG
	ENETDOWN
	ENETRESET
	ENETUNREACH
	ENFILE
	ENOATTR
	ENOBUFS
	ENODATA
	ENODEV
	ENOENT
	ENOEXEC
	ENOLCK
	ENOLINK
	ENOMEM
	ENOMSG
	ENOPROTOOPT
	ENOSPC
	ENOSR
	ENOSTR
	ENOSYS
	ENOTCONN
	ENOTDIR
	ENOTEMPTY
	ENOTRECOVERABLE
	ENOTSOCK
	ENOTSUP
	ENOTTY
	ENXIO
	EOPNOTSUPP
	EOVERFLOW
	EOWNERDEAD
	EPERM
	EPIPE
	EPROTO
	EPROTONOSUPPORT
	EPROTOTYPE
	ERANGE
	EROFS
	ESPIPE
	ESRCH
	ETIME
	ETIMEDOUT
	ETXTBSY
	EWOULDBLOCK
	EXDEV
)

// ErrnoFromErr converts a Go error to an Errno.
//
// If err is nil then ErrnoFromErr is 0. Otherwise the ErrnoFromErr will be EIO,
// unless a more specific Errno is found in the causal chain of err.
func ErrnoFromErr(err error) (errno Errno) {
	if nil == err {
		return
	}

	errno = EIO
	for e := err; nil != e; e = errors.Cause(e) {
		a := errors.Attachment(e)
		if nil == a {
			a = e
		}

		if i, ok := a.(Errno); ok && EIO != i {
			errno = i
			return
		}
	}

	return
}
