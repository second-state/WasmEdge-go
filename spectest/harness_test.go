// Package spectest runs the wast2json-converted WASM spec test suites,
// mirroring the WasmEdge C API spec tests (test/api and test/spec).
package spectest

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/second-state/WasmEdge-go/wasmedge"
)

type suiteDoc struct {
	Commands []command `json:"commands"`
}

type command struct {
	Type       string       `json:"type"`
	Line       uint64       `json:"line"`
	Name       string       `json:"name"`
	Filename   string       `json:"filename"`
	ModuleType string       `json:"module_type"`
	Text       string       `json:"text"`
	As         string       `json:"as"`
	Definition string       `json:"definition"`
	Action     *action      `json:"action"`
	Expected   []valueEntry `json:"expected"`
	Either     []valueEntry `json:"either"`
	// Thread command fields.
	Shared []struct {
		Module string `json:"module"`
	} `json:"shared"`
	Commands []command `json:"commands"`
	Thread   string    `json:"thread"`
}

type action struct {
	Type   string       `json:"type"`
	Module string       `json:"module"`
	Field  string       `json:"field"`
	Args   []valueEntry `json:"args"`
}

// valueEntry is one wast2json value: a string, a lane string array for
// v128, or absent for opaque reference expectations.
type valueEntry struct {
	Type     string          `json:"type"`
	LaneType string          `json:"lane_type"`
	Value    json.RawMessage `json:"value"`
}

func (v *valueEntry) str() (string, bool) {
	var s string
	if v.Value != nil && json.Unmarshal(v.Value, &s) == nil {
		return s, true
	}
	return "", false
}

func (v *valueEntry) lanes() ([]string, bool) {
	var lanes []string
	if v.Value != nil && json.Unmarshal(v.Value, &lanes) == nil {
		return lanes, true
	}
	return nil, false
}

func parseU64(s string) uint64 {
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("spectest: invalid numeric value %q", s))
	}
	return u
}

// buildArgs converts wast2json arguments into Go values, or returns a skip
// reason when an argument cannot be constructed through the public API.
func buildArgs(entries []valueEntry) ([]interface{}, string) {
	args := make([]interface{}, 0, len(entries))
	for i := range entries {
		entry := &entries[i]
		if lanes, ok := entry.lanes(); ok {
			args = append(args, lanesToV128(entry.LaneType, lanes))
			continue
		}
		s, _ := entry.str()
		switch entry.Type {
		case "i32":
			args = append(args, int32(uint32(parseU64(s))))
		case "i64":
			args = append(args, int64(parseU64(s)))
		case "f32":
			args = append(args, math.Float32frombits(uint32(parseU64(s))))
		case "f64":
			args = append(args, math.Float64frombits(parseU64(s)))
		case "externref":
			if s == "null" {
				args = append(args, wasmedge.NewNullExternRef())
			} else {
				// Track the original number as the referenced Go value so
				// that result comparison can check it after a round-trip.
				args = append(args, wasmedge.NewExternRef(uint32(parseU64(s))))
			}
		case "funcref":
			if s == "null" {
				args = append(args, wasmedge.NewFuncRef(nil))
			} else {
				return nil, "non-null funcref arguments are not supported"
			}
		default:
			// E.g. host "anyref" values cannot be constructed through the
			// WasmEdge C API.
			return nil, fmt.Sprintf("%s arguments are not supported", entry.Type)
		}
	}
	return args, ""
}

func lanesToV128(laneType string, lanes []string) wasmedge.V128 {
	var buf [16]byte
	switch laneType {
	case "i8":
		for i, lane := range lanes {
			buf[i] = uint8(parseU64(lane))
		}
	case "i16":
		for i, lane := range lanes {
			binary.LittleEndian.PutUint16(buf[i*2:], uint16(parseU64(lane)))
		}
	case "i32", "f32":
		for i, lane := range lanes {
			binary.LittleEndian.PutUint32(buf[i*4:], uint32(parseU64(lane)))
		}
	case "i64", "f64":
		for i, lane := range lanes {
			binary.LittleEndian.PutUint64(buf[i*8:], parseU64(lane))
		}
	default:
		panic("spectest: unknown v128 lane type " + laneType)
	}
	return wasmedge.NewV128(
		binary.LittleEndian.Uint64(buf[8:]), binary.LittleEndian.Uint64(buf[:8]))
}

// refMatches checks a reference expectation: "null" expects a null, empty
// matches any, and a number matches the payload (externref) or non-null.
func refMatches(valStr string, isNull bool, payload interface{}) bool {
	switch valStr {
	case "null":
		return isNull
	case "":
		return true
	default:
		if payload != nil {
			return payload == uint32(parseU64(valStr))
		}
		return !isNull
	}
}

// compareValue mirrors SpecTest::compare in WasmEdge test/spec, with the GC
// heap types matched as generic references (the C API hides heap types).
func compareValue(expected *valueEntry, got interface{}) bool {
	valStr, _ := expected.str()

	if strings.HasPrefix(expected.Type, "v128") || expected.LaneType != "" {
		lanes, ok := expected.lanes()
		if !ok {
			return false
		}
		v, ok := got.(wasmedge.V128)
		if !ok {
			return false
		}
		return v128LanesMatch(expected.LaneType, lanes, v)
	}

	if valStr != "" && strings.HasPrefix(valStr, "nan:") {
		switch expected.Type {
		case "f32":
			f, ok := got.(float32)
			return ok && math.IsNaN(float64(f))
		case "f64":
			f, ok := got.(float64)
			return ok && math.IsNaN(f)
		}
		return false
	}

	switch expected.Type {
	case "i32":
		g, ok := got.(int32)
		return ok && uint32(g) == uint32(parseU64(valStr))
	case "i64":
		g, ok := got.(int64)
		return ok && uint64(g) == parseU64(valStr)
	case "f32":
		// Compare the 32-bit pattern.
		g, ok := got.(float32)
		return ok && math.Float32bits(g) == uint32(parseU64(valStr))
	case "f64":
		// Compare the 64-bit pattern.
		g, ok := got.(float64)
		return ok && math.Float64bits(g) == parseU64(valStr)
	case "funcref", "nullfuncref":
		g, ok := got.(wasmedge.FuncRef)
		return ok && refMatches(valStr, g.IsNull(), nil)
	case "externref", "nullexternref":
		g, ok := got.(wasmedge.ExternRef)
		return ok && refMatches(valStr, g.IsNull(), g.GetRef())
	case "anyref", "eqref", "structref", "arrayref", "i31ref", "nullref",
		"exnref", "nullexnref":
		// Internal references. The C API cannot distinguish the heap types,
		// so accept any non-external reference with matching null-ness.
		switch g := got.(type) {
		case wasmedge.Ref:
			return refMatches(valStr, g.IsNull(), nil)
		case wasmedge.FuncRef:
			return refMatches(valStr, g.IsNull(), nil)
		default:
			return false
		}
	case "ref":
		// "ref" fits all reference types.
		switch g := got.(type) {
		case wasmedge.Ref:
			return refMatches(valStr, g.IsNull(), nil)
		case wasmedge.FuncRef:
			return refMatches(valStr, g.IsNull(), nil)
		case wasmedge.ExternRef:
			return refMatches(valStr, g.IsNull(), g.GetRef())
		default:
			return false
		}
	}
	return false
}

func v128LanesMatch(laneType string, lanes []string, v wasmedge.V128) bool {
	high, low := v.GetVal()
	var buf [16]byte
	binary.LittleEndian.PutUint64(buf[:8], low)
	binary.LittleEndian.PutUint64(buf[8:], high)

	switch laneType {
	case "i8":
		for i, lane := range lanes {
			if buf[i] != uint8(parseU64(lane)) {
				return false
			}
		}
	case "i16":
		for i, lane := range lanes {
			if binary.LittleEndian.Uint16(buf[i*2:]) != uint16(parseU64(lane)) {
				return false
			}
		}
	case "i32":
		for i, lane := range lanes {
			if binary.LittleEndian.Uint32(buf[i*4:]) != uint32(parseU64(lane)) {
				return false
			}
		}
	case "i64":
		for i, lane := range lanes {
			if binary.LittleEndian.Uint64(buf[i*8:]) != parseU64(lane) {
				return false
			}
		}
	case "f32":
		for i, lane := range lanes {
			bits := binary.LittleEndian.Uint32(buf[i*4:])
			if strings.HasPrefix(lane, "nan:") {
				if !math.IsNaN(float64(math.Float32frombits(bits))) {
					return false
				}
			} else if bits != uint32(parseU64(lane)) {
				return false
			}
		}
	case "f64":
		for i, lane := range lanes {
			bits := binary.LittleEndian.Uint64(buf[i*8:])
			if strings.HasPrefix(lane, "nan:") {
				if !math.IsNaN(math.Float64frombits(bits)) {
					return false
				}
			} else if bits != parseU64(lane) {
				return false
			}
		}
	default:
		return false
	}
	return true
}

func compareValues(expected []valueEntry, got []interface{}) bool {
	if len(expected) != len(got) {
		return false
	}
	for i := range expected {
		if !compareValue(&expected[i], got[i]) {
			return false
		}
	}
	return true
}

// messageMatches mirrors SpecTest::stringContains: the WasmEdge error
// message must be a prefix of the expected error text from the suite.
func messageMatches(expectedText string, err error) bool {
	return strings.HasPrefix(expectedText, err.Error())
}

// resolveRegister builds the alias map from the register commands: original
// module names (or anonymous module lines) to the registered names.
func resolveRegister(cmds []command) map[string]string {
	alias := map[string]string{}
	orgName := ""
	lastModLine := uint64(0)
	for i := range cmds {
		cmd := &cmds[i]
		switch cmd.Type {
		case "module":
			orgName = cmd.Name
			lastModLine = cmd.Line
		case "register":
			key := cmd.Name
			if key == "" {
				if orgName != "" {
					key = orgName
				} else {
					key = strconv.FormatUint(lastModLine, 10)
				}
			}
			if _, ok := alias[key]; !ok {
				alias[key] = cmd.As
			}
		}
	}
	return alias
}

// cmdProcessor executes one command array against a VM context; thread
// commands spawn a child processor, matching SpecTest::processCommands.
type cmdProcessor struct {
	t       *testing.T
	unit    *unitContext
	ctx     *specVM
	alias   map[string]string
	lastMod string
	threads map[string]chan struct{}
	astMap  map[string]*wasmedge.AST
}

// unitContext carries the per-unit information shared by all processors.
type unitContext struct {
	root   string
	folder folderConfig
	mode   specMode
	name   string
}

func (u *unitContext) filePath(filename string) string {
	return filepath.Join(u.root, u.folder.name, u.name, filename)
}

func runUnit(t *testing.T, root string, folder folderConfig, mode specMode, unitName string) {
	unit := &unitContext{root: root, folder: folder, mode: mode, name: unitName}

	raw, err := os.ReadFile(unit.filePath(unitName + ".json"))
	if err != nil {
		t.Fatalf("failed to read the test manifest: %v", err)
	}
	var doc suiteDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("failed to parse the test manifest: %v", err)
	}

	ctx := newSpecVM(t, folder, mode)
	defer ctx.release()

	proc := &cmdProcessor{
		t:       t,
		unit:    unit,
		ctx:     ctx,
		alias:   resolveRegister(doc.Commands),
		threads: map[string]chan struct{}{},
		astMap:  map[string]*wasmedge.AST{},
	}
	defer proc.cleanup()
	proc.runCommands(doc.Commands)
}

func (p *cmdProcessor) cleanup() {
	for _, ast := range p.astMap {
		ast.Release()
	}
}

func (p *cmdProcessor) runCommands(cmds []command) {
	for i := range cmds {
		p.runCommand(&cmds[i])
	}
	// Join any threads not explicitly waited on.
	for _, done := range p.threads {
		<-done
	}
}

func (p *cmdProcessor) fail(cmd *command, format string, args ...interface{}) {
	prefix := fmt.Sprintf("%s/%s:%d (%s): ", p.unit.folder.name, p.unit.name, cmd.Line, cmd.Type)
	p.t.Errorf(prefix+format, args...)
}

// moduleName resolves the module name an action refers to, honoring the
// register aliases and the last instantiated module.
func (p *cmdProcessor) moduleName(act *action) string {
	if act.Module != "" {
		if as, ok := p.alias[act.Module]; ok {
			return as
		}
		return act.Module
	}
	return p.lastMod
}

func (p *cmdProcessor) invoke(act *action) ([]interface{}, error, string) {
	args, skip := buildArgs(act.Args)
	if skip != "" {
		return nil, nil, skip
	}
	modName := p.moduleName(act)
	if modName != "" {
		rets, err := p.ctx.vm.ExecuteRegistered(modName, act.Field, args...)
		return rets, err, ""
	}
	rets, err := p.ctx.vm.Execute(act.Field, args...)
	return rets, err, ""
}

func (p *cmdProcessor) getGlobal(act *action) (interface{}, error) {
	var mod *wasmedge.Module
	if name := p.moduleName(act); name != "" {
		mod = p.ctx.vm.GetStore().FindModule(name)
	} else {
		mod = p.ctx.vm.GetActiveModule()
	}
	if mod == nil {
		return nil, fmt.Errorf("module instance not found")
	}
	glob := mod.FindGlobal(act.Field)
	if glob == nil {
		return nil, fmt.Errorf("global instance %q not found", act.Field)
	}
	return glob.GetValue(), nil
}

func (p *cmdProcessor) runCommand(cmd *command) {
	switch cmd.Type {
	case "module":
		if cmd.ModuleType == "text" {
			// WAT modules are not supported by WasmEdge.
			return
		}
		name := cmd.Name
		if name != "" {
			if as, ok := p.alias[name]; ok {
				name = as
			}
		} else if as, ok := p.alias[strconv.FormatUint(cmd.Line, 10)]; ok {
			name = as
		}
		p.lastMod = name

		path, err := p.ctx.prepare(p.unit.filePath(cmd.Filename))
		if err != nil {
			p.fail(cmd, "compilation failed: %v", err)
			return
		}
		if name != "" {
			if err := p.ctx.vm.RegisterWasmFile(name, path); err != nil {
				p.fail(cmd, "module registration failed: %v", err)
			}
		} else {
			if err := p.ctx.instantiate(path); err != nil {
				p.fail(cmd, "module instantiation failed: %v", err)
			}
		}

	case "module_definition":
		path, err := p.ctx.prepare(p.unit.filePath(cmd.Filename))
		if err != nil {
			p.fail(cmd, "compilation failed: %v", err)
			return
		}
		ast, err := p.ctx.vm.GetLoader().LoadFile(path)
		if err != nil {
			p.fail(cmd, "module definition loading failed: %v", err)
			return
		}
		if err := p.ctx.vm.GetValidator().Validate(ast); err != nil {
			ast.Release()
			p.fail(cmd, "module definition validation failed: %v", err)
			return
		}
		if cmd.Name != "" {
			if old, ok := p.astMap[cmd.Name]; ok {
				old.Release()
			}
			p.astMap[cmd.Name] = ast
		} else {
			ast.Release()
		}

	case "module_instance":
		ast, ok := p.astMap[cmd.Definition]
		if !ok {
			p.fail(cmd, "unknown module definition %q", cmd.Definition)
			return
		}
		name := cmd.Name
		if as, ok := p.alias[name]; ok {
			name = as
		}
		if err := p.ctx.vm.RegisterAST(name, ast); err != nil {
			p.fail(cmd, "module instantiation from definition failed: %v", err)
		}

	case "register":
		// Preprocessed into the alias map.

	case "action":
		if _, err, skip := p.invoke(cmd.Action); skip != "" {
			p.t.Logf("%s/%s:%d: skipped: %s", p.unit.folder.name, p.unit.name, cmd.Line, skip)
		} else if err != nil {
			p.fail(cmd, "action failed: %v", err)
		}

	case "assert_return":
		act := cmd.Action
		switch act.Type {
		case "invoke":
			rets, err, skip := p.invoke(act)
			if skip != "" {
				p.t.Logf("%s/%s:%d: skipped: %s", p.unit.folder.name, p.unit.name, cmd.Line, skip)
				return
			}
			if err != nil {
				p.fail(cmd, "invocation of %q failed: %v", act.Field, err)
				return
			}
			if cmd.Either != nil {
				for i := range cmd.Either {
					if compareValue(&cmd.Either[i], rets[0]) {
						return
					}
				}
				p.fail(cmd, "%q returned %v, want one of %s", act.Field, rets, entriesString(cmd.Either))
				return
			}
			if !compareValues(cmd.Expected, rets) {
				p.fail(cmd, "%q returned %v, want %s", act.Field, rets, entriesString(cmd.Expected))
			}
		case "get":
			got, err := p.getGlobal(act)
			if err != nil {
				p.fail(cmd, "get of %q failed: %v", act.Field, err)
				return
			}
			if !compareValue(&cmd.Expected[0], got) {
				p.fail(cmd, "global %q is %v, want %s", act.Field, got, entriesString(cmd.Expected))
			}
		default:
			p.fail(cmd, "unknown action type %q", act.Type)
		}

	case "assert_trap":
		_, err, skip := p.invoke(cmd.Action)
		if skip != "" {
			p.t.Logf("%s/%s:%d: skipped: %s", p.unit.folder.name, p.unit.name, cmd.Line, skip)
			return
		}
		if err == nil {
			p.fail(cmd, "invocation of %q should trap with %q", cmd.Action.Field, cmd.Text)
		} else if !messageMatches(cmd.Text, err) {
			p.fail(cmd, "trap message %q does not match %q", err.Error(), cmd.Text)
		}

	case "assert_exhaustion":
		// Not checked, matching the WasmEdge spec test behavior.

	case "assert_malformed":
		if cmd.ModuleType != "binary" {
			return
		}
		if err := p.ctx.load(p.unit.filePath(cmd.Filename)); err == nil {
			p.fail(cmd, "loading should fail with %q", cmd.Text)
		} else if !messageMatches(cmd.Text, err) {
			p.fail(cmd, "loading error %q does not match %q", err.Error(), cmd.Text)
		}

	case "assert_invalid":
		if cmd.ModuleType != "binary" {
			return
		}
		if err := p.ctx.validate(p.unit.filePath(cmd.Filename)); err == nil {
			p.fail(cmd, "validation should fail with %q", cmd.Text)
		} else if !messageMatches(cmd.Text, err) {
			p.fail(cmd, "validation error %q does not match %q", err.Error(), cmd.Text)
		}

	case "assert_unlinkable", "assert_uninstantiable":
		if err := p.ctx.instantiateFresh(p.unit.filePath(cmd.Filename)); err == nil {
			p.fail(cmd, "instantiation should fail with %q", cmd.Text)
		} else if !messageMatches(cmd.Text, err) {
			p.fail(cmd, "instantiation error %q does not match %q", err.Error(), cmd.Text)
		}

	case "assert_exception":
		if cmd.Action.Type != "invoke" {
			p.fail(cmd, "unknown action type %q", cmd.Action.Type)
			return
		}
		if _, err, skip := p.invoke(cmd.Action); skip != "" {
			p.t.Logf("%s/%s:%d: skipped: %s", p.unit.folder.name, p.unit.name, cmd.Line, skip)
		} else if err == nil {
			p.fail(cmd, "invocation of %q should throw an exception", cmd.Action.Field)
		}

	case "thread":
		// Determine the (parent name, alias name) pairs of shared modules:
		// the thread's own register commands define the alias names.
		sharedAlias := map[string]string{}
		for i := range cmd.Commands {
			sub := &cmd.Commands[i]
			if sub.Type == "register" && sub.Name != "" {
				sharedAlias[sub.Name] = sub.As
			}
		}

		child := &cmdProcessor{
			t:       p.t,
			unit:    p.unit,
			ctx:     newSpecVM(p.t, p.unit.folder, p.unit.mode),
			alias:   resolveRegister(cmd.Commands),
			threads: map[string]chan struct{}{},
			astMap:  map[string]*wasmedge.AST{},
		}
		parentStore := p.ctx.vm.GetStore()
		for _, shared := range cmd.Shared {
			parentName := shared.Module
			if as, ok := p.alias[parentName]; ok {
				parentName = as
			}
			aliasName := parentName
			if as, ok := sharedAlias[shared.Module]; ok {
				aliasName = as
			}
			if mod := parentStore.FindModule(parentName); mod != nil {
				if err := child.ctx.vm.RegisterModuleWithAlias(aliasName, mod); err != nil {
					p.fail(cmd, "sharing module %q failed: %v", parentName, err)
				}
			}
		}

		done := make(chan struct{})
		p.threads[cmd.Name] = done
		commands := cmd.Commands
		go func() {
			defer close(done)
			defer child.ctx.release()
			defer child.cleanup()
			child.runCommands(commands)
		}()

	case "wait":
		if done, ok := p.threads[cmd.Thread]; ok {
			<-done
			delete(p.threads, cmd.Thread)
		} else {
			p.fail(cmd, "wait for unknown thread %q", cmd.Thread)
		}

	default:
		p.fail(cmd, "unknown command type")
	}
}

func entriesString(entries []valueEntry) string {
	parts := make([]string, len(entries))
	for i := range entries {
		if s, ok := entries[i].str(); ok {
			parts[i] = entries[i].Type + ":" + s
		} else if lanes, ok := entries[i].lanes(); ok {
			parts[i] = entries[i].Type + entries[i].LaneType + ":" + strings.Join(lanes, " ")
		} else {
			parts[i] = entries[i].Type
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
