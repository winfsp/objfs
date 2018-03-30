/*
 * link.go
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

package cache

import "unsafe"

type link_t struct {
	prev, next *link_t
}

func (link *link_t) Init() {
	link.prev = link
	link.next = link
}

func (link *link_t) InsertTail(list *link_t) {
	prev := list.prev
	link.next = list
	link.prev = prev
	prev.next = link
	list.prev = link
}

func (link *link_t) Remove() {
	next := link.next
	prev := link.prev
	prev.next = next
	next.prev = prev
}

func containerOf(p unsafe.Pointer, o uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) - o)
}
