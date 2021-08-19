//go:build tools
// +build tools

package tools

// Manage tool dependencies via go.mod.
//
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
// https://github.com/golang/go/issues/25922
//
// nolint
import (
	_ "github.com/davidrjenni/reftools/cmd/fillstruct"
	_ "github.com/fatih/gomodifytags"
	_ "github.com/fatih/motion"
	_ "github.com/go-delve/delve/cmd/dlv"
	_ "github.com/gogo/protobuf/protoc-gen-gofast"
	_ "github.com/gogo/protobuf/protoc-gen-gogo"
	_ "github.com/gogo/protobuf/protoc-gen-gogofast"
	_ "github.com/gogo/protobuf/protoc-gen-gogofaster"
	_ "github.com/gogo/protobuf/protoc-gen-gogoslick"
	_ "github.com/golang/protobuf/protoc-gen-go"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/josharian/impl"
	_ "github.com/jstemmer/gotags"
	_ "github.com/kisielk/errcheck"
	_ "github.com/klauspost/asmfmt/cmd/asmfmt"
	_ "github.com/koron/iferr"
	_ "github.com/magefile/mage"
	_ "github.com/rogpeppe/godef"
	_ "github.com/stretchr/gorc"
	_ "github.com/twitchtv/twirp/protoc-gen-twirp"
	_ "github.com/verloop/twirpy/protoc-gen-twirpy"
	_ "golang.org/x/lint/golint"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/cmd/gorename"
	_ "golang.org/x/tools/cmd/guru"
	_ "golang.org/x/tools/gopls"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "honnef.co/go/tools/cmd/keyify"
	_ "honnef.co/go/tools/cmd/staticcheck"
)

//go:generate go install -v "github.com/davidrjenni/reftools/cmd/fillstruct"
//go:generate go install -v "github.com/fatih/gomodifytags"
//go:generate go install -v "github.com/fatih/motion"
//go:generate go install -v "github.com/go-delve/delve/cmd/dlv"
//go:generate go install -v "github.com/josharian/impl"
//go:generate go install -v "github.com/jstemmer/gotags"
//go:generate go install -v "github.com/kisielk/errcheck"
//go:generate go install -v "github.com/klauspost/asmfmt/cmd/asmfmt"
//go:generate go install -v "github.com/koron/iferr"
//go:generate go install -v "github.com/magefile/mage"
//go:generate go install -v "github.com/rogpeppe/godef"
//go:generate go install -v "github.com/stretchr/gorc"
//go:generate go install -v "golang.org/x/lint/golint"
//go:generate go install -v "golang.org/x/tools/cmd/goimports"
//go:generate go install -v "golang.org/x/tools/cmd/gorename"
//go:generate go install -v "golang.org/x/tools/cmd/guru"
//go:generate go install -v "golang.org/x/tools/gopls"
//go:generate go install -v "honnef.co/go/tools/cmd/keyify"
//go:generate go install -v "honnef.co/go/tools/cmd/staticcheck"
//go:generate go install -v "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
//go:generate go install -v "github.com/gogo/protobuf/protoc-gen-gofast"
//go:generate go install -v "github.com/gogo/protobuf/protoc-gen-gogo"
//go:generate go install -v "github.com/gogo/protobuf/protoc-gen-gogofast"
//go:generate go install -v "github.com/gogo/protobuf/protoc-gen-gogofaster"
//go:generate go install -v "github.com/gogo/protobuf/protoc-gen-gogoslick"
//go:generate go install -v "github.com/golang/protobuf/protoc-gen-go"
//go:generate go install -v "github.com/twitchtv/twirp/protoc-gen-twirp"
//go:generate go install -v "github.com/verloop/twirpy/protoc-gen-twirpy"
//go:generate go install -v "google.golang.org/protobuf/cmd/protoc-gen-go"
