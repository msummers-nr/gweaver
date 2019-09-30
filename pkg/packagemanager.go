package pkg

import (
	"go/ast"
	"go/token"
	"os"
)

type PackageManager interface {
	setup(s *source)
	writeWovenFile(node ast.Node, fn string, fset *token.FileSet)
}

func CreateDirIfNotExist(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			panic(err)
		}
	}
}
