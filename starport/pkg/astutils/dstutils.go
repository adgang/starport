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

func NewDstHelper(file string, inputs ...interface{}) (*DstHelper, error) {
	dstHelper := new(DstHelper)
	fileSet := token.NewFileSet()
	var src interface{}
	if len(inputs) > 0 {
		src = inputs[0]
	} else {
		src = nil
	}
	dstFile, err := decorator.ParseFile(fileSet, file, src, parser.AllErrors|parser.ParseComments)

	dstHelper.fileSet = fileSet

	dstHelper.dstFile = dstFile

	dstHelper.file = file

	return dstHelper, err
}

func (dstHelper *DstHelper) Close() {
	// TODO: cleanup the resources
}

type astFileProcess = func(astFile *ast.File, fileSet *token.FileSet, args ...interface{}) (error, bool)

func (dstHelper *DstHelper) WithAst(process astFileProcess, args ...interface{}) (error, bool) {
	restorer := decorator.NewRestorer()
	astFile, err := restorer.RestoreFile(dstHelper.dstFile)
	if err != nil {
		return err, false
	}
	done := false
	err, done = process(astFile, dstHelper.fileSet, args...)

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

func importExists(dstFile *dst.File, pkg string) (bool, bool) {
	importName := lastFragment(pkg)
	packageAlreadyImported := false

	// get a list of independent line imports
	imports := dstFile.Imports
	if nil != imports {
		for _, importSpec := range imports {
			pkgPath := strings.ReplaceAll(importSpec.Path.Value, "\"", "")
			fmt.Println(pkgPath, pkg)
			if pkgPath == pkg {
				packageAlreadyImported = true
				return false, true
			}

			if importSpec.Name == nil {
				usedName := lastFragment(pkgPath)
				if usedName == importName {
					return true, false
				}
			} else if importSpec.Name.Name == importName {
				return true, false
			} else {

			}
		}
	}

	// TODO: get all import lists and test for collisions
	return false, packageAlreadyImported
}

func (dstHelper *DstHelper) AddImport(pkg string) (done bool, err error) {

	collision, packageAlreadyImported := importExists(dstHelper.dstFile, pkg)

	fmt.Println(collision, packageAlreadyImported)
	if packageAlreadyImported {
		return false, nil
	}
	if collision {
		return false, fmt.Errorf("%s cannot be added as an import due to scope collision", pkg)
	}

	err, done = dstHelper.WithAst(addImportProcessor, pkg)

	return true, err
}

func (dstHelper *DstHelper) AddNamedImport(pkg string, name string) (done bool, err error) {
	collision, packageAlreadyImported := importExists(dstHelper.dstFile, pkg)
	if packageAlreadyImported {
		return false, nil
	}

	if collision {
		return false, fmt.Errorf("%s cannot be added as an import due to scope collision", pkg)
	}

	err, done = dstHelper.WithAst(addNamedImportProcessor, pkg)

	return true, err
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

func organizeImports(fileAst *ast.File, fileSet *token.FileSet, args ...interface{}) (error, bool) {
	ast.SortImports(fileSet, fileAst)
	return nil, true
}

func (dstHelper *DstHelper) Content() (string, error) {
	var buf bytes.Buffer

	// dstHelper.WithAst(organizeImports)

	err := decorator.NewRestorer().Fprint(&buf, dstHelper.dstFile)

	return buf.String(), err
}
