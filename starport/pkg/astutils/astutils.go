package astutils

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	printer "go/printer"

	// "go/printer"
	"go/token"
	"os"

	"golang.org/x/tools/go/ast/astutil"
)

type AstHelper struct {
	fileSet *token.FileSet
	astFile *ast.File
	file    string
}

func (astHelper *AstHelper) AstFile() *ast.File {
	return astHelper.astFile
}

func (astHelper *AstHelper) FileSet() *token.FileSet {
	return astHelper.fileSet
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

func (astHelper *AstHelper) AddNamedImport(pkg string, name string) (done bool) {

	alreadyImported := astutil.UsesImport(astHelper.astFile, pkg)

	if !alreadyImported {
		fmt.Println("adding import:" + pkg + " as " + name)
		astutil.AddNamedImport(astHelper.fileSet, astHelper.astFile, name, pkg)
	} else {
		fmt.Println("import of " + pkg + " already added")
	}

	return !alreadyImported
}

func (astHelper *AstHelper) AddToState(typeName string) {

	key := fmt.Sprintf(`%[1]vList`, typeName)
	value := fmt.Sprintf(`%[1]v`, typeName)

	fmt.Println(key, value)
	applyFunc := func(cursor *astutil.Cursor) bool {
		switch x := cursor.Node().(type) {

		case *ast.FuncDecl:
			list := x.Body.List
			if x.Name.Name == "DefaultGenesis" {
				ret := list[len(list)-1].(*ast.ReturnStmt)

				exp := ret.Results[0].(*ast.UnaryExpr)

				lit := exp.X.(*ast.CompositeLit)
				if lit.Type.(*ast.Ident).Name == "GenesisState" {
					// ast.Print(astHelper.fileSet, lit.Elts)

					lit.Elts = append(lit.Elts, &ast.KeyValueExpr{Key: &ast.Ident{Name: key},
						Value: &ast.CompositeLit{Type: &ast.ArrayType{Elt: &ast.Ident{Name: value}}}})

					// ast.Print(astHelper.fileSet, lit.Elts)

				}
				// ast.Print(astHelper.fileSet, ret)
				// for _, res := range ret.Results {
				// 	fmt.Println(res)
				// }
				fmt.Println(x.Name)
			}
		default:
		}
		return true
	}
	astutil.Apply(astHelper.astFile, nil, applyFunc)
	// astHelper.astFile.Decls
}

func (astHelper *AstHelper) Print() {
	printer.Fprint(os.Stdout, astHelper.fileSet, astHelper.astFile)
}

func (astHelper *AstHelper) Write() error {

	var buf bytes.Buffer

	format.Node(&buf, astHelper.fileSet, astHelper.astFile)

	// f, err := os.Create(astHelper.file)
	// fmt.Println("Couldnot open file")
	// if err != nil {
	// 	return err
	// }
	// ast.Fprint(os.Stdout, astHelper.fileSet, astHelper.astFile, nil)
	// return ast.Fprint(f, astHelper.fileSet, astHelper.astFile, nil)
	return os.WriteFile(astHelper.file, buf.Bytes(), 0777)
	// fmt.Println("writing to file")
	// return printer.Fprint(f, astHelper.fileSet, astHelper.astFile)

}

func (astHelper *AstHelper) Content() (string, error) {

	var buf bytes.Buffer

	err := format.Node(&buf, astHelper.fileSet, astHelper.astFile)
	fmt.Println("errr:")
	if err != nil {
		fmt.Println(err)
		return buf.String(), err
	}
	// fmt.Println(buf.String())

	// f, err := os.Create(astHelper.file)
	// fmt.Println("Couldnot open file")
	// if err != nil {
	// 	return err
	// }
	// ast.Fprint(os.Stdout, astHelper.fileSet, astHelper.astFile, nil)
	// return ast.Fprint(f, astHelper.fileSet, astHelper.astFile, nil)
	return buf.String(), nil
	// fmt.Println("writing to file")
	// return printer.Fprint(f, astHelper.fileSet, astHelper.astFile)

}
