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
	"errors" // remain compatible with package http
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/billziss-gh/golib/retry"
)

var DefaultTransport = NewTransport()
var DefaultClient = NewClient(DefaultTransport)
var DefaultRetryCount = 5

func NewTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport)
	return &http.Transport{
		Proxy:                 transport.Proxy,
		DialContext:           transport.DialContext,
		MaxIdleConns:          transport.MaxIdleConns,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       transport.IdleConnTimeout,
		TLSHandshakeTimeout:   transport.TLSHandshakeTimeout,
		ExpectContinueTimeout: transport.ExpectContinueTimeout,
	}
}

func NewClient(transport *http.Transport) *http.Client {
	return &http.Client{
		CheckRedirect: CheckRedirect,
		Transport:     transport,
	}
}

func Retry(body io.Seeker, do func() (*http.Response, error)) (rsp *http.Response, err error) {
	retry.Retry(
		retry.Count(DefaultRetryCount),
		retry.Backoff(time.Second, time.Second*30),
		func(i int) bool {

			// rewind body if there is one
			if nil != body {
				_, err := body.Seek(0, 0)
				if nil != err {
					return false
				}
			}

			rsp, err = do()

			if nil != err {
				// retry on connection errors without body
				if nil == body {
					return true
				}

				// retry on Dial and DNS errors
				for e := err; nil != e; {
					switch t := e.(type) {
					case *url.Error:
						e = t.Err
					case *net.OpError:
						e = t.Err
						if "dial" == t.Op {
							return true
						}
					case *net.DNSError:
						e = nil
						if t.Temporary() {
							return true
						}
					}
				}

				return false
			}

			// retry on HTTP 429, 503, 509
			if 429 == rsp.StatusCode || 503 == rsp.StatusCode || 509 == rsp.StatusCode {
				rsp.Body.Close()
				return true
			}

			return false
		})

	return
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

func CheckRedirect(req *http.Request, via []*http.Request) error {
	// remain compatible with package http
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	redirMux.Lock()
	defer redirMux.Unlock()
	return redirMap[via[0]]
}

var redirMap = map[*http.Request]error{}
var redirMux = sync.Mutex{}
