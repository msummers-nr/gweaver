/*
TODO How to deal with versions and version ranges
*/
package pkg

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"go/ast"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ModManager struct {
	tag              string
	writeRoot        string
	modules          map[string]string
	modulePath       string
	moduleVersion    string
	fullPackagePath  string
	fsPrefix         string
	fsFullWritePath  string
	fsPrefixOriginal string
}

func (m *ModManager) init() {
	m.modules = make(map[string]string)

	cmd := exec.Command("go", "list", "-m", "all")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	outStr := string(stdout.Bytes())
	for _, s := range strings.Split(outStr, "\n") {
		log.Tracef("modmanager.init: line: %s", s)
		// Skip blank lines
		if strings.TrimSpace(s) == "" {
			continue
		}
		mm := strings.Split(s, " ")
		switch len(mm) {
		case 1:
			m.modules[strings.TrimSpace(mm[0])] = ""
		case 2:
			m.modules[strings.TrimSpace(mm[0])] = strings.TrimSpace(mm[1])
		default:
			log.Warnf("modmanager.init: unexpected go list format: %s", s)
		}
	}
}

// Use New to accept config params
func NewModManager(writeRoot string, tag string) (m *ModManager) {
	if !strings.HasSuffix(tag, "-") {
		tag = "-" + tag
	}
	m = &ModManager{tag: tag, writeRoot: writeRoot}
	m.init()
	return m
}

// setup deals with things we can only know via source
// As other package managers are implemented refactor this to make the common steps explicit
func (m *ModManager) setup(s *source) {
	m.parsePath(s.pkg.CompiledGoFiles)
	log.Debugf("modmanager.set: fsPrefix: %s module: %s moduleVersion: %s package: %s", m.fsPrefix, m.modulePath, m.moduleVersion, m.fullPackagePath)
	m.fsPrefixOriginal = m.fsPrefix
	if m.writeRoot != "" {
		m.fsPrefix = m.writeRoot
	}
	m.fsFullWritePath = filepath.Clean(m.fsPrefix + m.modulePath + "@" + m.moduleVersion + m.tag + m.fullPackagePath)

	// Create the result directory
	err := os.MkdirAll(m.fsFullWritePath, os.ModePerm)
	if err != nil {
		log.Fatalf("modmanager.setup: error creating weave write directory: %s err: %+v", m.fsFullWritePath, err)
	}

	src := filepath.Clean(m.fsPrefixOriginal + string(filepath.Separator) + m.modulePath + "@" + m.moduleVersion)
	dst := filepath.Clean(m.fsPrefix + string(filepath.Separator) + m.modulePath + "@" + m.moduleVersion + m.tag)
	copyDir(src, dst)
	fixPermissions(dst)

	// Ensure the resulting module has a go.mod file
	m.copyOrCreateGoMod()

	// Add the replace to the local go.mod  file
	m.updateLocalGoMod()
	return
}

func (m *ModManager) parsePath(cgf []string) {
	if len(cgf) <= 0 {
		log.Fatalf("parsePath: compiledGoFiles[] is empty- unable to find woven source file directory.")
	}
	fqfp := filepath.Dir(cgf[0])
	log.Tracef("modmanager.parseFQFP: fqfp: %s", fqfp)
	for m.modulePath, m.moduleVersion = range m.modules {
		if m.modulePath == "" {
			continue
		}
		log.Tracef("modmanager.parseFQFP: modulePath: %s moduleVersion: %s", m.modulePath, m.moduleVersion)
		s := strings.Split(fqfp, m.modulePath)
		switch len(s) {
		case 0, 1:
			continue
		case 2:
			m.fsPrefix = s[0]
			m.fullPackagePath = s[1]
			if m.moduleVersion != "" {
				m.fullPackagePath = strings.TrimPrefix(m.fullPackagePath, "@"+m.moduleVersion)
				if !m.moduleVersionOk(m.modulePath, m.moduleVersion) {
				}
			}
			return
		default:
			log.Warnf("modmanager.parseFQFP: unexpected filename format: %s: len(s): %d", fqfp, len(s))
		}
	}
	return
}

func (m *ModManager) writeWovenFile(node ast.Node, fn string, fset *token.FileSet) {
	var buf bytes.Buffer
	fn = filepath.Base(fn)
	fqn := filepath.Clean(m.fsPrefix + m.modulePath + "@" + m.moduleVersion + m.tag + m.fullPackagePath + string(filepath.Separator) + fn)
	log.Debugf("Writing file: %s", fqn)
	printer.Fprint(&buf, fset, node)

	f, err := os.OpenFile(fqn, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorf("modmanager.writeWovenFile: error opening file: %s err: %+v", fqn, err)
		return
	}

	defer f.Close()
	if _, err := f.Write(buf.Bytes()); err != nil {
		log.Errorf("modmanager.writeWovenFile: error writing file: %s err: %+v", fqn, err)
	}
}

func (m *ModManager) copyOrCreateGoMod() {
	// Copy the original go.mod if it exists
	src := filepath.Clean(m.fsPrefixOriginal + m.modulePath + "@" + m.moduleVersion + "/go.mod")
	content, err := ioutil.ReadFile(src)
	if err != nil {
		// We'll assume file not found
		content = []byte("module " + m.modulePath)
	}

	dst := filepath.Clean(m.fsPrefix + m.modulePath + "@" + m.moduleVersion + m.tag + "/go.mod")
	err = ioutil.WriteFile(dst, content, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func (m *ModManager) updateLocalGoMod() {
	//replace github.com/davecgh/go-spew => /Users/mike/go/pkg/mod/github.com/davecgh/go-spew@v1.1.1-woven
	f, err := os.OpenFile("go.mod", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorf("modmanager.updateLocalGoMod: unable to open local go.mod file: %+v", err)
		return
	}

	defer f.Close()
	if _, err := f.WriteString("\nreplace " + m.modulePath + " => " + m.fsPrefix + m.modulePath + "@" + m.moduleVersion + m.tag + "\n"); err != nil {
		log.Println(err)
	}
}

// TODO implement module version check
func (m *ModManager) moduleVersionOk(module string, version string) (ok bool) {
	return true
}
