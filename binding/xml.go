// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type xmlBinding struct{}

func (xmlBinding) Name() string {
	return "xml"
}

func (b xmlBinding) Bind(req *http.Request, dt []byte, obj interface{}) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("invalid request")
	}
	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	if buf == nil || len(buf) <= 0 {
		buf = dt
	}
	return buf, b.BindBody(buf, obj)
}

func (xmlBinding) BindBody(body []byte, obj interface{}) error {
	return decodeXML(bytes.NewReader(body), obj)
}

func decodeXML(r io.Reader, obj interface{}) error {
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}
