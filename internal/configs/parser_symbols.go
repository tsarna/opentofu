// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/opentofu/opentofu/internal/addrs"
	"github.com/tsarna/functy"
	"github.com/tsarna/functy/symbols"
	"github.com/zclconf/go-cty/cty/function"
)

// Maximum number of execution steps in a functy function. Finite matters
// more than the value; converts a wedged library into a diagnostic;
// recursion/wall-clock limits are functy future work
const MaxSteps = 1_000_000

// Loads the .cty files from path for parsing. Returned filenames are keys in Sources()
func (p *Parser) LoadSymbolSources(root string, sourceRange hcl.Range) ([]functy.Source, hcl.Diagnostics) {
	fi, err := p.fs.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errDiag("Directory must exist", err, sourceRange)
		} else {
			return nil, errDiag("Error accessing directory", err, sourceRange)
		}
	}

	if !fi.IsDir() {
		return nil, errDiag("Source must be a directory",
			errors.New(root+" is not a directory"), sourceRange)
	}

	var diags hcl.Diagnostics
	var sources []functy.Source

	err = p.fs.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip dot-prefixed files and directories
		if path != root && strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, functy.Extension) {
			return nil
		}

		bytes, err := p.fs.ReadFile(path)

		if err == nil {
			sources = append(sources, functy.Source{Filename: path, Bytes: bytes})
			p.ForceFileSource(path, bytes)
		} else {
			diags = diags.Extend(errDiag("Error reading file", err, sourceRange))
			// fall through and return nil to allow accumulating additional errors
		}

		return nil
	})

	if err != nil {
		diags = diags.Extend(errDiag("Error from directory walking", err, sourceRange))
	}

	return sources, diags
}

// Return a map of the core functions to be made available to functy functions
// This merges OpenTofu's pure core plus functy's small standard library additions
func (p *Parser) FunctyCoreFunctions() map[string]function.Function {
	functions := make(map[string]function.Function)

	for name, f := range p.functions {
		functions[name] = f
	}

	for name, f := range functy.Stdlib() {
		functions[name] = f
	}

	return functions
}

func (p *Parser) LoadSymbolLibrary(module *Module) (*SymbolLibrary, hcl.Diagnostics) {
	if len(module.SymbolsBlocks) == 0 {
		return nil, nil
	}

	// Process symbols blocks in a deterministic order by sorting

	symbolsBlockLabels := make([]string, 0, len(module.SymbolsBlocks))
	for name := range module.SymbolsBlocks {
		symbolsBlockLabels = append(symbolsBlockLabels, name)
	}

	sort.Strings(symbolsBlockLabels)

	sourceRanges := make(map[string]hcl.Range)
	functyBlocks := make([]symbols.SymbolsBlock, 0, len(module.SymbolsBlocks))

	for _, label := range symbolsBlockLabels {
		block := module.SymbolsBlocks[label]

		if block.SourceAddr == nil {
			continue // failed, diagnostics already reported
		}

		functySymbolsBlock := symbols.SymbolsBlock{
			Label:     label,
			Source:    joinSymbolsSource(module, block.SourceAddr),
			Namespace: block.Namespace,
			DefRange:  block.DeclRange,
		}
		functyBlocks = append(functyBlocks, functySymbolsBlock)

		sourceRanges[functySymbolsBlock.Source] = block.SourceAddrRange
	}

	if len(functyBlocks) == 0 {
		return nil, nil
	}

	sourceLoader := func(source string) ([]functy.Source, hcl.Diagnostics) {
		sourceRange := sourceRanges[source]

		return p.LoadSymbolSources(source, sourceRange)
	}

	loaded, diags := symbols.NewBuilder().
		WithBaseFunctions(p.FunctyCoreFunctions()).
		WithBlocks(functyBlocks...).
		WithMaxSteps(MaxSteps).
		WithSourceLoader(sourceLoader).
		Build()

	return &SymbolLibrary{
		Functions:  loaded.Functions,
		Values:     loaded.Symbols,
		TypeLookup: loaded.Type,
	}, diags
}

func joinSymbolsSource(module *Module, source addrs.ModuleSource) string {
	// TODO: Phase 4, handle non-local modules
	return filepath.Join(module.SourceDir, source.String())
}

func errDiag(summary string, err error, rng hcl.Range) hcl.Diagnostics {
	return hcl.Diagnostics{{
		Severity: hcl.DiagError,
		Summary:  summary,
		Detail:   err.Error(),
		Subject:  &rng,
	}}
}
