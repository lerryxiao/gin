// Copyright 2018 Gin Core Team.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"gopkg.in/yaml.v2"
)

type yamlBinding struct{}

func (yamlBinding) Name() string {
	return "yaml"
}

func (b yamlBinding) Bind(req *http.Request, dt []byte, obj interface{}) ([]byte, error) {
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

func (yamlBinding) BindBody(body []byte, obj interface{}) error {
	return decodeYAML(bytes.NewReader(body), obj)
}

func decodeYAML(r io.Reader, obj interface{}) error {
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}
