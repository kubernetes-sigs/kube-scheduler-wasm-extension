// Read about tools here: https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

//go:build tools

package tools

import (
	_ "github.com/knqyf263/go-plugin/cmd/protoc-gen-go-plugin"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
