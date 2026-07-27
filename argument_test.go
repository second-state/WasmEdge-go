package wasmedge

import (
	"errors"
	"testing"
)

func TestNilResourceArgumentsReturnErrInvalidArgument(t *testing.T) {
	loader, err := NewLoader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Close()
	validator, err := NewValidator(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer validator.Close()
	executor, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	store := NewStore()
	defer store.Close()
	module := NewModule("nil-arguments")
	defer module.Close()
	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()

	checks := []struct {
		name string
		call func() error
	}{
		{"loader serialize", func() error {
			_, err := loader.Serialize(nil)
			return err
		}},
		{"validator validate", func() error { return validator.Validate(nil) }},
		{"executor instantiate store", func() error {
			_, err := executor.Instantiate(nil, nil)
			return err
		}},
		{"executor instantiate AST", func() error {
			_, err := executor.Instantiate(store, nil)
			return err
		}},
		{"executor register", func() error {
			_, err := executor.Register(store, nil, "nil")
			return err
		}},
		{"executor register import", func() error {
			return executor.RegisterImport(store, nil)
		}},
		{"executor register alias", func() error {
			return executor.RegisterImportWithAlias(store, nil, "nil")
		}},
		{"executor invoke", func() error {
			_, err := executor.Invoke(nil)
			return err
		}},
		{"executor invoke context", func() error {
			//nolint:staticcheck // Deliberately verifies nil-context rejection.
			_, err := executor.InvokeContext(nil, nil)
			return err
		}},
		{"module add function", func() error { return module.AddFunction("nil", nil) }},
		{"module add table", func() error { return module.AddTable("nil", nil) }},
		{"module add memory", func() error { return module.AddMemory("nil", nil) }},
		{"module add global", func() error { return module.AddGlobal("nil", nil) }},
		{"VM load", func() error { return vm.Load(nil) }},
		{"VM register AST", func() error { return vm.RegisterModule("nil", nil) }},
		{"VM register import", func() error { return vm.RegisterImport(nil) }},
		{"VM register alias", func() error {
			return vm.RegisterImportWithAlias(nil, "nil")
		}},
		{"VM execute context", func() error {
			//nolint:staticcheck // Deliberately verifies nil-context rejection.
			_, err := vm.ExecuteContext(nil, "nil")
			return err
		}},
		{"VM registered context", func() error {
			//nolint:staticcheck // Deliberately verifies nil-context rejection.
			_, err := vm.ExecuteRegisteredContext(nil, "nil", "nil")
			return err
		}},
		{"VM run bytes context", func() error {
			//nolint:staticcheck // Deliberately verifies nil-context rejection.
			_, err := vm.RunBytesContext(nil, nil, "nil")
			return err
		}},
		{"VM run file context", func() error {
			//nolint:staticcheck // Deliberately verifies nil-context rejection.
			_, err := vm.RunFileContext(nil, "nil", "nil")
			return err
		}},
		{"VM run AST context", func() error {
			//nolint:staticcheck // Deliberately verifies nil-context rejection.
			_, err := vm.RunContext(nil, nil, "nil")
			return err
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("got %v, want ErrInvalidArgument", err)
			}
		})
	}
}
