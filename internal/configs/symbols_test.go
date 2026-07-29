// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/opentofu/opentofu/internal/addrs"
	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func TestLoadSymbols(t *testing.T) {
	src, err := os.ReadFile("testdata/invalid-files/symbols.tf")
	if err != nil {
		t.Fatal(err)
	}

	parser := testParser(map[string]string{
		"symbols.tf": string(src),
	})

	_, diags := parser.LoadConfigFile("symbols.tf")
	assertExactDiagnostics(t, diags, []string{
		`symbols.tf:1,9-18: Invalid symbols block name; A name must start with a letter or underscore and may contain only letters, digits, underscores, and dashes.`,
		`symbols.tf:6,14-18: Invalid source; Null is not allowed for source.`,
		`symbols.tf:10,14-16: Invalid source; An empty string is not allowed for source.`,
		`symbols.tf:14,14-17: Variables not allowed; Variables may not be used here.`,
		`symbols.tf:18,14-19: Function calls not allowed; Functions may not be called here.`,
		`symbols.tf:23,17-20: Variables not allowed; Variables may not be used here.`,
		`symbols.tf:28,17-22: Function calls not allowed; Functions may not be called here.`,
		`symbols.tf:31,20-20: Missing required argument; The argument "source" is required, but no definition was found.`,
		`symbols.tf:36,14-31: Invalid source; A string value is required for source.`,
		`symbols.tf:41,17-34: Invalid namespace; A string value is required for namespace.`,
		`symbols.tf:46,5-10: Unsupported argument; An argument named "prime" is not expected here.`,
		`symbols.tf:50,14-19: Invalid source address; OpenTofu failed to determine your intended installation method for remote symbols package "abc".` +
			"\n\n" + `If you intended this as a local relative path, use "./abc" instead. The "./" prefix indicates that the address is a relative filesystem path.`,
		`symbols.tf:54,14-47: Invalid source address; Only local source addresses are allowed for symbols, but "git::https://github.com/foo/bar" is not a local address.`,
	})

	src, err = os.ReadFile("testdata/valid-modules/symbols/symbols.tf")
	if err != nil {
		t.Fatal(err)
	}

	parser = testParser(map[string]string{
		"symbols.tf": string(src),
	})

	file, diags := parser.LoadConfigFile("symbols.tf")

	gotSymbols := file.SymbolsBlocks
	wantSymbols := []*Symbols{
		{
			Label:         "foo",
			SourceAddrRaw: "./foo",
			SourceAddr:    addrs.ModuleSourceLocal("./foo"),
			SourceAddrRange: hcl.Range{
				Filename: "symbols.tf",
				Start:    hcl.Pos{Line: 2, Column: 14, Byte: 29},
				End:      hcl.Pos{Line: 2, Column: 21, Byte: 36},
			},
			DeclRange: hcl.Range{
				Filename: "symbols.tf",
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 14, Byte: 13},
			},
		},
		{
			Label:         "bar",
			SourceAddrRaw: "./bar",
			SourceAddr:    addrs.ModuleSourceLocal("./bar"),
			SourceAddrRange: hcl.Range{
				Filename: "symbols.tf",
				Start:    hcl.Pos{Line: 6, Column: 14, Byte: 69},
				End:      hcl.Pos{Line: 6, Column: 21, Byte: 76},
			},
			DeclRange: hcl.Range{
				Filename: "symbols.tf",
				Start:    hcl.Pos{Line: 5, Column: 1, Byte: 40},
				End:      hcl.Pos{Line: 5, Column: 14, Byte: 53},
			},
		},
		{
			Label:         "with_ns",
			SourceAddrRaw: "./xyz",
			SourceAddr:    addrs.ModuleSourceLocal("./xyz"),
			SourceAddrRange: hcl.Range{
				Filename: "symbols.tf",
				Start:    hcl.Pos{Line: 10, Column: 14, Byte: 113},
				End:      hcl.Pos{Line: 10, Column: 21, Byte: 120},
			},
			Namespace:    "baz::qux",
			NamespaceSet: true,
			DeclRange: hcl.Range{
				Filename: "symbols.tf",
				Start:    hcl.Pos{Line: 9, Column: 1, Byte: 80},
				End:      hcl.Pos{Line: 9, Column: 18, Byte: 97},
			},
		},
		{
			Label:         "blank_ns",
			SourceAddrRaw: "./xyz",
			SourceAddr:    addrs.ModuleSourceLocal("./xyz"),
			SourceAddrRange: hcl.Range{
				Filename: "symbols.tf",
				Start:    hcl.Pos{Line: 15, Column: 14, Byte: 185},
				End:      hcl.Pos{Line: 15, Column: 21, Byte: 192},
			},
			Namespace:    "",
			NamespaceSet: true,
			DeclRange: hcl.Range{
				Filename: "symbols.tf",
				Start:    hcl.Pos{Line: 14, Column: 1, Byte: 151},
				End:      hcl.Pos{Line: 14, Column: 19, Byte: 169},
			},
		},
		{
			Label:         "null_ns",
			SourceAddrRaw: "./xyz/../abc",
			SourceAddr:    addrs.ModuleSourceLocal("./abc"),
			SourceAddrRange: hcl.Range{
				Filename: "symbols.tf",
				Start:    hcl.Pos{Line: 20, Column: 14, Byte: 248},
				End:      hcl.Pos{Line: 20, Column: 28, Byte: 262},
			},
			DeclRange: hcl.Range{
				Filename: "symbols.tf",
				Start:    hcl.Pos{Line: 19, Column: 1, Byte: 215},
				End:      hcl.Pos{Line: 19, Column: 18, Byte: 232},
			},
		},
		{
			Label:         "foo2",
			SourceAddrRaw: "./foo",
			SourceAddr:    addrs.ModuleSourceLocal("./foo"),
			SourceAddrRange: hcl.Range{
				Filename: "symbols.tf",
				Start:    hcl.Pos{Line: 25, Column: 14, Byte: 317},
				End:      hcl.Pos{Line: 25, Column: 21, Byte: 324},
			},
			DeclRange: hcl.Range{
				Filename: "symbols.tf",
				Start:    hcl.Pos{Line: 24, Column: 1, Byte: 287},
				End:      hcl.Pos{Line: 24, Column: 15, Byte: 301},
			},
		},
	}

	cmpOpts := cmp.Options{
		ctydebug.CmpOptions,
	}
	if diff := cmp.Diff(wantSymbols, gotSymbols, cmpOpts); diff != "" {
		t.Error("wrong result:\n" + diff)
	}
}

func TestInvalidSymbolsModules(t *testing.T) {
	cases := []struct {
		dir       string
		wantDiags []string
	}{
		{
			"testdata/invalid-modules/duplicate-symbols",
			[]string{
				filepath.FromSlash("testdata/invalid-modules/duplicate-symbols/b.tf") +
					`:1,1-17: Duplicate symbols block; A symbols block named "dejavu" was already defined at ` +
					filepath.FromSlash("testdata/invalid-modules/duplicate-symbols/a.tf") +
					`:1,1-17. Symbols blocks must have unique labels.`,
			},
		},
		{
			"testdata/invalid-modules/override-nonexist-symbols",
			[]string{
				filepath.FromSlash("testdata/invalid-modules/override-nonexist-symbols/override.tf") +
					`:1,1-20: Missing symbols block to override; There is no symbols block named "overrider". An override file can only override a symbols block that was defined in a primary configuration file.`,
			},
		},
		{
			"testdata/invalid-modules/missing-symbols-dir",
			[]string{
				filepath.FromSlash("testdata/invalid-modules/missing-symbols-dir/missing-symbols.tf") +
					`:2,12-23: Directory must exist; stat testdata/invalid-modules/missing-symbols-dir/missing: no such file or directory`,
			},
		},
		{
			"testdata/invalid-modules/bad-cty-syntax",
			[]string{
				filepath.FromSlash("testdata/invalid-modules/bad-cty-syntax/bad/bad.cty") +
					`:1,1-5: Expected function declaration; Top-level functy declarations must be functions (func name(...) { ... }).`,
			},
		},
		{
			"testdata/invalid-modules/bad-symbols-namespace",
			[]string{
				filepath.FromSlash("testdata/invalid-modules/bad-symbols-namespace/badsymns.tf") +
					`:1,1-19: Symbols namespace has no declarations; In testdata/invalid-modules/bad-symbols-namespace/badsymns, namespace "bad" exists but exports nothing: it contains only private (underscore-prefixed) declarations.`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.dir, func(t *testing.T) {
			_, diags := testModuleFromDir(tc.dir)
			assertExactDiagnostics(t, diags, tc.wantDiags)
		})
	}
}

func TestSourcesPresent(t *testing.T) {
	parser := NewParser(nil)
	_, diags := parser.LoadConfigDir("testdata/invalid-modules/bad-cty-syntax", RootModuleCallForTesting())

	assert.NotEmpty(t, diags)

	sources := parser.Sources()

	for _, d := range diags {
		assert.Contains(t, sources, d.Subject.Filename, "Files mentioned in diags must be present in sources")
	}
}

func TestSymbolsLibContents(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/valid-modules/symbols")

	assert.False(t, diags.HasErrors(), "load should have succeeded")

	lib := mod.SymbolLibrary
	assert.NotNil(t, lib)

	// Functions

	lf := lib.Functions
	assert.NotNil(t, lf)
	assert.Len(t, lf, 3)

	// foo and foo2 use the same sources, so the function should be exported under both names
	assert.Contains(t, lf, "symbols::foo::double")
	assert.Contains(t, lf, "symbols::foo2::double")

	f := lf["symbols::foo::double"]
	r, err := f.Call([]cty.Value{cty.NumberIntVal(2)})
	assert.NoError(t, err)
	assert.True(t, r.Equals(cty.NumberIntVal(4)).True())

	assert.Contains(t, lf, "symbols::with_ns::quadruple")

	// Values

	sv := lib.Values
	assert.NotNil(t, sv)

	// The expected value of the "symbols" object, represented as JSON
	expectedJson := []byte(`
	{
	  "foo" : {
	    "foo" : "foo"
	  },
	  "foo2" : {
	    "foo" : "foo"
	  },
	  "bar" : {
	    "bar" : "bar"
	  },
	  "with_ns" : {
	    "xyz" : "baz::qux::xyz"
	  },
	  "blank_ns" : {
	    "xyz" : "xyz"
	  },
	  "null_ns" : {
	    "abc" : "abc"
	  }
	}`)

	ctyType, err := ctyjson.ImpliedType(expectedJson)
	assert.NoError(t, err)
	expected, err := ctyjson.Unmarshal(expectedJson, ctyType)
	assert.NoError(t, err)

	if !assert.True(t, expected.Equals(sv).True(), "symbols object should match") {
		t.Log("expected", valueToHCL(expected))
		t.Log("got", valueToHCL(sv))
	}

	// Types

	tl := lib.TypeLookup
	assert.NotNil(t, tl)

	ctyType, ok := tl("null_ns", "counting_thingy")
	assert.True(t, ok, "counting_thing type should exist")
	assert.Equal(t, cty.Number, ctyType)

	ctyType, ok = tl("with_ns", "stringy_thingy")
	assert.True(t, ok, "stringy_thing type should exist")
	assert.Equal(t, cty.String, ctyType, "type should be string")

	ctyType, ok = tl("foo", "nonesuch")
	assert.False(t, ok, "type nonesuch should not exist")
}

func TestNoSymbolsModule(t *testing.T) {
	// Can be any module that doesn't use symbols
	mod, diags := testModuleFromDir("testdata/valid-modules/provider-meta")

	assert.False(t, diags.HasErrors(), "load should have succeeded")

	assert.Nil(t, mod.SymbolLibrary)
}

func valueToHCL(val cty.Value) string {
	return string(hclwrite.TokensForValue(val).Bytes())
}
