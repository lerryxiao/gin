// Copyright 2017 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ugorji/go/codec"
)

type msgpackBinding struct{}

func (msgpackBinding) Name() string {
	return "msgpack"
}

func (b msgpackBinding) Bind(req *http.Request, dt []byte, obj interface{}) ([]byte, error) {
	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	if buf == nil || len(buf) <= 0 {
		buf = dt
	}
	return buf, b.BindBody(buf, obj)
}

func (msgpackBinding) BindBody(body []byte, obj interface{}) error {
	return decodeMsgPack(bytes.NewReader(body), obj)
}

func decodeMsgPack(r io.Reader, obj interface{}) error {
	cdc := new(codec.MsgpackHandle)
	if err := codec.NewDecoder(r, cdc).Decode(&obj); err != nil {
		return err
	}
	return validate(obj)
}
