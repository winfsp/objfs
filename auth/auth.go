/*
 * auth.go
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
	"bytes"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/billziss-gh/golib/config"
	"github.com/billziss-gh/golib/keyring"
	"github.com/billziss-gh/golib/util"
	"github.com/billziss-gh/objfs/objreg"
)

// CredentialMap maps names to credentials.
type CredentialMap map[string]interface{}

// Get gets a credential by name. It will convert the credential
// to a string using fmt.Sprint if required.
func (self CredentialMap) Get(name string) string {
	v := self[name]
	if nil == v {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// String implements flag.Value.String.
func (self *CredentialMap) String() string {
	return ""
}

// Set implements flag.Value.Set.
func (self *CredentialMap) Set(s string) (err error) {
	*self, err = ReadCredentials(s)
	return
}

// ReadCredentials reads credentials from a file or the system keyring.
// The path can be a file path, a "file:/path" URI or a
// "keyring:service/user" URI.
func ReadCredentials(path string) (cmap CredentialMap, err error) {
	uri, err := url.Parse(path)
	if nil != err || ("file" != uri.Scheme && "keyring" != uri.Scheme) {
		uri = &url.URL{Scheme: "file", Path: path}
		err = nil
	}

	var conf config.TypedConfig
	if "file" == uri.Scheme {
		var iconf interface{}
		iconf, err = util.ReadFunc(uri.Path, func(file *os.File) (interface{}, error) {
			return config.ReadTyped(file)
		})
		if nil == err {
			conf = iconf.(config.TypedConfig)
		}
	} else if "keyring" == uri.Scheme {
		service, user := "", ""
		parts := strings.SplitN(uri.Opaque, "/", 2)
		if 1 <= len(parts) {
			service = parts[0]
		}
		if 2 <= len(parts) {
			user = parts[1]
		}
		var pass string
		pass, err = keyring.Get(service, user)
		if nil != err {
			return nil, err
		}
		conf, err = config.ReadTyped(strings.NewReader(pass))
	}

	if nil != err {
		return nil, err
	}

	cmap = CredentialMap(conf[""])
	if nil == cmap {
		cmap = CredentialMap{}
	}
	return cmap, nil
}

// WriteCredentials writes credentials to a file or the system keyring.
// The path can be a file path, a "file:/path" URI or a
// "keyring:service/user" URI.
func WriteCredentials(path string, cmap CredentialMap) (err error) {
	uri, err := url.Parse(path)
	if nil != err || ("file" != uri.Scheme && "keyring" != uri.Scheme) {
		uri = &url.URL{Scheme: "file", Path: path}
		err = nil
	}

	conf := config.TypedConfig{}
	conf[""] = config.TypedSection(cmap)
	if "file" == uri.Scheme {
		err = util.WriteFunc(path, 0600, func(file *os.File) error {
			return config.WriteTyped(file, conf)
		})
	} else if "keyring" == uri.Scheme {
		var buf bytes.Buffer
		err = config.WriteTyped(&buf, conf)
		if nil != err {
			return
		}
		service, user := "", ""
		parts := strings.SplitN(uri.Opaque, "/", 2)
		if 1 <= len(parts) {
			service = parts[0]
		}
		if 2 <= len(parts) {
			user = parts[1]
		}
		err = keyring.Set(service, user, buf.String())
	}

	return
}

// DeleteCredentials deletes credentials to a file or the system keyring.
// The path can be a file path, a "file:/path" URI or a
// "keyring:service/user" URI.
func DeleteCredentials(path string) (err error) {
	uri, err := url.Parse(path)
	if nil != err || ("file" != uri.Scheme && "keyring" != uri.Scheme) {
		uri = &url.URL{Scheme: "file", Path: path}
		err = nil
	}

	if "file" == uri.Scheme {
		err = os.Remove(path)
	} else if "keyring" == uri.Scheme {
		service, user := "", ""
		parts := strings.SplitN(uri.Opaque, "/", 2)
		if 1 <= len(parts) {
			service = parts[0]
		}
		if 2 <= len(parts) {
			user = parts[1]
		}
		err = keyring.Delete(service, user)
	}

	return
}

// Session represents an authentication/authorization session.
type Session interface {
	Credentials() CredentialMap
}

// SessionRefresher refreshes a session.
type SessionRefresher interface {
	Refresh(force bool) error
}

// SessionDestroyer destroys a session.
type SessionDestroyer interface {
	Destroy()
}

// Auth is the primary interface implemented by an authenticator/authorizer.
type Auth interface {
	Session(credentials CredentialMap) (Session, error)
}

// Registry is the default authenticator/authorizer factory registry.
var Registry = objreg.NewObjectFactoryRegistry()
