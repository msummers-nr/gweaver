# gweaver
A simple source code weaver for Go programs

## Method
- Put the weaves in their own directory (default is `ext`) with a fully qualified package path
- Ensure the `weaver` executable is in the `PATH`
- Run `go build cmd/weaver/weaver.go  ;  ./weaver -weaveDir ext -writeDir /tmp/`

## Weaving
1. Each woven package gets a unique directory under `weaveDir`
2. _All_ weaves for a package go into the same directory
3. Each weave `go` file corresponds directly to the original package `go` file


## 
- Pick-up the weave definition files from the `ext` directory
- Read the weave's AST
- Using the package/module from the weave's path get the target package's AST
- Modify the target AST per the weave
- Copy the _entire_ modified module to a local 'fork'
- Write the modified AST to the fork
- add a `replace original/module => forked/module` to go.mod
  - If we have to support non vgo Go's we can 'fork' in the GOPATH
  
## To Do
- Comprehensive test suite
- Documentation
- Refactor the package init code, make the steps less implicit
  
## Annotations
- `// +weaver delete`
- `// +weaver insert`
- `// +weaver update`
- `// +weaver updateAndCallOriginal`
  
## References
- https://golang.org/pkg/go/ast/
- https://github.com/fatih/astrewrite/blob/master/astrewrite.go
- https://godoc.org/golang.org/x/tools/go/ast/astutil
- https://zupzup.org/ast-manipulation-go/
- https://zupzup.org/go-ast-traversal/
- https://godoc.org/golang.org/x/tools/go/ast/inspector
- https://arslan.io/2017/09/14/the-ultimate-guide-to-writing-a-go-tool/
- https://github.com/golang/tools/blob/master/cmd/stringer/stringer.go
