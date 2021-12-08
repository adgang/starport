package astutils

import (
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
		name   string
		input  string
		output string
		pkg    string
		added  bool
		err    error
	}{
		{
			name:  "adding an import",
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
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			helper, err := NewDstHelper("", tc.input)

			added, err := helper.AddImport(tc.pkg)
			require.Equal(t, true, added)

			if tc.err == nil {
				require.NoError(t, err)
			}
			helper.Content()
			var content string
			content, err = helper.Content()
			require.Equal(t, tc.output, content)

		})
	}
}
