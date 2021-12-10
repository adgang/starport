package astutils

import (
	"fmt"
	"go/token"
	"testing"

	"github.com/dave/dst"
	"github.com/stretchr/testify/require"
)

func TestDstHelperContent(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name: "parsing simple package",
			input: `
	package blah
`,
			output: `package blah
`,
		},

		{
			name: "parsing package with imports",
			input: `
	package blah
		import "fmt"
		import "buf"
`,
			output: `package blah

import "fmt"
import "buf"
`,
		},

		{
			name: "sorting import group",
			input: `
	package blah
		import (
			"fmt"
		 "buf"
		)

`,
			output: `package blah

import (
	"buf"
	"fmt"
)
`,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			helper, err := NewDstHelper("", tc.input)
			require.NoError(t, err)
			helper.Content()
			var content string
			content, err = helper.Content()
			require.Equal(t, tc.output, content)

		})
	}
}

func TestAddImport(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		output     string
		pkg        string
		added      bool
		err        error
		importName string
	}{

		{
			name:  "adding to empty imports",
			input: `package blah`,
			pkg:   "ast",
			added: true,

			output: `package blah

import "ast"
`,
		},

		{
			name: "adding an import twice",
			input: `package blah
import "ast"`,
			pkg:   "ast",
			added: false,
			output: `package blah

import "ast"
`,
		},

		{
			name: "adding an import twice to an import group",
			input: `package blah
import ("ast")`,
			pkg:   "ast",
			added: false,
			output: `package blah

import (
	"ast"
)
`,
		},

		{
			name:  "adding to an imports group",
			pkg:   "ast",
			added: true,
			input: `package blah
import (
	"io"
	)			
`,

			output: `package blah

import (
	"ast"
	"io"
)
`,
		},

		{
			name:  "adding with multiple imports group",
			pkg:   "ast",
			added: true,

			input: `package blah
import (
	"io"
	)			
	import (
		"buf"
		)			
	
`,
			output: `package blah

import (
	"ast"
	"buf"
	"io"
)
`,
		},

		{
			name:  "adding with multiple imports group",
			pkg:   "ast",
			added: true,

			input: `package blah
import (
	"io"
	)			
	import (
		"buf"
		)	

		import "cat"		
		import "delta"
	
`,
			output: `package blah

import (
	"ast"
	"buf"
	"cat"
	"delta"
	"io"
)
`,
		},

		{
			name:  "adding an existing import name",
			pkg:   "ast",
			added: false,
			input: `package blah
import (
	"pa/ast"
	)			
`,

			output: ``,
			err:    fmt.Errorf("ast cannot be added as an import due to scope collision"),
		},

		{
			name:       "adding named import to empty imports",
			input:      `package blah`,
			pkg:        "ast",
			added:      true,
			importName: "xyz",

			output: `package blah

import xyz "ast"
`,
		},

		{
			name: "collision of named/aliased import",
			input: `package blah
import "ast"
`,
			pkg:        "bla",
			added:      false,
			importName: "ast",

			output: `package blah

import ast "xyz"
`,
			err: fmt.Errorf("bla cannot be added as an import due to scope collision"),
		},

		{
			name: "collision of named/aliased import with another aliased",
			input: `package blah
import ast "ariz"
`,
			pkg:        "bla",
			added:      false,
			importName: "ast",

			output: `package blah

import ast "xyz"
`,
			err: fmt.Errorf("bla cannot be added as an import due to scope collision"),
		},

		{
			name: "collision of named/aliased import with misc others",
			input: `package blah
import ast "ariz"
import "buf"
`,
			pkg:        "bla",
			added:      true,
			importName: "bat",

			output: `package blah

import (
	ast "ariz"
	bat "bla"
	"buf"
)
`,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			helper, err := NewDstHelper("", tc.input)

			var added bool

			if tc.importName == "" {
				added, err = helper.AddImport(tc.pkg)

			} else {
				added, err = helper.AddNamedImport(tc.pkg, tc.importName)

			}
			fmt.Println(err)
			require.Equal(t, tc.added, added)

			if tc.err == nil {
				require.NoError(t, err)
				var content string
				content, err = helper.Content()
				require.Equal(t, tc.output, content)

			} else {
				require.EqualError(t, err, tc.err.Error())
			}

		})
	}
}

func TestNodeWalker(t *testing.T) {

	tests := []struct {
		name   string
		input  string
		vector string
		output string
		err    error
	}{{name: "first",
		input: `
package testing

func foo() {

}
		`,
		output: `package testing

func foo() {

	a := 123
}
`,
	}}
	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			helper, err := NewDstHelper("", tc.input)

			assignSmt := &dst.AssignStmt{Tok: token.DEFINE, Lhs: []dst.Expr{&dst.Ident{Name: "a"}}, Rhs: []dst.Expr{&dst.BasicLit{Kind: token.STRING, Value: "123"}}}

			// assignSmt := &dst.AssignStmt{Tok: token.ASSIGN, Lhs: []dst.Expr{&dst.Ident{Name: "list21IdMap"}}, Rhs: []dst.Expr{&dst.BasicLit{Kind: token.STRING, Value: "\"abc\""}}}
			selectors := []NodeSelector{
				{
					Map: FuctionFinder("foo"),
				},
				{
					Filter: FunctionMatcher("foo"),
					Map: func(node interface{}) dst.Node {
						fmt.Println("mapping...")
						funDecl := (node).(*dst.FuncDecl)
						body := funDecl.Body

						body.List = append(body.List, assignSmt)
						return funDecl
					},
				},
			}

			walker := NewNodeWalker(selectors)
			_, err = walker.Slide(helper.dstFile)

			if tc.err == nil {
				require.NoError(t, err)
				var content string
				content, err = helper.Content()
				require.Equal(t, tc.output, content)

			} else {
				require.EqualError(t, err, tc.err.Error())
			}

		})
	}

}

func TestNodeInjector(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		vector string
		output string
		err    error
	}{{name: "first",
		input: `
package testing

func injectee() {

}
		`,
		vector: `
package vector

func injector() {
	
	a := 123
	b := 6787
}
		`,
		output: `package testing

func injectee() {

	a := 123
	b := 6787
}
`,
	},
	}
	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			helper, err := NewDstHelper("", tc.input)

			vectorSelectors := []NodeSelector{
				{
					Map: FuctionFinder("injector"),
				},
			}
			vectorDstHelper, _ := NewDstHelper("", tc.vector)

			vectorWalker := NewNodeWalker(vectorSelectors)

			vectorNode, _ := vectorWalker.Slide(vectorDstHelper.dstFile)

			vectorFunction := vectorNode.(*dst.FuncDecl)
			_ = vectorFunction

			selectors := []NodeSelector{
				{
					Map: FuctionFinder("injectee"),
				},
				{
					Map: func(nodeOrFile interface{}) dst.Node {

						switch t := nodeOrFile.(type) {
						default:
							fmt.Println("asd")
							fmt.Println(t)
							fmt.Println(nodeOrFile.(*dst.FuncDecl).Body)

						}

						_ = vectorFunction
						functionDecl := nodeOrFile.(*dst.FuncDecl)
						functionDecl.Body.List = append(functionDecl.Body.List, vectorFunction.Body.List...)
						return nodeOrFile.(dst.Node)
					},
				},
			}

			walker := NewNodeWalker(selectors)
			walker.Slide(helper.dstFile)

			if tc.err == nil {
				require.NoError(t, err)
				var content string
				content, err = helper.Content()
				require.Equal(t, tc.output, content)

			} else {
				require.EqualError(t, err, tc.err.Error())
			}

		})
	}

}

func TestFunctionRhsNodeInjector(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		vector string
		output string
		err    error
	}{
		{
			name: "inject into rhs object in a function",
			input: `package something

func injectee(t *testing.T) {
	genesisState := types.GenesisState{

		Params2: types.DefaultParams(),
	}
}

	`,
			vector: `
	package vector

	func injector() {
		genesisState := types.GenesisState{
			Params1: types.DefaultParams(),
		}
}
			`,

			output: `package something

func injectee(t *testing.T) {
	genesisState := types.GenesisState{

		Params2: types.DefaultParams(),
		Params1: types.DefaultParams(),
	}
}
`,
		},
	}
	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			helper, err := NewDstHelper("", tc.input)

			vectorSelectors := []NodeSelector{
				{
					Map: FuctionFinder("injector"),
				},
			}
			vectorDstHelper, _ := NewDstHelper("", tc.vector)

			vectorWalker := NewNodeWalker(vectorSelectors)

			vectorNode, _ := vectorWalker.Slide(vectorDstHelper.dstFile)

			fmt.Println(vectorNode)

			vectorFunction := vectorNode.(*dst.FuncDecl)

			// vectorAssignStmt := vectorFunction.Body.List[0].(*dst.AssignStmt)
			// vectorCompositeLit := vectorAssignStmt.Rhs[0].(*dst.CompositeLit)
			// dst.Print(vectorCompositeLit.Elts)

			_ = vectorFunction

			selectors := []NodeSelector{
				{
					Map: FuctionFinder("injectee"),
				},
				{
					Map: func(nodeOrFile interface{}) dst.Node {

						dst.Print(nodeOrFile)
						functionDecl := nodeOrFile.(*dst.FuncDecl)
						return functionDecl.Body.List[0]
					},
				},
				{
					Map: func(nodeOrFile interface{}) dst.Node {

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

			walker := NewNodeWalker(selectors)
			walker.Slide(helper.dstFile)
			helper.Print()

			if tc.err == nil {
				require.NoError(t, err)
				var content string
				content, err = helper.Content()
				require.Equal(t, tc.output, content)

			} else {
				require.EqualError(t, err, tc.err.Error())
			}

		})
	}
}

func TestFunctionForLoopStructNodeInjector(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		vector string
		output string
		err    error
	}{
		{
			name: "inject into struct of for loop in a function",
			input: `package something

func injectee(t *testing.T) {
	genesisState := types.GenesisState{

		Params2: types.DefaultParams(),
	}
}

	`,
			vector: `
	package vector

	func injector() {
		genesisState := types.GenesisState{
			Params1: types.DefaultParams(),
		}
}
			`,

			output: `package something

func injectee(t *testing.T) {
	genesisState := types.GenesisState{

		Params2: types.DefaultParams(),
		Params1: types.DefaultParams(),
	}
}
`,
		},
	}
	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			helper, err := NewDstHelper("", tc.input)

			vectorSelectors := []NodeSelector{
				{
					Map: FuctionFinder("injector"),
				},
			}
			vectorDstHelper, _ := NewDstHelper("", tc.vector)

			vectorWalker := NewNodeWalker(vectorSelectors)

			vectorNode, _ := vectorWalker.Slide(vectorDstHelper.dstFile)

			fmt.Println(vectorNode)

			vectorFunction := vectorNode.(*dst.FuncDecl)

			// vectorAssignStmt := vectorFunction.Body.List[0].(*dst.AssignStmt)
			// vectorCompositeLit := vectorAssignStmt.Rhs[0].(*dst.CompositeLit)
			// dst.Print(vectorCompositeLit.Elts)

			_ = vectorFunction

			selectors := []NodeSelector{
				{
					Map: FuctionFinder("injectee"),
				},
				{
					Map: func(nodeOrFile interface{}) dst.Node {

						dst.Print(nodeOrFile)
						functionDecl := nodeOrFile.(*dst.FuncDecl)
						return functionDecl.Body.List[0]
					},
				},
				{
					Map: func(nodeOrFile interface{}) dst.Node {

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

			walker := NewNodeWalker(selectors)
			walker.Slide(helper.dstFile)
			helper.Print()

			if tc.err == nil {
				require.NoError(t, err)
				var content string
				content, err = helper.Content()
				require.Equal(t, tc.output, content)

			} else {
				require.EqualError(t, err, tc.err.Error())
			}

		})
	}
}
