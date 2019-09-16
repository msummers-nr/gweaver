package loom

import (
	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

// TODO Weaves must be per file- this fixes the import problem
type weave struct {
	file                    *ast.File
	inserts                 map[string]*ast.Node
	deletes                 map[string]*ast.Node
	replaces                map[string]*ast.Node
	replaceAndCallOriginals map[string]*ast.Node
	importAdds              []*ast.ImportSpec
	importDeletes           []*ast.ImportSpec
	packageName             string
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

func NewWeave(filename string) (w *weave) {
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("Weaver package: %+v\n", f.Name)
	for _, c := range f.Comments {
		log.Debugf("%+v\n", c.Text())
	}

	w = &weave{file: f, inserts: make(map[string]*ast.Node), deletes: make(map[string]*ast.Node), replaces: make(map[string]*ast.Node), replaceAndCallOriginals: make(map[string]*ast.Node)}

	// walk the tree once capturing the weave nodes
	ast.Inspect(f, func(n ast.Node) bool {
		switch t := n.(type) {

		case *ast.FuncDecl:
			log.Debugf("Inspect: type: %T value: %+v", t, t)
			spew.Dump(t)
			op := w.parseCommentGroup(t.Doc)
			if op == nop {
				break
			}
			//if op == replace {
			//	// TODO If it's a replace then we'll need to:
			//	// 1. Rename the original method/func
			//	// 2. Tweak the weave's invocation of the original to the renamed
			//	op = w.replaceOriginal(t, &n)
			//}
			w.addNode(op, t.Name.Name, &n)

			// GenDecl covers const, import, type, and var with Doc (block) comments
		case *ast.GenDecl:
			log.Debugf("Inspect: type: %T value: %+v", t, t)
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
			log.Debugf("Inspect: type: %T value: %+v", t, t)
			op := w.parseCommentGroup(t.Comment)
			if op == nop {
				break
			}
			w.addImport(op, t)
		case *ast.TypeSpec:
			log.Debugf("Inspect: type: %T value: %+v", t, t)
			op := w.parseCommentGroup(t.Comment)
			if op == nop {
				break
			}
			w.addNode(op, t.Name.Name, &n)
		case *ast.ValueSpec: // const is also a value spec
			log.Debugf("Inspect: type: %T value: %+v", t, t)
			op := w.parseCommentGroup(t.Comment)
			if op == nop {
				break
			}
			w.addNode(op, valueSpecName(t.Names), &n)
		case *ast.CommentGroup:
			log.Debugf("Inspect: type: %T value: %+v", t, t)
			w.parseCommentGroup(t)
		case *ast.Comment:
			log.Debugf("Inspect: type: %T value: %+v", t, t)
			w.parseComment(t)
		default:
			log.Debugf("Inspect: type: %T value: %+v", t, t)
		}
		return true
	})

	log.Debugf("Parsed weave:\n %+v \n", w)
	return
}

func (w *weave) replaceOriginal(decl *ast.FuncDecl, node *ast.Node) (op string) {
	log.Debugf("replaceOriginal: decl: %+v node: %+v", *decl, *node)
	op = replace
	// rename the original
	originalFnName := decl.Name.Name
	decl.Name.Name = originalFnName + originalSuffix
	op = insert
	// walk the weave node replacing the original calls
	ast.Inspect(*node, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.CallExpr:
			log.Debugf("\treplaceOriginal: Inspect: type: %T value: %+v", t, t)
		default:
			log.Debugf("\treplaceOriginal: Inspect: type: %T value: %+v", t, t)
		}
		return true
	})
	return
}

func (w *weave) processValueSpec(n *ast.Node, gd *ast.ValueSpec, op string, name string) {
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

func (w *weave) processTypeSpec(n *ast.Node, gd *ast.TypeSpec, op string, name string) {
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

func (w *weave) processImportSpec(n *ast.Node, gd *ast.ImportSpec, op string, name string) {
	switch op {
	case insert:
		w.importAdds = append(w.importAdds, gd)
	case delete:
		w.importDeletes = append(w.importDeletes, gd)
	case replace:
	default:
	}
}

func getGenDeclName(decl *ast.GenDecl) (name string) {
	switch l := len(decl.Specs); l {
	case 0:
		spew.Dump(decl)
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

func (w *weave) addNode(op string, name string, n *ast.Node) {
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
func (w *weave) addImport(op string, n *ast.ImportSpec) {
	switch op {
	case insert:
		w.importAdds = append(w.importAdds, n)
	case delete:
		w.importDeletes = append(w.importDeletes, n)
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

// TODO genericize this to deal with more than op:
//    - function rename
//    - debug comment
//    - ?
func (w *weave) parseCommentGroup(group *ast.CommentGroup) (op string) {
	//log.Debugf("parseCommentGroup:  group: %+v", group)
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

func (w *weave) parseComment(c *ast.Comment) (op string, ok bool) {
	op = nop
	ok = false
	s := strings.ToLower(strings.Trim(c.Text, " "))
	log.Debugf("parseComment: %s", s)
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
				log.Debugf("parseComment: i: %d v: %s", i, v)
			}
			log.Fatalf("parseComment: invalid packageFQN annotation: len: %d %+v text: %s", c, s)
		}
		log.Debugf("parseComment: packageName: %s", w.packageName)
		w.packageName = p[3]
		op = packageFQN
		ok = true
	}
	return
}

func (w *weave) GetPackageName() string {
	log.Debugf("GetPackageName: %s", w.packageName)
	return w.packageName
}

func (w *weave) has(n ast.Node) (r ast.Node, ok bool) {
	//nn := nodeName(n)
	//wn, ok := w.inserts[nn]
	//if ok {
	//	//r = wn.n
	//}
	//log.Tracef("has: ok: %s nn: %s", ok, nn)
	return
}

func (w *weave) getReplace(n ast.Node) (r *ast.Node, ok bool) {
	nn := nodeName(n)
	r, ok = w.replaces[nn]
	log.Tracef("getReplace: ok: %s nn: %s", ok, nn)
	return
}

func (w *weave) getReplaceAndCallOriginal(n ast.Node) (r *ast.Node, ok bool) {
	nn := nodeName(n)
	r, ok = w.replaceAndCallOriginals[nn]
	log.Tracef("getReplaceAndCallOriginal: ok: %s nn: %s", ok, nn)
	return
}

func (w *weave) getDelete(n ast.Node) (r *ast.Node, ok bool) {
	nn := nodeName(n)
	r, ok = w.deletes[nn]
	log.Tracef("getDelete: ok: %s nn: %s", ok, nn)
	return
}

func (w *weave) getInserts() map[string]*ast.Node {
	return w.inserts
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
