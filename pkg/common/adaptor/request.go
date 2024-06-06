/*
 * Copyright 2022 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package adaptor

import (
	"bytes"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"io"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/protocol"
)

// GetCompatRequest only support basic function of Request, not for all.
func GetCompatRequest(req *protocol.Request) (*http.Request, error) {
	r, err := http.NewRequest(string(req.Method()), req.URI().String(), bytes.NewReader(req.Body()))
	if err != nil {
		return r, err
	}

	h := make(map[string][]string)
	req.Header.VisitAll(func(k, v []byte) {
		h[string(k)] = append(h[string(k)], string(v))
	})

	r.Header = h
	return r, nil
}

// CopyToHertzRequest copy uri, host, method, protocol, header, but share body reader from http.Request to protocol.Request.
func CopyToHertzRequest(req *http.Request, hreq *protocol.Request) error {
	engine := route.NewEngine(config.NewOptions([]config.Option{}))
	return CopyToHertzRequestUseEngineConf(engine, req, hreq)
}

func CopyToHertzRequestUseEngineConf(engine *route.Engine, req *http.Request, hreq *protocol.Request) error {
	hreq.Header.InitContentLengthWithValue(-2)
	hreq.Header.SetRequestURI(req.RequestURI)
	hreq.Header.SetHost(req.Host)
	hreq.Header.SetMethod(req.Method)
	hreq.Header.SetProtocol(req.Proto)

	for k, v := range req.Header {
		for _, vv := range v {
			hreq.Header.Add(k, vv)

			switch k {
			case consts.HeaderContentLength:
				if hreq.Header.ContentLength() != -1 {
					if contentLength, err := strconv.Atoi(vv); err != nil {
						return err
					} else {
						hreq.Header.InitContentLengthWithValue(contentLength)
						hreq.Header.SetContentLengthBytes([]byte(vv))
					}
				}
			case consts.HeaderTransferEncoding:
				if vv != consts.HeaderTrailer {
					hreq.Header.SetContentLength(-1)
				}

			}
		}
	}

	if req.Body != nil {
		if engine.IsStreamRequestBody() || hreq.Header.ContentLength() == -1 {
			hreq.SetBodyStream(req.Body, -1)
		} else if hreq.Header.ContentLength() == -2 {
			if engine.IsStreamRequestBody() {
				hreq.Header.IgnoreBody()
				hreq.Header.SetContentLength(0)
			}
		} else {
			buf, err := io.ReadAll(&io.LimitedReader{R: req.Body, N: int64(hreq.Header.ContentLength())})
			hreq.SetBody(buf)
			if err != nil && err != io.EOF {
				panic(err)
			}
		}
	}
	return nil
}
