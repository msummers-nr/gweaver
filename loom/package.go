package loom

import (
	"bytes"
	"github.com/fatih/astrewrite"
	log "github.com/sirupsen/logrus"
	"go/ast"
	"go/printer"
	"golang.org/x/tools/go/packages"
)

type source struct {
	pkg *packages.Package
}

func NewPackage(p string) *source {
	cfg := &packages.Config{
		//Mode:  packages.NeedName | packages.NeedSyntax | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedTypesSizes,
		Mode:  packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedTypesSizes,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, p)
	if err != nil {
		log.Fatal(err)
	}
	if len(pkgs) != 1 {
		log.Fatalf("error: %d packages found", len(pkgs))
	}
	log.Debugf("NewPackage: package: %+v", pkgs[0])
	return &source{pkg: pkgs[0]}
}

func (p *source) ApplyWeave(w *weave) {
	// func goes here to capture the weave pointer w
	rewriteFunc := func(n ast.Node) (ast.Node, bool) {
		// Is this node in the weave?
		if r, ok := w.has(n); ok {
			// If yes, then return the weave node
			return r, true
		}
		return n, true
	}

	for i, f := range p.pkg.Syntax {
		if i == 0 {
			// TODO add anything new the weave provides to the first ast
		}
		rewritten := astrewrite.Walk(f, rewriteFunc)
		var buf bytes.Buffer
		printer.Fprint(&buf, p.pkg.Fset, rewritten)
		log.Debugf("Woven result:\n%s\n", buf.String())
	}
}

func (p *source) Write() {

}
