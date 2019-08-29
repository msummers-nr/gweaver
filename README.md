# gweaver
A simple source code weaver for Go programs

## Method
- Put the weaves in their own directory (default is `ext`) with a fully qualified package path
- Ensure the `weaver` executable is in the `PATH`
- Run `go generate ext/*`

## 
- Pick-up an aspect definition file (`*.ag`)
- Find the module associated with the `.ag` definition
- Read the original into the AST
- Modify the AST per the `.ag` definition
- Copy the _entire_ modified module to a local 'fork'
- Write the modified AST to the fork
- add a `replace original/module => forked/module` to go.mod
  - If we have to support no vgo Go's we can 'fork' in the GOPATH
  
## To DO
- Handle `import` CRUD
- Rename original `func` on `surround`
- Allow for line-end comments as control in weave
- Write modified AST
  - Handle the whole, big, _What to do with the Package?_ problem