# gweaver
A simple source code weaver for Go programs

## Method
- Put the weaves in their own directory (default is `ext`) with a fully qualified package path
- Ensure the `weaver` executable is in the `PATH`
- Run `go build cmd/weaver/weaver.go  ;  ./weaver -weaveRoot ext`

## Weaving
1. Each woven package gets a unique directory under `weaveRoot`
2. _All_ weaves for a package go into the same directory
3. Each weave `go` file corresponds directly to the original package `go` file


## 
- Pick-up the weave definition files from the `ext` directory
  - One weave file per `package`
- Read the weave's AST
- Using the package declared in the weave get the target package's AST
- Modify the target AST per the weave
- Copy the _entire_ modified package as module to a local 'fork'
- Write the modified AST to the fork
- add a `replace original/module => forked/module` to go.mod
  - If we have to support no vgo Go's we can 'fork' in the GOPATH
  
## To Do
- Handle `import` CRUD
- ~~Rename original `func` on `surround`~~
- Allow for line-end comments as control in weave
- Write modified AST
  - Handle the whole, big, _What to do with the Package?_ problem
     1. Where to write the woven package?
     2. Create the `go.mod` file in the woven packages root
     3. Use `go mod edit` to add the `replace` to the app's `go.mod` file (?)
  
## Annotations
- Set the fully qualified name of the package to weave. This _must_ go at the head of the weave file *before* the `package` declaration!
  - `// +weaver packageFQN fully/qualified/package/name`
  
## References
- https://golang.org/pkg/go/ast/
- https://github.com/fatih/astrewrite/blob/master/astrewrite.go
- https://godoc.org/golang.org/x/tools/go/ast/astutil
- https://zupzup.org/ast-manipulation-go/
- https://zupzup.org/go-ast-traversal/
- https://godoc.org/golang.org/x/tools/go/ast/inspector
- https://arslan.io/2017/09/14/the-ultimate-guide-to-writing-a-go-tool/
- https://github.com/golang/tools/blob/master/cmd/stringer/stringer.go
