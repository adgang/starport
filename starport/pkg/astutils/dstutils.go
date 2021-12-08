package astutils

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"golang.org/x/tools/go/ast/astutil"
)

type DstHelper struct {
	fileSet *token.FileSet
	dstFile *dst.File

	file string
}

func (dstHelper *DstHelper) DstFile() *dst.File {
	return dstHelper.dstFile
}

func (dstHelper *DstHelper) FileSet() *token.FileSet {
	return dstHelper.fileSet
}

func NewDstHelper(file string) (*DstHelper, error) {
	dstHelper := new(DstHelper)
	fileSet := token.NewFileSet()
	dstFile, err := decorator.ParseFile(fileSet, file, nil, parser.AllErrors|parser.ParseComments)

	dstHelper.fileSet = fileSet

	dstHelper.dstFile = dstFile

	dstHelper.file = file

	return dstHelper, err
}

func (dstHelper *DstHelper) Close() {
	// TODO: cleanup the resources
}

type astFileProcess = func(astFile *ast.File, fileSet *token.FileSet, args ...interface{}) (error, bool)

func (dstHelper *DstHelper) WithAst(process astFileProcess, fileSet *token.FileSet, args ...interface{}) (error, bool) {
	restorer := decorator.NewRestorer()
	astFile, err := restorer.RestoreFile(dstHelper.dstFile)
	if err != nil {
		return err, false
	}
	done := false
	err, done = process(astFile, fileSet, args...)

	dstHelper.dstFile, err = decorator.DecorateFile(restorer.Fset, astFile)
	return err, done
}

func addImportProcessor(astFile *ast.File, fileSet *token.FileSet, args ...interface{}) (error, bool) {
	done := astutil.AddImport(fileSet, astFile, args[0].(string))

	return nil, done
}

func addNamedImportProcessor(astFile *ast.File, fileSet *token.FileSet, args ...interface{}) (error, bool) {
	done := astutil.AddNamedImport(fileSet, astFile, args[0].(string), args[1].(string))

	return nil, done
}

func lastFragment(pkgPath string) string {
	parts := strings.Split(pkgPath, "/")
	pkgName := parts[len(parts)-1]
	return pkgName
}

func importExists(dstFile *dst.File, pkg string) bool {
	importName := lastFragment(pkg)

	// get a list of independent line imports
	imports := dstFile.Imports
	if nil != imports {
		for _, importSpec := range imports {
			if importSpec.Name == nil {
				usedName := lastFragment(importSpec.Path.Value)
				if usedName == importName {
					return true
				}
			} else if importSpec.Name.Name == importName {
				return true
			} else {

			}
		}
	}

	// TODO: get all import lists and test for collisions
	return false
}

func (dstHelper *DstHelper) AddImport(pkg string) (done bool, err error) {

	if importExists(dstHelper.dstFile, pkg) {
		return false, fmt.Errorf("%s cannot be added as an import due to scope collision", pkg)
	}

	alreadyImported := false
	err, done = dstHelper.WithAst(addImportProcessor, dstHelper.fileSet, pkg)
	if err != nil {
		return false, err
	}

	return !alreadyImported, err
}

func (dstHelper *DstHelper) AddNamedImport(pkg string, name string) (done bool, err error) {

	alreadyImported := false
	err, done = dstHelper.WithAst(addNamedImportProcessor, dstHelper.fileSet, pkg)
	if err != nil {
		return false, err
	}

	return !alreadyImported, err
}

func (dstHelper *DstHelper) Print() {
	decorator.Fprint(os.Stdout, dstHelper.dstFile)
}

func (dstHelper *DstHelper) Write() error {

	f, err := os.OpenFile(dstHelper.file, os.O_CREATE, 0755)

	if err != nil {
		return err
	}
	return decorator.NewRestorer().Fprint(f, dstHelper.dstFile)

}

func (dstHelper *DstHelper) Content() (string, error) {
	var buf bytes.Buffer

	err := decorator.NewRestorer().Fprint(&buf, dstHelper.dstFile)
	return buf.String(), err
}
