// Architecture fitness functions — the Go rendition of the book's
// NetArchTest examples (Chapter 6, "Testing and stabilizing"): automated
// tests that fail when a package grows a dependency it must not have, so
// the modular structure survives as the system evolves.
//
// Run on their own with `task check:architecture`.
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

// importsOf returns file → imported paths for every non-test Go file
// under root.
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
				assert.False(t, importPath == banned || strings.HasPrefix(importPath, banned+"/"),
					"%s depends on %s — forbidden for this package", file, importPath)
			}
		}
	}
}

// --- module isolation (the book's Should_*Architecture_BeCompliant) ---------

// TestShouldSalesArchitectureBeCompliant mirrors
// Should_SalesArchitecture_BeCompliant: the Sales module must not depend
// on any Warehouses project (and never on the composition root).
func TestShouldSalesArchitectureBeCompliant(t *testing.T) {
	assertNoDependencyOn(t, "../internal/sales", []string{
		modulePath + "/internal/warehouses",
		modulePath + "/internal/app",
		modulePath + "/internal/rest",
	})
}

// TestShouldWarehousesArchitectureBeCompliant: the Warehouses module must
// not depend on any Sales project.
func TestShouldWarehousesArchitectureBeCompliant(t *testing.T) {
	assertNoDependencyOn(t, "../internal/warehouses", []string{
		modulePath + "/internal/sales",
		modulePath + "/internal/app",
		modulePath + "/internal/rest",
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
		modulePath + "/internal/app",
		modulePath + "/internal/shared",
	})
}

// TestSharedCustomTypesStayLeaf: the shared custom types are a leaf — they
// depend on nothing inside the repo.
func TestSharedCustomTypesStayLeaf(t *testing.T) {
	assertNoDependencyOn(t, "../internal/shared", []string{
		modulePath + "/internal",
	})
}

// --- intra-module layering (the book's project split, per module) -----------

// The BrewUp project layout implies a direction between a module's parts:
//
//	sharedkernel  → (muflone, shared) only — it IS the published language
//	domain        → sharedkernel; never readmodel or the facade
//	readmodel     → sharedkernel (events to project); never domain
//	integration   → sharedkernel commands + bus; never domain or readmodel
//	sagas         → domain + sharedkernel (it drives saga aggregates and
//	                sends commands); never the read model
func TestModuleLayeringIsCompliant(t *testing.T) {
	for _, module := range []string{"sales", "warehouses"} {
		base := modulePath + "/internal/" + module

		assertNoDependencyOn(t, "../internal/"+module+"/sharedkernel", []string{
			base + "/domain",
			base + "/readmodel",
			base + "/integration",
			base + "/sagas",
		})
		assertNoDependencyOn(t, "../internal/"+module+"/domain", []string{
			base + "/readmodel",
			base + "/integration",
			base + "/sagas",
		})
		assertNoDependencyOn(t, "../internal/"+module+"/readmodel", []string{
			base + "/domain",
			base + "/integration",
			base + "/sagas",
		})
	}

	// sales/integration reacts by SENDING COMMANDS, never by touching the
	// write or read model directly.
	assertNoDependencyOn(t, "../internal/sales/integration", []string{
		modulePath + "/internal/sales/domain",
		modulePath + "/internal/sales/readmodel",
	})
	// The warehouse saga coordinates through its event-sourced saga
	// aggregate and commands; the read model is out of bounds.
	assertNoDependencyOn(t, "../internal/warehouses/sagas", []string{
		modulePath + "/internal/warehouses/readmodel",
	})
}
