package astutils

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"

	"golang.org/x/tools/go/ast/astutil"
)

type AstHelper struct {
	fileSet *token.FileSet
	astFile *ast.File
	file    string
}

func NewAstHelper(file string) (*AstHelper, error) {
	astHelper := new(AstHelper)
	fileSet := token.NewFileSet()

	astFile, err := parser.ParseFile(fileSet, file, nil, parser.AllErrors|parser.ParseComments)

	astHelper.fileSet = fileSet

	astHelper.astFile = astFile

	astHelper.file = file

	return astHelper, err
}

func (astHelper *AstHelper) Close() {
	// TODO: cleanup the resources
}

func (astHelper *AstHelper) AddImport(pkg string) (done bool) {

	alreadyImported := astutil.UsesImport(astHelper.astFile, pkg)

	if !alreadyImported {
		fmt.Println("adding import:" + pkg)
		astutil.AddImport(astHelper.fileSet, astHelper.astFile, pkg)
	} else {
		fmt.Println("import of " + pkg + " already added")
	}

	return !alreadyImported
}

func (astHelper *AstHelper) Print() {
	printer.Fprint(os.Stdout, astHelper.fileSet, astHelper.astFile)
}

func (astHelper *AstHelper) Write() error {

	var buf bytes.Buffer

	format.Node(&buf, astHelper.fileSet, astHelper.astFile)

	return os.WriteFile(astHelper.file, buf.Bytes(), 0777)
}
