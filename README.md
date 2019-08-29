# gweaver
A simple source code weaver for Go programs

## Method
- Put the weaves in their own directory (default is `ext`) with a fully qualified package path
- Ensure the `weaver` executable is in the `PATH`
- Run `go generate ext/*`

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
- Rename original `func` on `surround`
- Allow for line-end comments as control in weave
- Write modified AST
  - Handle the whole, big, _What to do with the Package?_ problem
  
## References
- https://golang.org/pkg/go/ast/#ImportSpec
- https://github.com/fatih/astrewrite/blob/master/astrewrite.go
- https://godoc.org/golang.org/x/tools/go/ast/astutil#Cursor.InsertAfter
- https://zupzup.org/ast-manipulation-go/
- https://zupzup.org/go-ast-traversal/
- https://godoc.org/golang.org/x/tools/go/ast/inspector
- https://arslan.io/2017/09/14/the-ultimate-guide-to-writing-a-go-tool/
- https://github.com/golang/tools/blob/master/cmd/stringer/stringer.go
