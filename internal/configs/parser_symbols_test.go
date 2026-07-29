// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/tsarna/functy"
	"github.com/zclconf/go-cty/cty/function"
)

// Test that the expected files are found, and no hidden ones
func TestLoadSymbolsSources(t *testing.T) {
	p := testParser(map[string]string{
		"a/foo.cty":     "a", // should be loaded
		"a/b/bar.cty":   "b", // should be loaded
		"a/.c/baz.cty":  "c", // should be skipped: in a hidden directory
		"a/b/.qux.cty":  "d", // should be skipped: hidden file
		"a/b/README.md": "e", // should be skipped: not a .cty file
		"d/x.cty":       "f", // Should not be found, not under "a"
	})

	sources, diags := p.LoadSymbolSources("a", hcl.Range{
		Filename: "foo.tf",
		Start:    hcl.Pos{Line: 2, Column: 3, Byte: 4},
		End:      hcl.Pos{Line: 2, Column: 8, Byte: 9},
	})

	assertDiagnosticCount(t, diags, 0)
	assert.Equal(t, 2, len(sources), "should find 2 .cty files")

	assert.Equal(t,
		[]functy.Source{
			{Filename: "a/b/bar.cty", Bytes: []byte("b")},
			{Filename: "a/foo.cty", Bytes: []byte("a")},
		},
		sources,
		"Should have found a/b/bar.cty and a/foo.cty in that (sorted) order",
	)

	assert.Contains(t, p.Sources(), "a/foo.cty")
	assert.Contains(t, p.Sources(), "a/b/bar.cty")
}

// Loading a directory that does not exist should fail
func TestMissingDirectory(t *testing.T) {
	p := testParser(nil)
	sources, diags := p.LoadSymbolSources("foo", hcl.Range{
		Filename: "foo.tf",
		Start:    hcl.Pos{Line: 2, Column: 3, Byte: 4},
		End:      hcl.Pos{Line: 2, Column: 8, Byte: 9},
	})

	assertExactDiagnostics(t, diags, []string{
		"foo.tf:2,3-8: Directory must exist; open foo: file does not exist",
	})
	assert.Empty(t, sources)
}

// Giving a non-directory should fail
func TestNonDirectory(t *testing.T) {
	p := testParser(map[string]string{
		"foo": "this a file, not a directory",
	})

	sources, diags := p.LoadSymbolSources("foo", hcl.Range{
		Filename: "foo.tf",
		Start:    hcl.Pos{Line: 2, Column: 3, Byte: 4},
		End:      hcl.Pos{Line: 2, Column: 8, Byte: 9},
	})

	assertExactDiagnostics(t, diags, []string{
		"foo.tf:2,3-8: Source must be a directory; foo is not a directory",
	})
	assert.Empty(t, sources)
}

// Loading an empty directory should succeed, but produce no sources
func TestEmptyDir(t *testing.T) {
	p := testParser(nil)
	sources, diags := p.LoadSymbolSources(".", hcl.Range{
		Filename: "foo.tf",
		Start:    hcl.Pos{Line: 2, Column: 3, Byte: 4},
		End:      hcl.Pos{Line: 2, Column: 8, Byte: 9},
	})

	assertDiagnosticCount(t, diags, 0)
	assert.Empty(t, sources)
}

func TestFunctyCoreFunctions(t *testing.T) {
	corefuncs := map[string]function.Function{
		"length": function.Function{}, // the actual function doesn't matter
	}

	p := testParser(nil)
	p.SetFunctions(corefuncs)
	funcs := p.FunctyCoreFunctions()

	assert.Contains(t, funcs, "length", "base functions should be present")
	assert.Contains(t, funcs, "typeof", "functy's stdlib extensions should be present")

	funcs["addedfunc"] = function.Function{}

	assert.Contains(t, funcs, "addedfunc", "addedfunc has been added to this copy")

	// Check that mutation doesn't leak into future calls

	funcs = p.FunctyCoreFunctions()

	assert.Contains(t, funcs, "length", "base functions should be present")
	assert.Contains(t, funcs, "typeof", "functy's stdlib extensions should be present")

	assert.NotContains(t, funcs, "addedfunc", "addedfunc from first map should not have leaked into this one")
}
