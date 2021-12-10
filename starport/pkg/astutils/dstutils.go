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
	done := astutil.AddNamedImport(fileSet, astFile, args[1].(string), args[0].(string))

	return nil, done
}

func lastFragment(pkgPath string) string {
	parts := strings.Split(pkgPath, "/")
	pkgName := parts[len(parts)-1]
	return pkgName
}

func importExists(dstFile *dst.File, pkg string, importName string) (bool, bool) {
	packageAlreadyImported := false
	collision := false

	// get a list of independent line imports
	imports := dstFile.Imports
	if nil != imports {
		for _, importSpec := range imports {
			pkgPath := strings.ReplaceAll(importSpec.Path.Value, "\"", "")
			if pkgPath == pkg {
				packageAlreadyImported = true
			} else {

				if importSpec.Name == nil {
					usedName := lastFragment(pkgPath)
					if usedName == importName {
						collision = true
						break
					}
				} else if importSpec.Name.Name == importName {
					collision = true
					break
				}
			}
		}
	}

	// TODO: get all import lists and test for collisions
	return collision, packageAlreadyImported
}

func (dstHelper *DstHelper) AddImport(pkg string) (done bool, err error) {

	importName := lastFragment(pkg)

	collision, packageAlreadyImported := importExists(dstHelper.dstFile, pkg, importName)

	if collision {
		return false, fmt.Errorf("%s cannot be added as an import due to scope collision", pkg)
	}

	if packageAlreadyImported {
		return false, nil
	}

	err, done = dstHelper.WithAst(addImportProcessor, pkg)

	return true, err
}

func (dstHelper *DstHelper) AddNamedImport(pkg string, name string) (done bool, err error) {
	collision, packageAlreadyImported := importExists(dstHelper.dstFile, pkg, name)
	if packageAlreadyImported {
		return false, nil
	}

	if collision {
		return false, fmt.Errorf("%s cannot be added as an import due to scope collision", pkg)
	}

	err, done = dstHelper.WithAst(addNamedImportProcessor, pkg, name)

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

type NodeFilter func(node dst.Node) bool
type NodeFileMapper func(nodeOrFile interface{}) dst.Node

type NodeSelector struct {
	Name   string
	Filter NodeFilter
	Map    NodeFileMapper
}

func (selector *NodeSelector) Process(node dst.Node) dst.Node {
	fmt.Println("processing")

	if selector.Filter == nil || selector.Filter(node) {
		newNode := selector.Map(node)
		fmt.Println(newNode)

		return newNode
	}

	return nil

}

type NodeWalker struct {
	selectors []NodeSelector
}

func NewNodeWalker(selectors []NodeSelector) NodeWalker {
	return NodeWalker{selectors: selectors}
}

func (walker NodeWalker) Slide(node dst.Node) (dst.Node, error) {
	curNode := node
	for _, selector := range walker.selectors {

		if curNode == nil {
			return curNode, fmt.Errorf("could not find %s while walking the dst node", selector.Name)
		}
		curNode = selector.Process(curNode)
	}
	return curNode, nil
}

func FunctionMatcher(name string) NodeFilter {
	return func(node dst.Node) bool {
		switch (node).(type) {
		case *dst.FuncDecl:
			return node.(*dst.FuncDecl).Name.Name == name
		default:
			return false
		}
	}
}

func FuctionFinder(name string) NodeFileMapper {

	// TODO: Currently it only finds functions in dst.File(top level of file). Add deeper finds if necessary
	return func(nodeOrFile interface{}) dst.Node {

		switch nodeOrFile.(type) {
		case *dst.File:
			file := *nodeOrFile.(*dst.File)
			for _, decl := range file.Decls {

				switch decl.(type) {
				case dst.Decl:
					declNode := decl.(dst.Decl)
					switch decl.(type) {
					case *dst.FuncDecl:
						functionDecl := decl.(*dst.FuncDecl)
						if functionDecl != nil && functionDecl.Name != nil && functionDecl.Name.Name == name {
							return declNode
						}

					}

				}
			}
			return nil
		default:
			return nil
		}
	}
}
