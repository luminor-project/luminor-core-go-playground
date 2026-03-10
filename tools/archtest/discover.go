package main

import (
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// discoverFacadeValueSymbols scans all facade packages in the given verticals
// and returns the set of exported value symbols (types, constants, sentinel errors).
// Functions (constructors) are excluded — they should only be called from
// composition roots (cmd/), which the archtest does not check.
//
// For type aliases (e.g. type Status = domain.Status), the underlying type
// is also added so that Go's transparent alias resolution doesn't cause
// false positives.
func discoverFacadeValueSymbols(p policy) (map[string]struct{}, error) {
	var patterns []string
	for _, v := range p.verticals {
		for _, pkg := range p.allowedCrossVerticalPkg {
			patterns = append(patterns, "./internal/"+v+"/"+pkg)
		}
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Dir: p.rootDir,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}

	symbols := make(map[string]struct{})
	for _, pkg := range pkgs {
		if pkg.Types == nil {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if !obj.Exported() {
				continue
			}
			if shouldAllowSymbol(obj) {
				symbols[pkg.PkgPath+"."+name] = struct{}{}
				addAliasUnderlying(symbols, obj)
			}
		}
	}
	return symbols, nil
}

// shouldAllowSymbol returns true for value-oriented symbols that may be
// freely shared across verticals: types, constants, and sentinel error vars.
// Functions (constructors, helpers) return false.
func shouldAllowSymbol(obj types.Object) bool {
	switch obj.(type) {
	case *types.TypeName:
		return true
	case *types.Const:
		return true
	case *types.Var:
		return isSentinelError(obj.Type())
	default:
		return false
	}
}

// isSentinelError checks whether a type implements the error interface.
func isSentinelError(t types.Type) bool {
	errorType := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	return types.Implements(t, errorType)
}

// addAliasUnderlying handles Go type aliases (e.g. type Status = domain.Status).
// When the facade re-exports a domain type via an alias, Go's type checker may
// resolve cross-vertical usage to the underlying domain type. We add the
// underlying type's symbol so that the check doesn't produce false positives.
func addAliasUnderlying(symbols map[string]struct{}, obj types.Object) {
	tn, ok := obj.(*types.TypeName)
	if !ok || !tn.IsAlias() {
		return
	}
	// For an alias, the underlying named type lives in the domain package.
	named, ok := tn.Type().(*types.Named)
	if !ok {
		return
	}
	underlyingObj := named.Obj()
	if underlyingObj == nil || underlyingObj.Pkg() == nil {
		return
	}
	// Only add if it points to an internal package (not stdlib).
	pkgPath := underlyingObj.Pkg().Path()
	if strings.Contains(pkgPath, "/internal/") {
		symbols[pkgPath+"."+underlyingObj.Name()] = struct{}{}
	}
}
