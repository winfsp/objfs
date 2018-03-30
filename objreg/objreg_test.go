/*
 * objreg_test.go
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

package objreg

import (
	"testing"

	"github.com/billziss-gh/golib/errors"
	"github.com/billziss-gh/objfs/errno"
)

func fooCtor(args ...interface{}) (interface{}, error) {
	return "foo", nil
}

func errCtor(args ...interface{}) (interface{}, error) {
	return nil, errors.New("err")
}

func TestObjreg(t *testing.T) {
	registry := NewObjectFactoryRegistry()
	registry.RegisterFactory("foo", fooCtor)
	registry.RegisterFactory("err", errCtor)

	if nil == registry.GetFactory("foo") {
		t.Error()
	}
	if nil != registry.GetFactory("bar") {
		t.Error()
	}
	if nil == registry.GetFactory("err") {
		t.Error()
	}

	obj, err := registry.NewObject("foo")
	if "foo" != obj || nil != err {
		t.Error()
	}
	obj, err = registry.NewObject("bar")
	if nil != obj || nil == err || errno.EINVAL != errors.Attachment(err) {
		t.Error()
	}
	obj, err = registry.NewObject("err")
	if nil != obj || nil == err || nil != errors.Attachment(err) || "err" != err.Error() {
		t.Error()
	}

	registry.UnregisterFactory("foo")
	if nil != registry.GetFactory("foo") {
		t.Error()
	}
	obj, err = registry.NewObject("foo")
	if nil != obj || nil == err || errno.EINVAL != errors.Attachment(err) {
		t.Error()
	}
}
