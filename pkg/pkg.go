package pkg

import (
	log "github.com/sirupsen/logrus"
	"go/ast"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
	"gweaver/weave"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
)

type source struct {
	pkg             *packages.Package
	isFirstFunction bool
	mgr             PackageManager
}

func NewPackage(p string, mgr PackageManager) (s *source) {
	log.Tracef("NewPackage: name: %s", p)
	//p = "./" + p
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedDeps,
		Tests: false,
	}
	//pkgs, err := packages.Load(cfg, p+"...")
	pkgs, err := packages.Load(cfg, p)
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range pkgs {
		if p.Errors != nil {
			log.Errorf("Error loading pkg: %s error: %+v", p.Name, p.Errors)
		}
	}

	if len(pkgs) != 1 {
		log.Fatalf("error: %d packages found. Name: %s", len(pkgs), p)
	}
	log.Tracef("NewPackage: pkg: %+v", pkgs[0])

	s = &source{pkg: pkgs[0], mgr: mgr}
	mgr.setup(s)
	log.Debugf("NewPackage: name: %s path: %s", s.pkg.Name, s.pkg.PkgPath)
	return s
}

func (p *source) applyWeave(w *weave.Weave, f *ast.File) ast.Node {
	// TODO some operations should happen on the parent, delete for instance
	// preApply & postApply go inside this method so they can capture the weave pointer

	p.isFirstFunction = true
	preApply := func(c *astutil.Cursor) (ok bool) {
		// Insert everything before the first FuncDecl
		if p.firstFunc(c.Node()) {
			for _, v := range w.GetInserts() {
				log.Tracef("Inserting: %+v", *v)
				c.InsertBefore(*v)
			}
		}

		// Let's see if we replace this node
		wn, ok := w.GetReplace(c.Node())
		if ok {
			log.Tracef("Replace: %+v with: %+v", c.Node(), *wn)
			c.Replace(*wn)
		}

		wn, ok = w.GetReplaceAndCallOriginal(c.Node())
		if ok {
			log.Tracef("ReplaceAndCallOriginal: %+v with: %+v", c.Node(), *wn)
			renameAsOriginal(c.Node())
			c.InsertBefore(*wn)
		}

		// See if we delete this node
		wn, ok = w.GetDelete(c.Node())
		if ok {
			log.Tracef("Delete: %+v", c.Node())
			c.Delete()
		}
		return true
	}

	postApply := func(c *astutil.Cursor) (ok bool) {
		return true
	}
	return astutil.Apply(f, preApply, postApply)
}

func (p *source) ApplyWeave(wp *weave.Pkg) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("panic: recover: %+v", r)
			debug.PrintStack()
		}
	}()

	log.Tracef("ApplyWeave")
	log.Tracef("ApplyWeave: processing p: %+v", *p)
	log.Tracef("ApplyWeave: processing p.pkg: %+v", *p.pkg)

	// For each file's AST in the pkg
	for fi, f := range p.pkg.Syntax {
		log.Tracef("ApplyWeave: processing f: %+v", f)
		// f is *ast.File but f.Name is _really_ the package name! :-(
		w := wp.GetWeaveForFile(filepath.Base(p.pkg.CompiledGoFiles[fi]))
		// If the weave is nil there is no weave for this file/ast, write as-is
		if w == nil {
			p.mgr.writeWovenFile(f, p.pkg.CompiledGoFiles[fi], p.pkg.Fset)
			continue
		}

		for _, i := range w.ImportAdds {
			if i.Name == nil {
				astutil.AddImport(p.pkg.Fset, f, pathFix(i.Path.Value))
			} else {
				astutil.AddNamedImport(p.pkg.Fset, f, i.Name.String(), pathFix(i.Path.Value))
			}
		}
		for _, i := range w.ImportDeletes {
			if i.Name == nil {
				astutil.DeleteImport(p.pkg.Fset, f, pathFix(i.Path.Value))
			} else {
				astutil.DeleteNamedImport(p.pkg.Fset, f, i.Name.String(), pathFix(i.Path.Value))
			}
		}
		log.Tracef("ApplyWeave: f: %+v", f)
		rewritten := p.applyWeave(w, f)
		p.mgr.writeWovenFile(rewritten, p.pkg.CompiledGoFiles[fi], p.pkg.Fset)
	}
}

// Rename the original func so we can take its place
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

func (p *source) importPath(file *ast.File) {
	log.Debugf("importPath: path: %s file name: %s", p.pkg.PkgPath, file.Name.Name)
}

func (p *source) copyDir() {
	src := filepath.Dir(p.pkg.CompiledGoFiles[0])
	var err error
	dst := src + ".original"
	if runtime.GOOS == "windows" {
		cmd := exec.Command("Xcopy", "/E /I ", src, dst)
		log.Printf("copyDir: cmd: %s", cmd.String())
		err = cmd.Run()

	} else {
		cmd := exec.Command("cp", "-a", src, dst)
		log.Printf("copyDir: cmd: %s", cmd.String())
		err = cmd.Run()
	}
	if err != nil {
		log.Errorf("copyDir: error: %+v", err)
	}
}
