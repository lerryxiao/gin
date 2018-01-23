// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gin

import (
	"io"
	"os"

	"github.com/lerryxiao/gin/binding"
)

// EnvGinMode gin环境变量
const EnvGinMode = "GIN_MODE"

const (
	// DebugMode 调试模式
	DebugMode string = "debug"
	// ReleaseMode 费调试模式
	ReleaseMode string = "release"
	// TestMode 测试模式
	TestMode string = "test"
)
const (
	debugCode = iota
	releaseCode
	testCode
)

// DefaultWriter is the default io.Writer used the Gin for debug output and
// middleware output like Logger() or Recovery().
// Note that both Logger and Recovery provides custom ways to configure their
// output io.Writer.
// To support coloring in Windows use:
// 		import "github.com/mattn/go-colorable"
// 		gin.DefaultWriter = colorable.NewColorableStdout()
var DefaultWriter io.Writer = os.Stdout

// DefaultErrorWriter 默认错误输出
var DefaultErrorWriter io.Writer = os.Stderr

var ginMode = debugCode
var modeName = DebugMode

func init() {
	initDebug()
	mode := os.Getenv(EnvGinMode)
	if len(mode) == 0 {
		SetMode(DebugMode)
	} else {
		SetMode(mode)
	}
	initRouter()
	initProfile()
}

// SetMode 设置运行模式
func SetMode(value string) {
	switch value {
	case DebugMode:
		ginMode = debugCode
	case ReleaseMode:
		ginMode = releaseCode
	case TestMode:
		ginMode = testCode
	default:
		panic("gin mode unknown: " + value)
	}
	modeName = value
}

// DisableBindValidation 无法绑定
func DisableBindValidation() {
	binding.Validator = nil
}

// EnableJSONDecoderUseNumber json解码使用数字
func EnableJSONDecoderUseNumber() {
	binding.EnableDecoderUseNumber = true
}

// Mode 模式
func Mode() string {
	return modeName
}
