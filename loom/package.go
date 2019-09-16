package loom

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"go/ast"
	"go/printer"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
	"runtime/debug"
	"strings"
)

type source struct {
	pkg             *packages.Package
	isFirstFunction bool
}

func NewPackage(p string) *source {
	log.Debugf("NewPackage: name: %s", p)
	//p = "./" + p
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
	if pkgs[0].Errors != nil {
		log.Fatalf("Error loading package: %s error: %+v", p, pkgs[0].Errors)
	}
	log.Debugf("NewPackage: package: %+v", pkgs[0])
	return &source{pkg: pkgs[0]}
}

func (p *source) ApplyWeave(w *weave) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("panic: recover: %+v", r)
			debug.PrintStack()
		}
	}()

	log.Debugf("ApplyWeave")

	// TODO some operations should happen on the parent, delete for instance
	// preApply & postApply go inside this method so they can capture the weaver pointer
	preApply := func(c *astutil.Cursor) (ok bool) {
		// Insert everything before the first FuncDecl
		if p.firstFunc(c.Node()) {
			for _, v := range w.getInserts() {
				log.Debugf("Inserting: %+v", *v)
				c.InsertBefore(*v)
			}
		}

		// Let's see if we replace this node
		wn, ok := w.getReplace(c.Node())
		if ok {
			log.Debugf("Replace: %+v with: %+v", c.Node(), *wn)
			c.Replace(*wn)
		}

		wn, ok = w.getReplaceAndCallOriginal(c.Node())
		if ok {
			log.Debugf("ReplaceAndCallOriginal: %+v with: %+v", c.Node(), *wn)
			renameAsOriginal(c.Node())
			c.InsertBefore(*wn)
		}

		// See if we delete this node
		wn, ok = w.getDelete(c.Node())
		if ok {
			log.Debugf("Delete: %+v", c.Node())
			c.Delete()
		}
		return true
	}

	postApply := func(c *astutil.Cursor) (ok bool) {
		return true
	}

	//log.Debugf("ApplyWeave: processing p: %+v", *p)
	//log.Debugf("ApplyWeave: processing p.pkg: %+v", *p.pkg)
	// For each file's AST in the package
	for _, f := range p.pkg.Syntax {
		//		log.Debugf("ApplyWeave: processing f: %+v", f)
		for _, i := range w.importAdds {
			if i.Name == nil {
				astutil.AddImport(p.pkg.Fset, f, pathFix(i.Path.Value))
			} else {
				astutil.AddNamedImport(p.pkg.Fset, f, i.Name.String(), pathFix(i.Path.Value))
			}
		}
		for _, i := range w.importDeletes {
			if i.Name == nil {
				astutil.DeleteImport(p.pkg.Fset, f, pathFix(i.Path.Value))
			} else {
				astutil.DeleteNamedImport(p.pkg.Fset, f, i.Name.String(), pathFix(i.Path.Value))
			}
		}
		p.isFirstFunction = true
		log.Debugf("ApplyWeave: f: %+v", f)
		rewritten := astutil.Apply(f, preApply, postApply)
		var buf bytes.Buffer
		printer.Fprint(&buf, p.pkg.Fset, rewritten)
		log.Debugf("Woven result:\n%s\n", buf.String())
		// TODO Write must happen per file
	}
}

func renameAsOriginal(node ast.Node) {
	switch t := node.(type) {
	case *ast.FuncDecl:
		t.Name.Name = t.Name.Name + "Original"
	default:
	}
}

func pathFix(s string) string {
	s = strings.ReplaceAll(s, "\"", "")
	s = strings.ReplaceAll(s, "\\", "")
	return s
}

func (p *source) firstFunc(n ast.Node) (yes bool) {
	switch n.(type) {
	case *ast.FuncDecl:
		if p.isFirstFunction {
			p.isFirstFunction = false
			return true
		}
	default:
		return false
	}

	return false
}

func (p *source) Write() {

}
