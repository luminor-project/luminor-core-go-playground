package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

func checkImportBoundaries(p policy) ([]string, error) {
	var violations []string
	for _, fromVertical := range p.verticals {
		verticalPath := filepath.Join(p.rootDir, "internal", fromVertical)
		err := filepath.Walk(verticalPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_templ.go") {
				return nil
			}
			imports, err := getImports(path)
			if err != nil {
				return nil
			}
			for _, imp := range imports {
				toVertical, toSubpkg, ok := internalVerticalPathParts(p, imp)
				if !ok || toVertical == fromVertical {
					continue
				}
				if slices.Contains(p.allowedCrossVerticalPkg, toSubpkg) {
					continue
				}
				if slices.Contains(p.forbiddenCrossSubpkgs, toSubpkg) {
					violations = append(violations, fmt.Sprintf("%s imports %s (cross-vertical imports allowed only via facade packages)", relPath(p.rootDir, path), imp))
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return violations, nil
}

//nolint:cyclop // Rule checks are intentionally explicit to keep boundary policy logic straightforward.
func checkTypeBoundaries(p policy) ([]string, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedModule,
	}
	cfg.Dir = p.rootDir
	pkgs, err := packages.Load(cfg, "./internal/...")
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, nil
	}
	if errs := packages.PrintErrors(pkgs); errs > 0 {
		return nil, fmt.Errorf("package load errors: %d", errs)
	}

	var violations []string
	for _, pkg := range pkgs {
		fromVertical, ok := pkgVertical(p, pkg.PkgPath)
		if !ok {
			continue
		}
		for i, file := range pkg.Syntax {
			filePath := pkg.GoFiles[i]
			if strings.HasSuffix(filePath, "_test.go") || strings.HasSuffix(filePath, "_templ.go") {
				continue
			}
			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				obj := pkg.TypesInfo.Uses[sel.Sel]
				if obj == nil || obj.Pkg() == nil {
					return true
				}

				toVertical, toSubpkg, cross := internalVerticalPathParts(p, obj.Pkg().Path())
				if !cross || toVertical == fromVertical {
					return true
				}
				if toSubpkg == "" {
					return true
				}

				fullSymbol := obj.Pkg().Path() + "." + obj.Name()
				if _, allowed := p.allowedCrossSymbols[fullSymbol]; allowed {
					return true
				}
				if isAllowlistedValueFieldAccess(p, pkg.TypesInfo, sel) {
					return true
				}

				if isAllowedInterfaceSymbol(obj) {
					return true
				}
				if isInterfaceMethodCall(pkg.TypesInfo, sel) {
					return true
				}

				violations = append(violations, fmt.Sprintf("%s references foreign concrete symbol %s (%T). Prefer a consumer-local interface boundary.", relPath(p.rootDir, filePath), fullSymbol, obj))
				return true
			})
		}
	}
	return violations, nil
}

func getImports(filePath string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var imports []string
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, path)
	}
	return imports, nil
}

func pkgVertical(p policy, pkgPath string) (string, bool) {
	for _, vertical := range p.verticals {
		prefix := p.modulePath + "/internal/" + vertical
		if strings.HasPrefix(pkgPath, prefix) {
			return vertical, true
		}
	}
	return "", false
}

func internalVerticalPathParts(p policy, importPath string) (vertical string, subpkg string, ok bool) {
	prefix := p.modulePath + "/internal/"
	if !strings.HasPrefix(importPath, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(importPath, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	if !slices.Contains(p.verticals, parts[0]) {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func isAllowedInterfaceSymbol(obj types.Object) bool {
	typeName, ok := obj.(*types.TypeName)
	if !ok {
		return false
	}
	_, ok = typeName.Type().Underlying().(*types.Interface)
	return ok
}

func isInterfaceMethodCall(info *types.Info, sel *ast.SelectorExpr) bool {
	_, isFunc := info.Uses[sel.Sel].(*types.Func)
	if !isFunc {
		return false
	}
	recvType := info.Types[sel.X].Type
	if recvType == nil {
		return false
	}
	_, isIface := recvType.Underlying().(*types.Interface)
	return isIface
}

func isAllowlistedValueFieldAccess(p policy, info *types.Info, sel *ast.SelectorExpr) bool {
	xType := info.Types[sel.X].Type
	if xType == nil {
		return false
	}
	named, ok := xType.(*types.Named)
	if !ok {
		if ptr, isPtr := xType.(*types.Pointer); isPtr {
			named, ok = ptr.Elem().(*types.Named)
		}
	}
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}
	baseSymbol := named.Obj().Pkg().Path() + "." + named.Obj().Name()
	_, allowed := p.allowedCrossSymbols[baseSymbol]
	return allowed
}

func relPath(root, path string) string {
	if root == "" || root == "." {
		return path
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
