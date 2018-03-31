/*
 * errc.go
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

package fs

import (
	"github.com/billziss-gh/cgofuse/fuse"
	"github.com/billziss-gh/golib/errors"
	"github.com/billziss-gh/objfs/errno"
)

var errnomap = map[errno.Errno]int{
	errno.E2BIG:           -fuse.E2BIG,
	errno.EACCES:          -fuse.EACCES,
	errno.EADDRINUSE:      -fuse.EADDRINUSE,
	errno.EADDRNOTAVAIL:   -fuse.EADDRNOTAVAIL,
	errno.EAFNOSUPPORT:    -fuse.EAFNOSUPPORT,
	errno.EAGAIN:          -fuse.EAGAIN,
	errno.EALREADY:        -fuse.EALREADY,
	errno.EBADF:           -fuse.EBADF,
	errno.EBADMSG:         -fuse.EBADMSG,
	errno.EBUSY:           -fuse.EBUSY,
	errno.ECANCELED:       -fuse.ECANCELED,
	errno.ECHILD:          -fuse.ECHILD,
	errno.ECONNABORTED:    -fuse.ECONNABORTED,
	errno.ECONNREFUSED:    -fuse.ECONNREFUSED,
	errno.ECONNRESET:      -fuse.ECONNRESET,
	errno.EDEADLK:         -fuse.EDEADLK,
	errno.EDESTADDRREQ:    -fuse.EDESTADDRREQ,
	errno.EDOM:            -fuse.EDOM,
	errno.EEXIST:          -fuse.EEXIST,
	errno.EFAULT:          -fuse.EFAULT,
	errno.EFBIG:           -fuse.EFBIG,
	errno.EHOSTUNREACH:    -fuse.EHOSTUNREACH,
	errno.EIDRM:           -fuse.EIDRM,
	errno.EILSEQ:          -fuse.EILSEQ,
	errno.EINPROGRESS:     -fuse.EINPROGRESS,
	errno.EINTR:           -fuse.EINTR,
	errno.EINVAL:          -fuse.EINVAL,
	errno.EISCONN:         -fuse.EISCONN,
	errno.EISDIR:          -fuse.EISDIR,
	errno.ELOOP:           -fuse.ELOOP,
	errno.EMFILE:          -fuse.EMFILE,
	errno.EMLINK:          -fuse.EMLINK,
	errno.EMSGSIZE:        -fuse.EMSGSIZE,
	errno.ENAMETOOLONG:    -fuse.ENAMETOOLONG,
	errno.ENETDOWN:        -fuse.ENETDOWN,
	errno.ENETRESET:       -fuse.ENETRESET,
	errno.ENETUNREACH:     -fuse.ENETUNREACH,
	errno.ENFILE:          -fuse.ENFILE,
	errno.ENOATTR:         -fuse.ENOATTR,
	errno.ENOBUFS:         -fuse.ENOBUFS,
	errno.ENODATA:         -fuse.ENODATA,
	errno.ENODEV:          -fuse.ENODEV,
	errno.ENOENT:          -fuse.ENOENT,
	errno.ENOEXEC:         -fuse.ENOEXEC,
	errno.ENOLCK:          -fuse.ENOLCK,
	errno.ENOLINK:         -fuse.ENOLINK,
	errno.ENOMEM:          -fuse.ENOMEM,
	errno.ENOMSG:          -fuse.ENOMSG,
	errno.ENOPROTOOPT:     -fuse.ENOPROTOOPT,
	errno.ENOSPC:          -fuse.ENOSPC,
	errno.ENOSR:           -fuse.ENOSR,
	errno.ENOSTR:          -fuse.ENOSTR,
	errno.ENOSYS:          -fuse.ENOSYS,
	errno.ENOTCONN:        -fuse.ENOTCONN,
	errno.ENOTDIR:         -fuse.ENOTDIR,
	errno.ENOTEMPTY:       -fuse.ENOTEMPTY,
	errno.ENOTRECOVERABLE: -fuse.ENOTRECOVERABLE,
	errno.ENOTSOCK:        -fuse.ENOTSOCK,
	errno.ENOTSUP:         -fuse.ENOTSUP,
	errno.ENOTTY:          -fuse.ENOTTY,
	errno.ENXIO:           -fuse.ENXIO,
	errno.EOPNOTSUPP:      -fuse.EOPNOTSUPP,
	errno.EOVERFLOW:       -fuse.EOVERFLOW,
	errno.EOWNERDEAD:      -fuse.EOWNERDEAD,
	errno.EPERM:           -fuse.EPERM,
	errno.EPIPE:           -fuse.EPIPE,
	errno.EPROTO:          -fuse.EPROTO,
	errno.EPROTONOSUPPORT: -fuse.EPROTONOSUPPORT,
	errno.EPROTOTYPE:      -fuse.EPROTOTYPE,
	errno.ERANGE:          -fuse.ERANGE,
	errno.EROFS:           -fuse.EROFS,
	errno.ESPIPE:          -fuse.ESPIPE,
	errno.ESRCH:           -fuse.ESRCH,
	errno.ETIME:           -fuse.ETIME,
	errno.ETIMEDOUT:       -fuse.ETIMEDOUT,
	errno.ETXTBSY:         -fuse.ETXTBSY,
	errno.EWOULDBLOCK:     -fuse.EWOULDBLOCK,
	errno.EXDEV:           -fuse.EXDEV,

	//errno.EIO:           -fuse.EIO,
}

// FuseErrc converts a Go error to a FUSE error code.
// If err is nil then FuseErrc is 0. Otherwise the FuseErrc will be -EIO,
// unless a more specific FuseErrc is found in the causal chain of err.
func FuseErrc(err error) (errc int) {
	if nil == err {
		return
	}

	Tracef("%+v", err)

	errc = -fuse.EIO
	for e := err; nil != e; e = errors.Cause(e) {
		a := errors.Attachment(e)
		if nil == a {
			a = e
		}

		switch i := a.(type) {
		case errno.Errno:
			if rc, ok := errnomap[i]; ok {
				errc = rc
				return
			}
		}
	}

	return
}
