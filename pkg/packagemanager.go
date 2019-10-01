package pkg

import (
	log "github.com/sirupsen/logrus"
	"go/ast"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

// WARNING
// Go exec does NOT glob commands, this has to be done MANUALLY
func copyDir(src, dst string) {
	var err error
	// First ensure the dst exists
	err = os.MkdirAll(dst, os.ModePerm)
	if err != nil {
		log.Fatalf("copyDir: error creating output directory: %+v", err)
	}
	src = filepath.Clean(src + string(filepath.Separator) + "*")
	dst = filepath.Clean(dst + string(filepath.Separator))
	if runtime.GOOS == "windows" {
		cmd := exec.Command("Xcopy", "/E /I ", src, dst)
		log.Debugf("copyDir: cmd: %s", cmd.String())
		err = cmd.Run()

	} else {
		p := []string{"-R", "-f", "-t", dst}
		s, err := filepath.Glob(src)
		if err != nil {

		}
		p = append(p, s...)
		cmd := exec.Command("cp", p...)
		log.Debugf("copyDir: cmd: %s", cmd.String())
		err = cmd.Run()
	}
	if err != nil {
		log.Fatalf("copyDir: error copying directory: %+v", err)
	}
}

func fixPermissions(src string) {
	var err error
	if runtime.GOOS == "windows" {
		cmd := exec.Command("Xcopy", "/E /I ", src)
		log.Debugf("fixPermissions: cmd: %s", cmd.String())
		err = cmd.Run()

	} else {
		cmd := exec.Command("chmod", "-R", "+rw", src)
		log.Debugf("fixPermissions: cmd: %s", cmd.String())
		err = cmd.Run()
	}
	if err != nil {
		log.Fatalf("fixPermissions: error: %+v", err)
	}
}

func fileContains(file string, query string) (ok bool) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Errorf("fileContains: file: %s err: %+v", file, err)
		return false
	}
	s := string(b)
	// //check whether s contains substring text
	return strings.Contains(s, query)
}
