package wasmedge

import (
	"reflect"
	"testing"
)

func TestPluginRegistryRoundTrip(t *testing.T) {
	LoadPluginsFromDefaultPaths()

	names := PluginNames()
	if len(names) == 0 {
		t.Skip("no WasmEdge plugins are installed")
	}

	for _, name := range names {
		plugin, ok := FindPlugin(name)
		if !ok {
			t.Fatalf("FindPlugin(%q) did not find a listed plugin", name)
		}
		if got := plugin.Name(); got != name {
			t.Errorf("FindPlugin(%q).Name() = %q", name, got)
		}
	}

	name := names[0]
	if _, ok := FindPlugin("wasi_logging"); ok {
		name = "wasi_logging"
	}
	plugin, _ := FindPlugin(name)
	first := plugin.ModuleNames()
	second := plugin.ModuleNames()
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("%s ModuleNames changed between calls: %q then %q", name, first, second)
	}

	if name == "wasi_logging" {
		const moduleName = "wasi:logging/logging"
		if !containsString(first, moduleName) {
			t.Fatalf("%s ModuleNames = %q; want %q", name, first, moduleName)
		}
	}
}

func TestPluginNilSafety(t *testing.T) {
	var nilPlugin *Plugin
	if got := nilPlugin.Name(); got != "" {
		t.Fatalf("nil Plugin Name = %q; want empty", got)
	}
	if got := nilPlugin.ModuleNames(); got != nil {
		t.Fatalf("nil Plugin ModuleNames = %q; want nil", got)
	}
	if module, err := nilPlugin.CreateModule("module"); module != nil || err == nil {
		t.Fatalf("nil Plugin CreateModule = (%v, %v); want (nil, error)", module, err)
	}

	zeroPlugin := new(Plugin)
	if got := zeroPlugin.ModuleNames(); got != nil {
		t.Fatalf("zero Plugin ModuleNames = %q; want nil", got)
	}
}

func TestInitWASINNEmptyPreloads(t *testing.T) {
	LoadPluginsFromDefaultPaths()
	if err := InitWASINN(nil); err != nil {
		t.Fatal(err)
	}
	if err := InitWASINN([]string{}); err != nil {
		t.Fatal(err)
	}
}

func containsString(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
