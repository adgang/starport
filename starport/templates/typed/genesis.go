package typed

import (
	"context"
	"fmt"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
	"github.com/tendermint/starport/starport/pkg/astutils"
	"github.com/tendermint/starport/starport/pkg/protoanalysis"
)

// ProtoGenesisStateMessage is the name of the proto message that represents the genesis state
const ProtoGenesisStateMessage = "GenesisState"

// GenesisStateHighestFieldNumber returns the highest field number in the genesis state proto message
// This allows to determine next the field numbers
func GenesisStateHighestFieldNumber(path string) (int, error) {
	pkgs, err := protoanalysis.Parse(context.Background(), nil, path)
	if err != nil {
		return 0, err
	}
	if len(pkgs) == 0 {
		return 0, fmt.Errorf("%s is not a proto file", path)
	}
	m, err := pkgs[0].MessageByName(ProtoGenesisStateMessage)
	if err != nil {
		return 0, err
	}

	return m.HighestFieldNumber, nil
}

func AddKeysToDefaultGenesisState(dstHelper *astutils.DstHelper, key string, typeName string) {

	applyFunc := func(cursor *dstutil.Cursor) bool {
		switch x := cursor.Node().(type) {

		case *dst.FuncDecl:
			list := x.Body.List
			if x.Name.Name == "DefaultGenesis" {

				ret := list[len(list)-1].(*dst.ReturnStmt)

				exp := ret.Results[0].(*dst.UnaryExpr)

				lit := exp.X.(*dst.CompositeLit)
				if lit.Type.(*dst.Ident).Name == "GenesisState" {
					newExpr := &dst.KeyValueExpr{Key: &dst.Ident{Name: key},
						Value: &dst.CompositeLit{Type: &dst.ArrayType{Elt: &dst.Ident{Name: typeName}}}}

					newExpr.Decorations().Before = dst.NewLine
					newExpr.Decorations().After = dst.NewLine

					lit.Elts = append(lit.Elts, newExpr)
				}
			}
		default:
		}
		return true
	}
	dstutil.Apply(dstHelper.DstFile(), nil, applyFunc)

}

func nodeToFunction(node dst.Node) *dst.FuncDecl {
	return node.(*dst.FuncDecl)
}

func AddGenesisStateValidation(dstHelper *astutils.DstHelper, expressionList string) error {

	selectors := []astutils.NodeSelector{
		{
			Map: astutils.FuctionFinder("Validate"),
		},
		{
			Filter: astutils.FunctionMatcher("Validate"),
			Map: func(node interface{}) dst.Node {
				body := (node.(*dst.FuncDecl)).Body
				lines := body.List
				lastLineIndex := len(lines) - 1

				templateText := fmt.Sprintf(`package unknown
				func placeholder() {
					%s
				}
				`, expressionList)

				dstF, err := decorator.Parse(templateText)

				if err != nil {
					return nil
				}
				statements := dstF.Decls[0].(*dst.FuncDecl).Body.List
				returnStmt := body.List[lastLineIndex]
				statements[0].Decorations().Before = dst.EmptyLine
				statements[len(statements)-1].Decorations().After = dst.EmptyLine
				body.List = append(body.List[0:lastLineIndex], statements...)
				body.List = append(body.List, returnStmt)

				return (node.(*dst.FuncDecl))

			},
		},
	}

	walker := astutils.NewNodeWalker(selectors)

	node, err := walker.Slide(dstHelper.DstFile())
	if err != nil {
		return fmt.Errorf("could not find function to update file")
	}
	if node != nil {

		return nil
	}

	return fmt.Errorf("could not find place to update file")
}

func AddToModuleInitGenesis(dstHelper *astutils.DstHelper, expressionList string) error {

	return dstHelper.AppendToFunction("InitGenesis", expressionList)

}

func AddToModuleExportGenesis(dstHelper *astutils.DstHelper, expressionList string) error {

	return dstHelper.AppendToFunctionBeforeLastStatement("ExportGenesis", expressionList)

}

func AddToTestGenesisRequire(dstHelper *astutils.DstHelper, expressionList string) error {

	return dstHelper.AppendToFunction("TestGenesis", expressionList)

}

func AddToTestGenesisState(dstHelper *astutils.DstHelper, expressionList string) error {

	vectorTemplate := `
	package vector

	func injector() {
		genesisState := types.GenesisState{
			%s
		}
}`

	injectorVector := fmt.Sprintf(vectorTemplate, expressionList)

	vectorSelectors := []astutils.NodeSelector{
		{
			Map: astutils.FuctionFinder("injector"),
		},
	}
	vectorDstHelper, _ := astutils.NewDstHelper("", injectorVector)

	vectorWalker := astutils.NewNodeWalker(vectorSelectors)

	vectorNode, _ := vectorWalker.Slide(vectorDstHelper.DstFile())

	fmt.Println(vectorNode)

	vectorFunction := vectorNode.(*dst.FuncDecl)

	functionName := "TestGenesis"
	selectors := []astutils.NodeSelector{
		{
			Map: astutils.FuctionFinder(functionName),
		},
		{
			Map: func(nodeOrFile interface{}) dst.Node {
				// dst.Print(nodeOrFile)

				functionDecl := nodeOrFile.(*dst.FuncDecl)
				dst.Print(functionDecl.Body.List[0:2])
				return functionDecl.Body.List[0]
			},
		},

		{
			Map: func(nodeOrFile interface{}) dst.Node {
				dst.Print(nodeOrFile)
				assignStmt := nodeOrFile.(*dst.AssignStmt)
				dst.Print(dst.Print(assignStmt.Rhs))
				// panic(1)
				compositeLit := assignStmt.Rhs[0].(*dst.CompositeLit)
				vectorAssignStmt := vectorFunction.Body.List[0].(*dst.AssignStmt)
				vectorCompositeLit := vectorAssignStmt.Rhs[0].(*dst.CompositeLit)

				// fmt.Println("-----")
				// dst.Print(compositeLit.Elts)

				compositeLit.Elts = append(compositeLit.Elts, vectorCompositeLit.Elts...)
				// fmt.Println("-----")
				assignStmt.Rhs[0] = dst.Clone(compositeLit).(dst.Expr)

				dst.Print(dst.Print(assignStmt))

				return compositeLit

			},
		},
	}

	walker := astutils.NewNodeWalker(selectors)

	node, err := walker.Slide(dstHelper.DstFile())
	if err != nil {
		return fmt.Errorf("could not find function %s to update file", functionName)
	}
	if node != nil {

		return nil
	}

	return fmt.Errorf("could not find place to update file")

}
