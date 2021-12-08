package astutils

import (
	"fmt"
	"os"

	"github.com/jhump/protoreflect/desc/protoparse/ast"

	"github.com/jhump/protoreflect/desc/protoparse"
)

type PBAstHelper struct {
	parser   *protoparse.Parser
	fileNode *ast.FileNode
	file     string
}

func NewPBAstHelper(file string) (*PBAstHelper, error) {
	pbastHelper := new(PBAstHelper)
	pbastHelper.file = file

	pbastHelper.parser = &protoparse.Parser{}

	fileNode, err := pbastHelper.parser.ParseToAST(file)

	if err != nil {
		return nil, err
	}
	pbastHelper.fileNode = fileNode[0]

	fmt.Println("created node1")
	pbastHelper.Print()

	return pbastHelper, nil
}

func (pbastHelper *PBAstHelper) Close() {
	// TODO: cleanup the resources
}

func ApplyFunc(node *ast.Node) {
	// node as
}

func (pbastHelper *PBAstHelper) AddImport(pkg string) (done bool) {
	astFile := pbastHelper.fileNode

	for _, child := range astFile.Children() {
		fmt.Println(child.Start())
	}

	// ast.Walk(astFile)
	// panic(1)
	return true
}

func (pbastHelper *PBAstHelper) Print() {

	ast.Print(os.Stdout, pbastHelper.fileNode)
}

func (pbastHelper *PBAstHelper) Write() error {

	f, err := os.Open(pbastHelper.file)

	if err != nil {
		return err
	}
	ast.Print(f, pbastHelper.fileNode)
	return err

}
