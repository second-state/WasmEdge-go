package wasmedge

import "testing"

func TestPlugin(t *testing.T) {
	// No plug-in is guaranteed to be installed in the test environment; the
	// APIs must behave gracefully either way.
	LoadPluginDefaultPaths()
	plugins := ListPlugins()
	t.Logf("found plug-ins: %v", plugins)

	if FindPlugin("no_such_plugin") != nil {
		t.Error("FindPlugin on an unknown name should return nil")
	}
	for _, name := range plugins {
		plugin := FindPlugin(name)
		if plugin == nil {
			t.Errorf("FindPlugin(%q) should not return nil", name)
			continue
		}
		t.Logf("plug-in %q modules: %v", name, plugin.ListModule())
	}
}
