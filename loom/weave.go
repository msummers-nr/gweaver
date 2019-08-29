package loom

import (
	log "github.com/sirupsen/logrus"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type weave struct {
	file   *ast.File
	weaves map[string]*weaveNode
}

const (
	create       string = "create"
	surround     string = "surround"
	replace      string = "replace"
	nop          string = "nop"
	weaverSuffix string = "+weaving"
)

type weaveNode struct {
	n  ast.Node
	op string
}

func NewWeave(filename string) *weave {
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("Weaver package: %+v\n", f.Name)
	for _, c := range f.Comments {
		log.Debugf("%+v\n", c.Text())
	}
	wn := make(map[string]*weaveNode)
	// walk the tree once capturing the weave nodes
	ast.Inspect(f, func(n ast.Node) bool {
		log.Debugf("node: %+v\n", n)
		switch t := n.(type) {
		case *ast.FuncDecl:
			op := operation(t.Doc)
			// TODO error if op == nop
			if op == nop {
				log.Fatalf("")
			}
			wn[t.Name.Name] = &weaveNode{op: op, n: n}
		case *ast.TypeSpec:
			op := operation(t.Doc)
			// TODO error if op == nop
			wn[t.Name.Name] = &weaveNode{op: op, n: n}
		case *ast.ValueSpec:
			op := operation(t.Doc)
			// TODO warn if op == nop . This is ok if it's a ValueSpec _inside_ a TypeSpec
			wn[name(t.Names)] = &weaveNode{op: op, n: n}
		case *ast.ImportSpec:
			// TODO
		}
		return true
	})
	log.Debugf("Parsed weave:\n %+v \n", wn)
	return &weave{file: f, weaves: wn}
}

func name(i []*ast.Ident) string {
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

func operation(group *ast.CommentGroup) (op string) {
	log.Debugf("operation:  group: %+v", group)
	if group == nil {
		return nop
	}
	op = nop
	for _, c := range group.List {
		s := strings.ToLower(strings.Trim(c.Text, " "))
		if !strings.HasSuffix(s, weaverSuffix) {
			continue
		}
		if strings.HasSuffix(s, create) {
			op = create
		} else if strings.HasSuffix(s, surround) {
			op = surround
		} else if strings.HasSuffix(s, replace) {
			op = replace
		}
	}
	if op == nop {
		log.Warnf("No operation found: %+v", group.List)
	}
	return op
}

func (w *weave) GetPackageName() string {
	return w.file.Name.Name
}

func (w *weave) has(n ast.Node) (r ast.Node, ok bool) {
	// TODO
	return
}
