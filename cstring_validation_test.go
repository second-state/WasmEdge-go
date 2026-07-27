package wasmedge

import (
	"errors"
	"testing"
)

func TestCStringValidation(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		if err := validateCString("value", "plain"); err != nil {
			t.Fatalf("plain string rejected: %v", err)
		}
		if err := validateCString("value", "before\x00after"); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("embedded NUL error = %v, want ErrInvalidArgument", err)
		}
	})

	t.Run("slice index", func(t *testing.T) {
		err := validateCStringSlice("item", []string{"first", "bad\x00value"})
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("embedded NUL slice error = %v, want ErrInvalidArgument", err)
		}
	})
}

func TestPublicCStringAPIsRejectEmbeddedNUL(t *testing.T) {
	const bad = "before\x00after"

	loader, err := NewLoader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Close()
	if _, err := loader.LoadFile(bad); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("Loader.LoadFile error = %v, want ErrInvalidArgument", err)
	}

	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	generation := vm.active.value.Load()
	if err := vm.RegisterModuleFile("mod", bad); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("VM.RegisterModuleFile error = %v, want ErrInvalidArgument", err)
	}
	if err := vm.LoadFile(bad); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("VM.LoadFile error = %v, want ErrInvalidArgument", err)
	}
	if _, err := vm.RunFileAsync(bad, "main"); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("VM.RunFileAsync error = %v, want ErrInvalidArgument", err)
	}
	if got := vm.active.value.Load(); got != generation {
		t.Errorf("invalid path advanced VM generation from %d to %d", generation, got)
	}

	compiler, err := NewCompiler(nil)
	if err == nil {
		defer compiler.Close()
		if err := compiler.CompileFile(bad, "out"); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("Compiler.CompileFile input error = %v, want ErrInvalidArgument", err)
		}
		if err := compiler.CompileFile("in", bad); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("Compiler.CompileFile output error = %v, want ErrInvalidArgument", err)
		}
		if err := compiler.CompileBytes(nil, bad); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("Compiler.CompileBytes error = %v, want ErrInvalidArgument", err)
		}
	} else if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("NewCompiler error = %v", err)
	}

	if err := LoadPlugins(bad); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("LoadPlugins error = %v, want ErrInvalidArgument", err)
	}
	if err := InitWASINN([]string{bad}); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("InitWASINN error = %v, want ErrInvalidArgument", err)
	}
	if err := InitWasmEdgeProcess([]string{bad}, false); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("InitWasmEdgeProcess error = %v, want ErrInvalidArgument", err)
	}

	for name, cfg := range map[string]WASIConfig{
		"argument": {
			Args: []string{bad}, Stdio: InheritWASIStdio(),
		},
		"environment": {
			Envs: []string{bad}, Stdio: InheritWASIStdio(),
		},
		"preopen": {
			Preopens: []string{bad}, Stdio: InheritWASIStdio(),
		},
	} {
		t.Run("WASI "+name, func(t *testing.T) {
			if _, err := NewWASIModule(cfg); !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("NewWASIModule error = %v, want ErrInvalidArgument", err)
			}
		})
	}

	wasi, err := NewWASIModule(WASIConfig{Stdio: InheritWASIStdio()})
	if err != nil {
		t.Fatal(err)
	}
	defer wasi.Close()
	if err := wasi.InitWASI(WASIConfig{
		Args: []string{bad}, Stdio: InheritWASIStdio(),
	}); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("Module.InitWASI error = %v, want ErrInvalidArgument", err)
	}

	for name, driver := range map[string]func([]string) int{
		"compiler": DriverCompiler,
		"tool":     DriverTool,
		"unitool":  DriverUniTool,
	} {
		t.Run("driver "+name, func(t *testing.T) {
			if got := driver([]string{"wasmedge", bad}); got != driverInvalidArgumentExitCode {
				t.Errorf("invalid driver exit code = %d, want %d",
					got, driverInvalidArgumentExitCode)
			}
		})
	}
}
