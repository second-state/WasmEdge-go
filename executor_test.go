package wasmedge

import (
	"errors"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func TestExecutorDerivedModulesLeaseImportedDependencies(t *testing.T) {
	tests := []struct {
		name            string
		instantiate     func(*Executor, *Store, *ASTModule) (*Module, error)
		closeStoreFirst bool
	}{
		{
			name: "instantiate",
			instantiate: func(exec *Executor, store *Store, ast *ASTModule) (*Module, error) {
				return exec.Instantiate(store, ast)
			},
		},
		{
			name: "register",
			instantiate: func(exec *Executor, store *Store, ast *ASTModule) (*Module, error) {
				return exec.Register(store, ast, "registered")
			},
			closeStoreFirst: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := NewLoader(nil)
			if err != nil {
				t.Fatal(err)
			}
			defer loader.Close()
			ast, err := loader.LoadBytes(testwasm.HostCallModule())
			if err != nil {
				t.Fatal(err)
			}
			defer ast.Close()
			validator, err := NewValidator(nil)
			if err != nil {
				t.Fatal(err)
			}
			defer validator.Close()
			if err := validator.Validate(ast); err != nil {
				t.Fatal(err)
			}

			exec, err := NewExecutor(nil)
			if err != nil {
				t.Fatal(err)
			}
			defer exec.Close()
			store := NewStore()
			env := NewModule("env")
			if err := env.AddFunction(
				"host_add",
				MustWrapFunc(func(a, b int32) int32 { return a + b }),
			); err != nil {
				t.Fatal(err)
			}
			if err := exec.RegisterImport(store, env); err != nil {
				t.Fatal(err)
			}

			derived, err := tt.instantiate(exec, store, ast)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = store.Close()
				_ = derived.Close()
				_ = env.Close()
			})

			if err := env.Close(); !errors.Is(err, ErrInUse) {
				t.Fatalf("close imported module: got %v, want ErrInUse", err)
			}

			if tt.closeStoreFirst {
				if err := store.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if err := derived.Close(); err != nil {
				t.Fatal(err)
			}
			if !tt.closeStoreFirst {
				if err := store.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if err := env.Close(); err != nil {
				t.Fatalf("close imported module after dependant: %v", err)
			}
		})
	}
}
