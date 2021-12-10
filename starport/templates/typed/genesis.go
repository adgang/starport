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
			Filter: astutils.FunctionMatcher("Validate"),
			Map: func(node dst.Node) dst.Node {
				body := (nodeToFunction(node)).Body
				lines := body.List
				lastLineIndex := len(lines) - 1
				// append(decs, dst.NewLine)
				// assignSmt := &dst.AssignStmt{Tok: token.ASSIGN, Lhs: []dst.Expr{&dst.Ident{Name: "list21IdMap"}}, Rhs: []dst.Expr{&dst.BasicLit{Kind: token.STRING, Value: "\"abc\""}}}
				// body.List = append(body.List[0:lastLineIndex-1], assignSmt, lines[lastLineIndex])

				// fmt.Println("----xxxxx")

				// assignSmt := &dst.AssignStmt{Tok: token.ASSIGN, Lhs: []dst.Expr{&dst.Ident{Name: "list21IdMap"}}, Rhs: []dst.Expr{&dst.BasicLit{Kind: token.STRING, Value: "\"abc\""}}}
				// body.List = append(body.List[0:lastLineIndex-1], dstF.Decls...)

				// dst.Print(lines[lastLineIndex-2 : lastLineIndex+1])

				// panic(1)
				// (nodeToFunction(node)).Body.List = append(lines, &dst.CommClause{Comm: dst.Stmt})
				// dst.Print((nodeToFunction(node)).Body.List)

				// dstF, err := decorator.Parse(
				// 	`package blah

				// func placeholder() {

				//  }
				// `)

				// templateText := fmt.Sprintf(`package unknown
				// func placeholder() {
				// 	%s
				// }
				// `, "asd := 123")

				templateText := fmt.Sprintf(`package unknown
				func placeholder() {
					%s
				}
				`, expressionList)

				dstF, err := decorator.Parse(templateText)

				fmt.Println("----xxxxx")

				fmt.Println(err)
				fmt.Println("----xxxxx")
				dst.Print(dstF)
				statements := dstF.Decls[0].(*dst.FuncDecl).Body.List
				returnStmt := body.List[lastLineIndex]
				statements[0].Decorations().Before = dst.EmptyLine
				statements[len(statements)-1].Decorations().After = dst.EmptyLine
				body.List = append(body.List[0:lastLineIndex], statements...)
				body.List = append(body.List, returnStmt)

				return nil
			},
		},
	}

	for _, decl := range dstHelper.DstFile().Decls {
		walker := astutils.NewNodeWalker(selectors)

		node, err := walker.Slide(decl)
		if err != nil {
			return fmt.Errorf("could not find function to update file")
		}
		if node != nil {

			return nil
		}
	}

	return fmt.Errorf("could not find place to update file")
}
