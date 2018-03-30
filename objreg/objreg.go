/*
 * objreg.go
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
	"sync"

	"github.com/billziss-gh/golib/errors"
	"github.com/billziss-gh/objfs/errno"
)

// ObjectFactory acts as a constructor for a class of objects.
type ObjectFactory func(args ...interface{}) (interface{}, error)

// ObjectFactoryRegistry maintains a mapping of names to object factories.
type ObjectFactoryRegistry struct {
	reg map[string]ObjectFactory
	mux sync.Mutex
}

// RegisterFactory registers an object factory.
func (self *ObjectFactoryRegistry) RegisterFactory(name string, factory ObjectFactory) {
	self.mux.Lock()
	defer self.mux.Unlock()
	self.reg[name] = factory
}

// UnregisterFactory unregisters an object factory.
func (self *ObjectFactoryRegistry) UnregisterFactory(name string) {
	self.mux.Lock()
	defer self.mux.Unlock()
	delete(self.reg, name)
}

// GetFactory gets an object factory by name.
func (self *ObjectFactoryRegistry) GetFactory(name string) ObjectFactory {
	self.mux.Lock()
	defer self.mux.Unlock()
	return self.reg[name]
}

// NewObject creates an object using its registered object factory.
func (self *ObjectFactoryRegistry) NewObject(
	name string, args ...interface{}) (interface{}, error) {
	factory := self.GetFactory(name)
	if nil == factory {
		return nil, errors.New(": unknown object factory "+name, nil, errno.EINVAL)
	}
	return factory(args...)
}

// NewObjectFactoryRegistry creates a new object factory registry.
func NewObjectFactoryRegistry() *ObjectFactoryRegistry {
	return &ObjectFactoryRegistry{
		reg: map[string]ObjectFactory{},
		mux: sync.Mutex{},
	}
}
