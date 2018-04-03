/*
 * httputil.go
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

package httputil

import (
	"crypto/tls"
	"errors" // remain compatible with package http
	"net/http"
	"sync"
)

var (
	redirMap = map[*http.Request]error{}
	redirMux = sync.Mutex{}
)

func checkRedirect(req *http.Request, via []*http.Request) error {
	// remain compatible with package http
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	redirMux.Lock()
	defer redirMux.Unlock()
	return redirMap[via[0]]
}

func AllowRedirect(req *http.Request, allow bool) {
	redirMux.Lock()
	defer redirMux.Unlock()
	if allow {
		delete(redirMap, req)
	} else {
		redirMap[req] = http.ErrUseLastResponse
	}
}

func NewClient() *http.Client {
	return &http.Client{
		CheckRedirect: checkRedirect,
	}
}

func SetInsecureSkipVerify(client *http.Client) {
	if nil == client.Transport {
		transport := http.DefaultTransport.(*http.Transport)
		client.Transport = &http.Transport{
			Proxy:                 transport.Proxy,
			DialContext:           transport.DialContext,
			MaxIdleConns:          transport.MaxIdleConns,
			IdleConnTimeout:       transport.IdleConnTimeout,
			TLSHandshakeTimeout:   transport.TLSHandshakeTimeout,
			ExpectContinueTimeout: transport.ExpectContinueTimeout,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		}
	} else if transport, ok := client.Transport.(*http.Transport); ok {
		if nil == transport.TLSClientConfig {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		} else {
			transport.TLSClientConfig.InsecureSkipVerify = true
		}
	}
}

var DefaultClient = NewClient()
