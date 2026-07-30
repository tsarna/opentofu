// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

const (
	IdentSymbols = "symbols"
)

func NewSymbolsAttr(name string) SymbolsAttr {
	return SymbolsAttr{Name: name}
}

// SymbolsAttr is the address of an attribute of the "symbols" object in
// the interpolation scope, like "symbols.mylib.myconst"
type SymbolsAttr struct {
	referenceable
	Name string
}

func (sa SymbolsAttr) String() string {
	return IdentSymbols + "." + sa.Name
}

func (sa SymbolsAttr) UniqueKey() UniqueKey {
	return sa // A SymbolsAttr is its own UniqueKey
}

func (sa SymbolsAttr) uniqueKeySigil() {}
