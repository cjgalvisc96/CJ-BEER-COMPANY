// Architecture fitness functions — the Go rendition of the book's
// NetArchTest examples (Chapter 6, "Testing and stabilizing"): automated
// tests that fail when a module grows a dependency it must not have, so
// the modular structure survives as the system evolves.
package tests

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const modulePath = "github.com/cjgalvisc96/cj-beer-company"

// importsOf returns package-dir → imported paths for every non-test Go
// file under root.
func importsOf(t *testing.T, root string) map[string][]string {
	t.Helper()
	imports := make(map[string][]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			imports[path] = append(imports[path], strings.Trim(imported.Path.Value, `"`))
		}
		return nil
	})
	require.NoError(t, err)
	return imports
}

func assertNoDependencyOn(t *testing.T, root string, forbidden []string) {
	t.Helper()
	for file, imported := range importsOf(t, root) {
		for _, importPath := range imported {
			for _, banned := range forbidden {
				assert.False(t, strings.HasPrefix(importPath, banned),
					"%s depends on %s — forbidden for this module", file, importPath)
			}
		}
	}
}

// TestShouldSalesArchitectureBeCompliant mirrors
// Should_SalesArchitecture_BeCompliant: the Sales module must not depend
// on any Warehouses project.
func TestShouldSalesArchitectureBeCompliant(t *testing.T) {
	assertNoDependencyOn(t, "../internal/sales", []string{
		modulePath + "/internal/warehouses",
	})
}

// TestShouldWarehousesArchitectureBeCompliant: the Warehouses module must
// not depend on any Sales project.
func TestShouldWarehousesArchitectureBeCompliant(t *testing.T) {
	assertNoDependencyOn(t, "../internal/warehouses", []string{
		modulePath + "/internal/sales",
	})
}

// TestShouldBrewUpArchitectureBeCompliant mirrors
// Should_BrewUpArchitecture_BeCompliant: the REST layer relies solely on
// each module's facade — never on Domain, SharedKernel, or ReadModel
// internals.
func TestShouldBrewUpArchitectureBeCompliant(t *testing.T) {
	assertNoDependencyOn(t, "../internal/rest", []string{
		modulePath + "/internal/sales/domain",
		modulePath + "/internal/sales/sharedkernel",
		modulePath + "/internal/sales/readmodel",
		modulePath + "/internal/warehouses/domain",
		modulePath + "/internal/warehouses/sharedkernel",
		modulePath + "/internal/warehouses/readmodel",
	})
}

// TestMufloneStaysGeneric: the framework must know nothing about the
// business modules.
func TestMufloneStaysGeneric(t *testing.T) {
	assertNoDependencyOn(t, "../internal/muflone", []string{
		modulePath + "/internal/sales",
		modulePath + "/internal/warehouses",
		modulePath + "/internal/rest",
	})
}
