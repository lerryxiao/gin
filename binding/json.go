// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/lerryxiao/gin/internal/json"
)

// EnableDecoderUseNumber is used to call the UseNumber method on the JSON
// Decoder instance. UseNumber causes the Decoder to unmarshal a number into an
// interface{} as a Number instead of as a float64.
var EnableDecoderUseNumber = false

type jsonBinding struct{}

func (jsonBinding) Name() string {
	return "json"
}

func (b jsonBinding) Bind(req *http.Request, dt []byte, obj interface{}) ([]byte, error) {
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

func (jsonBinding) BindBody(body []byte, obj interface{}) error {
	return decodeJSON(bytes.NewReader(body), obj)
}

func decodeJSON(r io.Reader, obj interface{}) error {
	decoder := json.NewDecoder(r)
	if EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}
