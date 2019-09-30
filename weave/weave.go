package weave

import (
	log "github.com/sirupsen/logrus"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
)

type Pkg struct {
	weaves map[string]*Weave
}

type Weave struct {
	file                    *ast.File
	inserts                 map[string]*ast.Node
	deletes                 map[string]*ast.Node
	replaces                map[string]*ast.Node
	replaceAndCallOriginals map[string]*ast.Node
	ImportAdds              []*ast.ImportSpec
	ImportDeletes           []*ast.ImportSpec
}

// Warning: I always compare to lowerCase so ensure the constants are lower case
const (
	insert                 string = "insert"
	delete                 string = "delete"
	replace                string = "replace"
	replaceAndCallOriginal string = "replaceandcalloriginal"
	nop                    string = "nop"
	weaverSuffix           string = "+weaver"
	packageFQN             string = "packagefqn"
	separator              string = " "
	originalSuffix         string = "Original"
)

func New(files []string) (w *Pkg) {
	w = &Pkg{weaves: make(map[string]*Weave)}
	for _, file := range files {
		w.weaves[filepath.Base(file)] = new(file)
	}
	return
}

func new(filename string) (w *Weave) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	log.Tracef("Weaver pkg: %+v\n", f.Name)
	for _, c := range f.Comments {
		log.Tracef("%+v\n", c.Text())
	}

	w = &Weave{file: f, inserts: make(map[string]*ast.Node), deletes: make(map[string]*ast.Node), replaces: make(map[string]*ast.Node), replaceAndCallOriginals: make(map[string]*ast.Node)}

	// walk the tree once capturing the Weave nodes
	ast.Inspect(f, func(n ast.Node) bool {
		switch t := n.(type) {

		case *ast.FuncDecl:
			log.Tracef("Inspect: type: %T value: %+v", t, t)
			//spew.Dump(t)
			op := w.parseCommentGroup(t.Doc)
			if op == nop {
				break
			}
			if op == replace {
				op = w.replaceOriginal(t, &n)
			}
			w.addNode(op, t.Name.Name, &n)

			// GenDecl covers const, import, type, and var with Doc (block) comments
		case *ast.GenDecl:
			log.Tracef("Inspect: type: %T value: %+v", t, t)
			op := w.parseCommentGroup(t.Doc)
			if op == nop {
				break
			}
			name := getGenDeclName(t)
			switch d := t.Specs[0].(type) {
			case *ast.ImportSpec:
				w.processImportSpec(&n, d, op, name)
			case *ast.TypeSpec:
				w.processTypeSpec(&n, d, op, name)
			case *ast.ValueSpec: // const is also a value spec
				w.processValueSpec(&n, d, op, name)
			default:
			}

			// These 3 cases cover const, import, type, and  var with line comments
		case *ast.ImportSpec:
			log.Tracef("Inspect: type: %T value: %+v", t, t)
			op := w.parseCommentGroup(t.Comment)
			if op == nop {
				break
			}
			w.addImport(op, t)
		case *ast.TypeSpec:
			log.Tracef("Inspect: type: %T value: %+v", t, t)
			op := w.parseCommentGroup(t.Comment)
			if op == nop {
				break
			}
			w.addNode(op, t.Name.Name, &n)
		case *ast.ValueSpec: // const is also a value spec
			log.Tracef("Inspect: type: %T value: %+v", t, t)
			op := w.parseCommentGroup(t.Comment)
			if op == nop {
				break
			}
			w.addNode(op, valueSpecName(t.Names), &n)
		case *ast.CommentGroup:
			log.Tracef("Inspect: type: %T value: %+v", t, t)
			w.parseCommentGroup(t)
		case *ast.Comment:
			log.Tracef("Inspect: type: %T value: %+v", t, t)
			w.parseComment(t)
		default:
			log.Tracef("Inspect: type: %T value: %+v", t, t)
		}
		return true
	})

	log.Tracef("Parsed Weave:\n %+v \n", w)
	return
}

func (w *Weave) replaceOriginal(decl *ast.FuncDecl, node *ast.Node) (op string) {
	log.Tracef("replaceOriginal: decl: %+v node: %+v", *decl, *node)
	op = replace
	// rename the original
	originalFnName := decl.Name.Name
	decl.Name.Name = originalFnName + originalSuffix
	op = insert
	// walk the Weave node replacing the original calls
	ast.Inspect(*node, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.CallExpr:
			log.Tracef("\treplaceOriginal: Inspect: type: %T value: %+v", t, t)
		default:
			log.Tracef("\treplaceOriginal: Inspect: type: %T value: %+v", t, t)
		}
		return true
	})
	return
}

func (w *Weave) processValueSpec(n *ast.Node, gd *ast.ValueSpec, op string, name string) {
	switch op {
	case insert:
		w.inserts[name] = n
	case delete:
		w.deletes[name] = n
	case replace:
		gn := ast.Node(gd)
		w.replaces[name] = &gn
	default:
	}
}

func (w *Weave) processTypeSpec(n *ast.Node, gd *ast.TypeSpec, op string, name string) {
	switch op {
	case insert:
		w.inserts[name] = n
	case delete:
		w.deletes[name] = n
	case replace:
		gn := ast.Node(gd)
		w.replaces[name] = &gn
	default:
	}
}

func (w *Weave) processImportSpec(n *ast.Node, gd *ast.ImportSpec, op string, name string) {
	switch op {
	case insert:
		w.ImportAdds = append(w.ImportAdds, gd)
	case delete:
		w.ImportDeletes = append(w.ImportDeletes, gd)
	case replace:
	default:
	}
}

func getGenDeclName(decl *ast.GenDecl) (name string) {
	switch l := len(decl.Specs); l {
	case 0:
		//spew.Dump(decl)
		log.Fatalf("GenDecl has no Specs")
	case 1:
		switch t := decl.Specs[0].(type) {
		case *ast.TypeSpec:
			return t.Name.Name
		case *ast.ValueSpec:
			return valueSpecName(t.Names)
		}
	default:
		log.Warnf("GenDecl has multiple Specs")
	}
	return
}

func (w *Weave) addNode(op string, name string, n *ast.Node) {
	switch op {
	case insert:
		w.inserts[name] = n
	case delete:
		w.deletes[name] = n
	case replace:
		w.replaces[name] = n
	case replaceAndCallOriginal:
		w.replaceAndCallOriginals[name] = n
	default:
	}
}
func (w *Weave) addImport(op string, n *ast.ImportSpec) {
	switch op {
	case insert:
		w.ImportAdds = append(w.ImportAdds, n)
	case delete:
		w.ImportDeletes = append(w.ImportDeletes, n)
	default:
	}
}

func valueSpecName(i []*ast.Ident) string {
	if len(i) == 1 {
		return i[0].Name
	}
	if len(i) <= 0 {
		log.Warnf("ValueSpec has no Ident")
	} else {
		log.Warnf("ValueSpace has multiple Idents: %+v", i)
	}
	return "Unknown"
}

// TODO genericize this to deal with more than op: function rename, debug comment, ?
func (w *Weave) parseCommentGroup(group *ast.CommentGroup) (op string) {
	//log.Tracef("parseCommentGroup:  group: %+v", group)
	op = nop
	if group == nil {
		return
	}
	for _, c := range group.List {
		op, ok := w.parseComment(c)
		if ok {
			return op
		}
	}
	return op
}

func (w *Weave) parseComment(c *ast.Comment) (op string, ok bool) {
	op = nop
	ok = false
	s := strings.ToLower(strings.Trim(c.Text, " "))
	log.Tracef("parseComment: %s", s)
	if !strings.Contains(s, weaverSuffix) {
		return
	}
	if strings.HasSuffix(s, insert) {
		op = insert
		ok = true
	} else if strings.HasSuffix(s, delete) {
		op = delete
		ok = true
	} else if strings.HasSuffix(s, replace) {
		op = replace
		ok = true
	} else if strings.HasSuffix(s, replaceAndCallOriginal) {
		op = replaceAndCallOriginal
		ok = true
	} else if strings.Contains(s, packageFQN) {
		space := regexp.MustCompile(`\s+`)
		s := space.ReplaceAllString(s, " ")
		p := strings.Split(s, separator)
		if len(p) != 4 {
			for i, v := range p {
				log.Tracef("parseComment: i: %d v: %s", i, v)
			}
			log.Fatalf("parseComment: invalid packageFQN annotation: len: %d %+v text: %s", c, s)
		}
		op = packageFQN
		ok = true
	}
	return
}

func (w *Pkg) has(n ast.Node) (r ast.Node, ok bool) {
	//nn := nodeName(n)
	//wn, ok := w.inserts[nn]
	//if ok {
	//	//r = wn.n
	//}
	//log.Tracef("has: ok: %s nn: %s", ok, nn)
	return
}

func (w *Weave) GetReplace(n ast.Node) (r *ast.Node, ok bool) {
	nn := nodeName(n)
	r, ok = w.replaces[nn]
	log.Tracef("getReplace: ok: %s nn: %s", ok, nn)
	return
}

func (w *Weave) GetReplaceAndCallOriginal(n ast.Node) (r *ast.Node, ok bool) {
	nn := nodeName(n)
	r, ok = w.replaceAndCallOriginals[nn]
	log.Tracef("getReplaceAndCallOriginal: ok: %s nn: %s", ok, nn)
	return
}

func (w *Weave) GetDelete(n ast.Node) (r *ast.Node, ok bool) {
	nn := nodeName(n)
	r, ok = w.deletes[nn]
	log.Tracef("getDelete: ok: %s nn: %s", ok, nn)
	return
}

func (w *Weave) GetInserts() map[string]*ast.Node {
	return w.inserts
}

func (w *Pkg) GetWeaveForFile(file string) (ww *Weave) {
	if !strings.HasSuffix(file, ".go") {
		file = strings.TrimSpace(file) + ".go"
	}
	ww = w.weaves[file]
	log.Debugf("GetWeaveForFile: file: %s weave: %+v weaves: %+v", file, ww, w.weaves)
	if log.IsLevelEnabled(log.DebugLevel) {
		//spew.Dump(w.weaves)
	}
	return
}

func nodeName(n ast.Node) (name string) {
	switch t := n.(type) {
	case *ast.FuncDecl:
		return t.Name.Name
	//case *ast.GenDecl:
	//	if len(t.Specs) < 1 {
	//		return ""
	//	}
	//	return nodeName(t.Specs[0])
	case *ast.TypeSpec:
		return t.Name.Name
	case *ast.ValueSpec:
		return valueSpecName(t.Names)
	case *ast.ImportSpec:
		return t.Path.Value
	default:
		return ""
	}
}
