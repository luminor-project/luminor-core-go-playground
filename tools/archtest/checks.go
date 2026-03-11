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
				if !slices.Contains(p.allowedCrossVerticalPkg, toSubpkg) {
					violations = append(violations, fmt.Sprintf("%s imports %s (cross-vertical imports allowed only via %v packages)", relPath(p.rootDir, path), imp, p.allowedCrossVerticalPkg))
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

// forbiddenDomainImports lists import path prefixes that domain packages must never use.
// Domain packages should be pure business logic with zero infrastructure coupling.
var forbiddenDomainImports = []string{
	modulePath + "/internal/platform/",
	"database/sql",
	"net/http",
	"github.com/jackc/pgx",
}

func checkDomainPurity(p policy) ([]string, error) {
	var violations []string
	for _, vertical := range p.verticals {
		domainPath := filepath.Join(p.rootDir, "internal", vertical, "domain")
		err := filepath.Walk(domainPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return filepath.SkipDir
				}
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			imports, err := getImports(path)
			if err != nil {
				return nil
			}
			for _, imp := range imports {
				for _, forbidden := range forbiddenDomainImports {
					if strings.HasPrefix(imp, forbidden) {
						violations = append(violations, fmt.Sprintf(
							"%s imports %s (domain packages must not import infrastructure)",
							relPath(p.rootDir, path), imp))
					}
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
func checkNoExportedFacadeInterfaces(p policy) ([]string, error) {
	facadeOnly := make(map[string]bool)
	for _, v := range p.facadeOnlyVerticals {
		facadeOnly[v] = true
	}

	var violations []string
	for _, vertical := range p.verticals {
		if facadeOnly[vertical] {
			continue
		}
		facadeDir := filepath.Join(p.rootDir, "internal", vertical, "facade")
		err := filepath.Walk(facadeDir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return filepath.SkipDir
				}
				return walkErr
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return nil
			}
			for _, decl := range f.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}
				for _, spec := range genDecl.Specs {
					ts := spec.(*ast.TypeSpec)
					if _, isIface := ts.Type.(*ast.InterfaceType); isIface && ts.Name.IsExported() {
						violations = append(violations, fmt.Sprintf(
							"%s exports interface %s (interfaces should be defined by consumers, not in facade packages)",
							relPath(p.rootDir, path), ts.Name.Name))
					}
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

func checkVerticalSubpackages(p policy) ([]string, error) {
	allowed := make(map[string]bool)
	for _, s := range p.allowedVerticalSubpkgs {
		allowed[s] = true
	}

	var violations []string
	for _, vertical := range p.verticals {
		verticalDir := filepath.Join(p.rootDir, "internal", vertical)
		entries, err := os.ReadDir(verticalDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if !allowed[entry.Name()] {
				violations = append(violations, fmt.Sprintf(
					"internal/%s/%s is not a recognized vertical subpackage (allowed: %v)",
					vertical, entry.Name(), p.allowedVerticalSubpkgs))
			}
		}
	}
	return violations, nil
}

func checkNoUnknownVerticals(p policy) ([]string, error) {
	internalDir := filepath.Join(p.rootDir, "internal")
	entries, err := os.ReadDir(internalDir)
	if err != nil {
		return nil, fmt.Errorf("read internal/: %w", err)
	}
	known := make(map[string]bool)
	for _, v := range p.verticals {
		known[v] = true
	}
	for _, s := range p.sharedPackages {
		known[s] = true
	}

	var violations []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !known[entry.Name()] {
			violations = append(violations, fmt.Sprintf(
				"internal/%s is not declared in archtest policy (add to verticals or sharedPackages)",
				entry.Name()))
		}
	}
	return violations, nil
}

func checkEventStoreImmutability(p policy) ([]string, error) {
	storeDir := filepath.Join(p.rootDir, "internal", "platform", "eventstore")
	var violations []string
	err := filepath.Walk(storeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path) //nolint:gosec // path is constructed from trusted rootDir, not user input
		if err != nil {
			return nil
		}
		for i, line := range strings.Split(string(content), "\n") {
			upper := strings.ToUpper(line)
			if strings.Contains(upper, "UPDATE ") || strings.Contains(upper, "DELETE ") {
				// Exclude Go comments that discuss the rule itself.
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") {
					continue
				}
				violations = append(violations, fmt.Sprintf(
					"%s:%d contains mutation SQL (event store must be append-only): %s",
					relPath(p.rootDir, path), i+1, strings.TrimSpace(line)))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return violations, nil
}

func checkNoDirectTimeNow(p policy) ([]string, error) {
	businessDirs := []string{"domain", "facade", "infra", "subscriber"}
	var violations []string
	for _, vertical := range p.verticals {
		for _, subpkg := range businessDirs {
			dir := filepath.Join(p.rootDir, "internal", vertical, subpkg)
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					if os.IsNotExist(err) {
						return filepath.SkipDir
					}
					return err
				}
				if info.IsDir() || !strings.HasSuffix(path, ".go") {
					return nil
				}
				if strings.HasSuffix(path, "_test.go") {
					return nil
				}
				vs, err := checkFileForTimeNow(p.rootDir, path)
				if err != nil {
					return nil
				}
				violations = append(violations, vs...)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return violations, nil
}

// checkFileForTimeNow uses AST analysis to detect any reference to time.Now,
// regardless of import alias (e.g. import t "time" → t.Now()).
func checkFileForTimeNow(rootDir, path string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}

	// Find the local name for the "time" import (usually "time", but could be aliased).
	var timeAlias string
	for _, imp := range f.Imports {
		impPath := strings.Trim(imp.Path.Value, `"`)
		if impPath != "time" {
			continue
		}
		if imp.Name != nil {
			timeAlias = imp.Name.Name
		} else {
			timeAlias = "time"
		}
		break
	}
	if timeAlias == "" {
		return nil, nil // file doesn't import "time" at all
	}

	var violations []string
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == timeAlias && sel.Sel.Name == "Now" {
			pos := fset.Position(sel.Pos())
			violations = append(violations, fmt.Sprintf(
				"%s:%d references time.Now (inject a Clock instead)",
				relPath(rootDir, path), pos.Line))
		}
		return true
	})
	return violations, nil
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
