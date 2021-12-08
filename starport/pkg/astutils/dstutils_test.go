package astutils

import (
	"fmt"
	"testing"

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
