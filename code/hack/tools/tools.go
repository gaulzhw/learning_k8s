//go:build tools
// +build tools

// This package imports things required by build scripts, to force `go mod` to see them as dependencies
package tools

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
	_ "k8s.io/code-generator/cmd/informer-gen"
	_ "k8s.io/code-generator/cmd/lister-gen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
