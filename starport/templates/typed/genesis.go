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

	vectorFunction := vectorNode.(*dst.FuncDecl)

	functionName := "TestGenesis"
	selectors := []astutils.NodeSelector{
		{
			Map: astutils.FuctionFinder(functionName),
		},
		{
			Map: func(nodeOrFile interface{}) dst.Node {

				functionDecl := nodeOrFile.(*dst.FuncDecl)
				return functionDecl.Body.List[0]
			},
		},

		{
			Map: func(nodeOrFile interface{}) dst.Node {
				assignStmt := nodeOrFile.(*dst.AssignStmt)
				compositeLit := assignStmt.Rhs[0].(*dst.CompositeLit)
				vectorAssignStmt := vectorFunction.Body.List[0].(*dst.AssignStmt)
				vectorCompositeLit := vectorAssignStmt.Rhs[0].(*dst.CompositeLit)

				vectorElts := vectorCompositeLit.Elts
				lastElt := vectorElts[len(vectorElts)-1]
				vectorElts[0].Decorations().Before = dst.EmptyLine
				lastElt.Decorations().After = dst.NewLine
				compositeLit.Elts = append(compositeLit.Elts, vectorCompositeLit.Elts...)

				assignStmt.Rhs[0] = dst.Clone(compositeLit).(dst.Expr)

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

func AddToTypesTestGenesisState(dstHelper *astutils.DstHelper, expressionList string) error {

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

	vectorFunction := vectorNode.(*dst.FuncDecl)

	functionName := "TestGenesisState_Validate"
	selectors := []astutils.NodeSelector{
		{
			Map: astutils.FuctionFinder(functionName),
		},
		{
			Map: func(nodeOrFile interface{}) dst.Node {

				functionDecl := nodeOrFile.(*dst.FuncDecl)
				return functionDecl.Body.List[0]
			},
		},

		{
			Map: func(nodeOrFile interface{}) dst.Node {
				assignStmt := nodeOrFile.(*dst.AssignStmt)
				compositeLit := assignStmt.Rhs[0].(*dst.CompositeLit)
				vectorAssignStmt := vectorFunction.Body.List[0].(*dst.AssignStmt)
				vectorCompositeLit := vectorAssignStmt.Rhs[0].(*dst.CompositeLit)

				vectorElts := vectorCompositeLit.Elts
				lastElt := vectorElts[len(vectorElts)-1]
				vectorElts[0].Decorations().Before = dst.EmptyLine
				lastElt.Decorations().After = dst.NewLine
				compositeLit.Elts = append(compositeLit.Elts, vectorCompositeLit.Elts...)

				assignStmt.Rhs[0] = dst.Clone(compositeLit).(dst.Expr)

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

func AddTestToGenesisStateValidate(dstHelper *astutils.DstHelper, expressionList string) error {

	vectorTemplate := `
	package vector

	func injector() {
		genesisState := []struct{}{
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

	vectorFunction := vectorNode.(*dst.FuncDecl)

	functionName := "TestGenesisState_Validate"
	selectors := []astutils.NodeSelector{
		{
			Map: astutils.FuctionFinder(functionName),
		},
		{
			Map: func(nodeOrFile interface{}) dst.Node {

				functionDecl := nodeOrFile.(*dst.FuncDecl)
				return functionDecl.Body.List[1]
			},
		},

		{
			Map: func(nodeOrFile interface{}) dst.Node {
				assignStmt := nodeOrFile.(*dst.AssignStmt)
				compositeLit := assignStmt.Rhs[0].(*dst.CompositeLit)
				vectorAssignStmt := vectorFunction.Body.List[0].(*dst.AssignStmt)
				vectorCompositeLit := vectorAssignStmt.Rhs[0].(*dst.CompositeLit)

				vectorElts := vectorCompositeLit.Elts
				if l := len(vectorElts) - 1; l >= 0 {
					lastElt := vectorElts[l]
					vectorElts[0].Decorations().Before = dst.EmptyLine
					lastElt.Decorations().After = dst.NewLine

				}
				compositeLit.Elts = append(compositeLit.Elts, vectorCompositeLit.Elts...)

				assignStmt.Rhs[0] = dst.Clone(compositeLit).(dst.Expr)

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

func AddSingletonToInitGenesis(dstHelper *astutils.DstHelper, expressionList string) error {

	vectorTemplate := `
	package vector

	func injector() {
			%s
		
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

	vectorFunction := vectorNode.(*dst.FuncDecl)

	functionName := "InitGenesis"
	selectors := []astutils.NodeSelector{
		{
			Map: astutils.FuctionFinder(functionName),
		},
		{
			Map: func(nodeOrFile interface{}) dst.Node {

				functionDecl := nodeOrFile.(*dst.FuncDecl)
				statements := functionDecl.Body.List

				for i, st := range statements {
					if i == 2 {
						dst.Print(st)
						_ = vectorFunction
						panic(1)
					}
				}

				return functionDecl
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

func AddSingletonToModuleExport(dstHelper *astutils.DstHelper, expressionList string) error {

	vectorTemplate := `
	package vector

	func injector() {
			%s
		
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

	vectorFunction := vectorNode.(*dst.FuncDecl)

	functionName := "InitGenesis"
	selectors := []astutils.NodeSelector{
		{
			Map: astutils.FuctionFinder(functionName),
		},
		{
			Map: func(nodeOrFile interface{}) dst.Node {

				functionDecl := nodeOrFile.(*dst.FuncDecl)
				statements := functionDecl.Body.List

				for i, st := range statements {
					if i == 2 {
						dst.Print(st)
						_ = vectorFunction
						panic(1)
					}
				}

				return functionDecl
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

func AddSingletonToDefaultGenesisState(dstHelper *astutils.DstHelper, expressionList string) error {

	vectorTemplate := `
	package vector

	func injector() {
		list := SomeType {
			%s
		}
		
}`

	injectorVector := fmt.Sprintf(vectorTemplate, expressionList)

	vectorSelectors := []astutils.NodeSelector{
		{
			Map: astutils.FuctionFinder("injector"),
		},
		{
			Map: func(nodeOrFile interface{}) dst.Node {
				functionDecl := nodeOrFile.(*dst.FuncDecl)
				statements := functionDecl.Body.List

				rhs := statements[0].(*dst.AssignStmt).Rhs[0].(*dst.CompositeLit)
				return rhs
			},
		},
	}
	vectorDstHelper, _ := astutils.NewDstHelper("", injectorVector)

	vectorWalker := astutils.NewNodeWalker(vectorSelectors)

	vectorNode, _ := vectorWalker.Slide(vectorDstHelper.DstFile())

	functionName := "DefaultGenesis"
	anchorToken := "Params"
	selectors := []astutils.NodeSelector{
		{
			Map: astutils.FuctionFinder(functionName),
		},
		{
			Map: func(nodeOrFile interface{}) dst.Node {
				functionDecl := nodeOrFile.(*dst.FuncDecl)
				statements := functionDecl.Body.List

				retStmt := statements[len(statements)-1].(*dst.ReturnStmt)
				return retStmt
			},
		},
		{
			Map: func(nodeOrFile interface{}) dst.Node {

				retStmt := nodeOrFile.(*dst.ReturnStmt)

				compositeLit := retStmt.Results[0].(*dst.UnaryExpr).X.(*dst.CompositeLit)
				var tokenIndex int
				for i, elt := range compositeLit.Elts {
					if exp := elt.(*dst.KeyValueExpr); exp != nil {
						if anchorToken == exp.Key.(*dst.Ident).Name {
							tokenIndex = i
						}
					}
				}

				vectorKeyValPairs := (vectorNode).(*dst.CompositeLit).Elts

				trailingSlice := dst.Clone(compositeLit).(*dst.CompositeLit).Elts[tokenIndex:]
				compositeLit.Elts = append(compositeLit.Elts[0:tokenIndex], vectorKeyValPairs...)

				compositeLit.Elts = append(compositeLit.Elts, trailingSlice...)

				return retStmt
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
