/*
 * auth_test.go
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

package auth

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func testReadWriteCredentials(t *testing.T, path string) {
	cmap0 := CredentialMap{}
	cmap0["username"] = "emanresu"
	cmap0["password"] = "drowssap"

	err := WriteCredentials(path, cmap0)
	if nil != err {
		t.Error(err)
	}

	cmap, err := ReadCredentials(path)
	if nil != err {
		t.Error(err)
	}

	if !reflect.DeepEqual(cmap0, cmap) {
		t.Error()
	}

	err = DeleteCredentials(path)
	if nil != err {
		t.Error(err)
	}
}

func TestReadWriteCredentials(t *testing.T) {
	testReadWriteCredentials(t, "keyring:objfs/auth_test")

	path := filepath.Join(os.TempDir(), "auth_test")
	os.Remove(path)
	testReadWriteCredentials(t, path)
}
