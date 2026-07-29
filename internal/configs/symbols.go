// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/opentofu/opentofu/internal/addrs"
	"github.com/opentofu/opentofu/internal/getmodules"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

type Symbols struct {
	// Label is the consumer-chosen local name (the block label), unique across the
	// config. It replaces the functy namespace in every output key: a function
	// bound under label "lib" is callable as symbols::lib::name regardless of which
	// functy namespace it came from.
	Label string

	// SourceAddrRaw is a module-style source address.
	// It names a *directory* — a functy "unit", all its `.cty` files
	// parsed together — not a single file.
	SourceAddrRaw string

	// The parsed source address, or nil if the raw source address was invalid,
	// in which case the diagnostics have already been emitted.
	// Prototype Phase 1: This is currently enforced to be a local path,
	// so it will also be nill with a diagnostic emitted if a remote source address is given.
	SourceAddr addrs.ModuleSource

	// The range of the source address expression in the configuration file.
	// On override, this is the range of the overriding block's source expression.
	SourceAddrRange hcl.Range

	// Namespace selects which functy namespace within the unit to bind; "" (omitted)
	// binds the global, unnamespaced surface. One block per namespace.
	Namespace    string
	NamespaceSet bool

	DeclRange hcl.Range
}

var symbolsBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "source",
			Required: true,
		},
		{
			Name: "namespace",
		},
	},
}

// The result of processing all the symbols blocks
type SymbolLibrary struct {
	// Functions keyed by symbols::label::name
	Functions map[string]function.Function

	// A Cty object value of shape {label: {constName: value}}
	// that will become the symbols variable in the evaluation context.
	Values cty.Value

	// Look up the type symbols::label::types(typeName), returning type and ok
	TypeLookup func(label, typeName string) (cty.Type, bool)
}

func decodeSymbolsBlock(block *hcl.Block, override bool) (*Symbols, hcl.Diagnostics) {
	s := &Symbols{
		Label:     block.Labels[0],
		DeclRange: block.DefRange,
	}

	schema := symbolsBlockSchema
	if override {
		schema = schemaForOverrides(schema)
	}

	content, diags := block.Body.Content(schema)

	if !hclsyntax.ValidIdentifier(s.Label) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid symbols block name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[0],
		})
	}

	var attrDiags hcl.Diagnostics

	if attr, exists := content.Attributes["source"]; exists {
		s.SourceAddrRaw, s.SourceAddr, attrDiags = decodeSymbolsSource(attr)
		s.SourceAddrRange = attr.Expr.Range()
		diags = append(diags, attrDiags...)
	}

	if attr, exists := content.Attributes["namespace"]; exists {
		s.Namespace, s.NamespaceSet, attrDiags = decodeSymbolsNamespace(attr)
		diags = append(diags, attrDiags...)
	}

	return s, diags
}

func decodeSymbolsSource(attr *hcl.Attribute) (string, addrs.ModuleSource, hcl.Diagnostics) {
	var source string
	var sourceParsed addrs.ModuleSource

	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() {
		return source, sourceParsed, diags
	}

	if val.IsNull() {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid source",
			Detail:   fmt.Sprintf("Null is not allowed for %s.", attr.Name),
			Subject:  attr.Expr.Range().Ptr(),
		})

		return source, sourceParsed, diags
	}

	var err error
	val, err = convert.Convert(val, cty.String)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid source",
			Detail:   fmt.Sprintf("A string value is required for %s.", attr.Name),
			Subject:  attr.Expr.Range().Ptr(),
		})
		return source, sourceParsed, diags
	}

	if !val.IsWhollyKnown() {
		// If there is a syntax error, HCL sets the value of the given attribute
		// to cty.DynamicVal. A diagnostic for the syntax error will already
		// bubble up, so we will move forward gracefully here.
		return source, sourceParsed, diags
	}

	source = val.AsString()

	if source == "" {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid source",
			Detail:   fmt.Sprintf("An empty string is not allowed for %s.", attr.Name),
			Subject:  attr.Expr.Range().Ptr(),
		})

		return source, sourceParsed, diags
	}

	sourceParsed, err = addrs.ParseModuleSource(source)
	var pathErr *getmodules.MaybeRelativePathErr

	if err == nil {
		if _, ok := sourceParsed.(addrs.ModuleSourceLocal); !ok {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid source address",
				Detail:   fmt.Sprintf("Only local source addresses are allowed for symbols, but %q is not a local address.", sourceParsed),
				Subject:  attr.Expr.Range().Ptr(),
			})

			sourceParsed = nil
		}
	} else {
		if errors.As(err, &pathErr) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid source address",
				Detail: fmt.Sprintf(
					"OpenTofu failed to determine your intended installation method for remote symbols package %q.\n\nIf you intended this as a local relative path, use \"./%s\" instead. The \"./\" prefix indicates that the address is a relative filesystem path.",
					pathErr.Addr, pathErr.Addr,
				),
				Subject: attr.Expr.Range().Ptr(),
			})
		} else {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid symbols source address",
				Detail:   fmt.Sprintf("Failed to parse symbols source address: %s.", err),
				Subject:  attr.Expr.Range().Ptr(),
			})
		}
	}

	return source, sourceParsed, diags
}

func decodeSymbolsNamespace(attr *hcl.Attribute) (string, bool, hcl.Diagnostics) {
	var ns string
	var nsSet bool

	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() {
		return ns, nsSet, diags
	}

	// If the namespace evaluates to null, treat it as if no namespace was specified
	if val.IsNull() {
		ns = ""
		nsSet = false
	} else {
		var err error
		val, err = convert.Convert(val, cty.String)
		if err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid namespace",
				Detail:   fmt.Sprintf("A string value is required for %s.", attr.Name),
				Subject:  attr.Expr.Range().Ptr(),
			})
			return ns, nsSet, diags
		}

		if !val.IsWhollyKnown() {
			// If there is a syntax error, HCL sets the value of the given attribute
			// to cty.DynamicVal. A diagnostic for the syntax error will already
			// bubble up, so we will move forward gracefully here.
			return ns, nsSet, diags
		}

		ns = val.AsString()
		nsSet = true
	}

	return ns, nsSet, diags
}

func (s *Symbols) merge(os *Symbols) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if os.SourceAddrRaw != "" {
		s.SourceAddrRaw = os.SourceAddrRaw
		s.SourceAddr = os.SourceAddr
		s.SourceAddrRange = os.SourceAddrRange
	}

	if os.NamespaceSet {
		s.Namespace = os.Namespace
		s.NamespaceSet = os.NamespaceSet
	}

	return diags
}
