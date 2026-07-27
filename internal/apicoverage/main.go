// Command apicoverage verifies that the Go binding accounts for every
// function export in the stable WasmEdge 0.17.1 C headers.
package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	defaultManifest = "docs/wasmedge-0.17.1-api.csv"

	classDirect   classification = "direct"
	classSemantic classification = "semantic"
	classOmitted  classification = "omitted"
	classProvider classification = "provider"
)

var (
	stableHeaders = []string{
		"wasmedge_basic.h",
		"wasmedge_value.h",
		"wasmedge_configure.h",
		"wasmedge_ast.h",
		"wasmedge_instance.h",
		"wasmedge_execution.h",
		"wasmedge_vm.h",
		"wasmedge_compiler.h",
		"wasmedge_plugin.h",
		"wasmedge_tools.h",
	}

	expectedClassCounts = map[classification]int{
		classDirect:   272,
		classSemantic: 16,
		classOmitted:  2,
		classProvider: 1,
	}

	symbolPattern = regexp.MustCompile(`\b(WasmEdge_[A-Za-z0-9_]+)\s*\(`)
	symbolToken   = regexp.MustCompile(`\bWasmEdge_[A-Za-z0-9_]+\b`)
	validSymbol   = regexp.MustCompile(`^WasmEdge_[A-Za-z0-9_]+$`)
)

type classification string

type export struct {
	header   string
	symbol   string
	provider bool
}

type manifestEntry struct {
	header string
	symbol string
	class  classification
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "apicoverage: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("apicoverage", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var (
		sdkPath     string
		repoPath    string
		manifest    string
		dumpExports bool
	)
	flags.StringVar(&sdkPath, "sdk", os.Getenv("WASMEDGE_SDK"),
		"WasmEdge SDK root or directory containing the stable headers (defaults to WASMEDGE_SDK)")
	flags.StringVar(&repoPath, "repo", ".", "repository root to inspect")
	flags.StringVar(&manifest, "manifest", defaultManifest,
		"manifest path, relative to -repo unless absolute")
	flags.BoolVar(&dumpExports, "dump-exports", false,
		"print the exports found in the SDK as CSV, without running the audit")
	flags.Usage = func() {
		_, _ = fmt.Fprintf(flags.Output(),
			"Usage: go run ./internal/apicoverage -sdk PATH [options]\n\n")
		_, _ = fmt.Fprintln(flags.Output(),
			"Verify the WasmEdge 0.17.1 stable C API coverage manifest and direct bindings.")
		_, _ = fmt.Fprintln(flags.Output(), "Options:")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	if strings.TrimSpace(sdkPath) == "" {
		return errors.New("missing SDK path: pass -sdk PATH or set WASMEDGE_SDK")
	}

	headerDir, err := findHeaderDir(sdkPath)
	if err != nil {
		return err
	}
	exports, err := readExports(headerDir)
	if err != nil {
		return err
	}
	if dumpExports {
		return writeExports(stdout, exports)
	}

	repoRoot, err := canonicalDirectory(repoPath, "repository root")
	if err != nil {
		return err
	}
	manifestPath, err := resolveManifestPath(repoRoot, manifest)
	if err != nil {
		return err
	}
	entries, err := readManifest(manifestPath)
	if err != nil {
		return err
	}
	if err := verifyAccounting(exports, entries); err != nil {
		return err
	}

	references, err := findNativeReferences(repoRoot)
	if err != nil {
		return err
	}
	var missing []string
	for _, entry := range entries {
		if entry.class == classDirect && !references[entry.symbol] {
			missing = append(missing, entry.symbol)
		}
	}
	if len(missing) != 0 {
		slices.Sort(missing)
		return fmt.Errorf(
			"%d direct API symbols have no C.<symbol> Go reference or native C/H reference:\n  - %s",
			len(missing), strings.Join(missing, "\n  - "),
		)
	}

	if _, err := fmt.Fprintf(
		stdout,
		"WasmEdge 0.17.1 API coverage OK: runtime=290 all=291 direct=272 semantic=16 omitted=2 provider=1\nheaders: %s\nmanifest: %s\n",
		headerDir, manifestPath,
	); err != nil {
		return fmt.Errorf("write coverage report: %w", err)
	}
	return nil
}

func findHeaderDir(sdkPath string) (string, error) {
	root, err := canonicalDirectory(sdkPath, "SDK path")
	if err != nil {
		return "", err
	}

	candidates := []string{
		filepath.Join(root, "include", "wasmedge"),
		filepath.Join(root, "wasmedge"),
		root,
	}
	for _, candidate := range candidates {
		resolved, err := filepath.EvalSymlinks(candidate)
		if err == nil && containsStableHeaders(resolved) {
			return resolved, nil
		}
	}
	return "", fmt.Errorf(
		"cannot find all ten stable WasmEdge headers below %q; checked %s",
		root, strings.Join(candidates, ", "),
	)
}

func canonicalDirectory(path, description string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", description, path, err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", description, absolute, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("inspect %s %q: %w", description, resolved, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s %q is not a directory", description, resolved)
	}
	return resolved, nil
}

func resolveManifestPath(repoRoot, manifest string) (string, error) {
	path := manifest
	relative := !filepath.IsAbs(path)
	if relative {
		path = filepath.Join(repoRoot, path)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve API manifest %q: %w", manifest, err)
	}
	if relative && !pathWithin(repoRoot, absolute) {
		return "", fmt.Errorf(
			"relative API manifest %q escapes repository root %q; pass an absolute path explicitly",
			manifest, repoRoot,
		)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve API manifest %q: %w", absolute, err)
	}
	if relative && !pathWithin(repoRoot, resolved) {
		return "", fmt.Errorf(
			"relative API manifest %q resolves outside repository root %q",
			manifest, repoRoot,
		)
	}
	return resolved, nil
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil || filepath.IsAbs(relative) {
		return false
	}
	return relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func containsStableHeaders(dir string) bool {
	for _, header := range stableHeaders {
		info, err := os.Stat(filepath.Join(dir, header))
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

func readExports(headerDir string) ([]export, error) {
	var exports []export
	seen := make(map[string]string)
	for _, header := range stableHeaders {
		found, err := readHeaderExports(filepath.Join(headerDir, header), header)
		if err != nil {
			return nil, err
		}
		for _, item := range found {
			if previous, ok := seen[item.symbol]; ok {
				return nil, fmt.Errorf(
					"SDK declares %s more than once: %s and %s",
					item.symbol, previous, header,
				)
			}
			seen[item.symbol] = header
			exports = append(exports, item)
		}
	}
	return exports, nil
}

func readHeaderExports(path, header string) ([]export, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read SDK header %q: %w", path, err)
	}
	clean := stripCLiterals(stripCComments(string(source)))

	const (
		runtimeMacro  = "WASMEDGE_CAPI_EXPORT"
		providerMacro = "WASMEDGE_CAPI_PLUGIN_EXPORT"
	)

	var (
		exports     []export
		decl        strings.Builder
		inDecl      bool
		inDirective bool
		provider    bool
		startLine   int
	)
	scanner := bufio.NewScanner(strings.NewReader(clean))
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if !inDecl {
			if inDirective {
				inDirective = continuesPreprocessorDirective(rawLine)
				continue
			}
			if strings.HasPrefix(line, "#") {
				inDirective = continuesPreprocessorDirective(rawLine)
				continue
			}
			if line == "" {
				continue
			}
			switch {
			case hasLeadingToken(line, providerMacro):
				inDecl = true
				provider = true
			case hasLeadingToken(line, runtimeMacro):
				inDecl = true
				provider = false
			default:
				continue
			}
			startLine = lineNumber
			decl.Reset()
		}

		if decl.Len() != 0 {
			decl.WriteByte(' ')
		}
		decl.WriteString(line)
		if !strings.ContainsRune(line, ';') {
			continue
		}

		text := decl.String()
		matches := symbolPattern.FindAllStringSubmatch(text, -1)
		if len(matches) == 0 {
			return nil, fmt.Errorf(
				"parse export declaration in %s:%d: no WasmEdge function name in %q",
				header, startLine, text,
			)
		}
		if len(matches) != 1 {
			var names []string
			for _, match := range matches {
				names = append(names, match[1])
			}
			return nil, fmt.Errorf(
				"parse export declaration in %s:%d: ambiguous WasmEdge function names %s in %q",
				header, startLine, strings.Join(names, ", "), text,
			)
		}
		exports = append(exports, export{
			header:   header,
			symbol:   matches[0][1],
			provider: provider,
		})
		inDecl = false
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan SDK header %q: %w", path, err)
	}
	if inDecl {
		return nil, fmt.Errorf(
			"parse export declaration in %s:%d: declaration has no terminating semicolon",
			header, startLine,
		)
	}
	return exports, nil
}

func continuesPreprocessorDirective(line string) bool {
	return strings.HasSuffix(strings.TrimSpace(line), `\`)
}

func hasLeadingToken(line, token string) bool {
	if !strings.HasPrefix(line, token) {
		return false
	}
	return len(line) == len(token) ||
		line[len(token)] == ' ' ||
		line[len(token)] == '\t'
}

func stripCComments(source string) string {
	const (
		code = iota
		slash
		lineComment
		blockComment
		blockCommentStar
		stringLiteral
		charLiteral
	)

	var result strings.Builder
	result.Grow(len(source))
	state := code
	escaped := false
	for _, character := range source {
		switch state {
		case code:
			switch character {
			case '/':
				state = slash
			case '"':
				result.WriteRune(character)
				state = stringLiteral
				escaped = false
			case '\'':
				result.WriteRune(character)
				state = charLiteral
				escaped = false
			default:
				result.WriteRune(character)
			}
		case slash:
			switch character {
			case '/':
				result.WriteString("  ")
				state = lineComment
			case '*':
				result.WriteString("  ")
				state = blockComment
			default:
				result.WriteByte('/')
				result.WriteRune(character)
				state = code
			}
		case lineComment:
			if character == '\n' {
				result.WriteByte('\n')
				state = code
			} else {
				result.WriteByte(' ')
			}
		case blockComment:
			switch character {
			case '*':
				result.WriteByte(' ')
				state = blockCommentStar
			case '\n':
				result.WriteByte('\n')
			default:
				result.WriteByte(' ')
			}
		case blockCommentStar:
			switch character {
			case '/':
				result.WriteByte(' ')
				state = code
			case '*':
				result.WriteByte(' ')
			case '\n':
				result.WriteByte('\n')
				state = blockComment
			default:
				result.WriteByte(' ')
				state = blockComment
			}
		case stringLiteral:
			result.WriteRune(character)
			if character == '"' && !escaped {
				state = code
			}
			if character == '\\' && !escaped {
				escaped = true
			} else {
				escaped = false
			}
		case charLiteral:
			result.WriteRune(character)
			if character == '\'' && !escaped {
				state = code
			}
			if character == '\\' && !escaped {
				escaped = true
			} else {
				escaped = false
			}
		}
	}
	if state == slash {
		result.WriteByte('/')
	}
	return result.String()
}

func stripCLiterals(source string) string {
	const (
		code = iota
		stringLiteral
		charLiteral
	)

	var result strings.Builder
	result.Grow(len(source))
	state := code
	escaped := false
	for _, character := range source {
		switch state {
		case code:
			switch character {
			case '"':
				result.WriteByte(' ')
				state = stringLiteral
				escaped = false
			case '\'':
				result.WriteByte(' ')
				state = charLiteral
				escaped = false
			default:
				result.WriteRune(character)
			}
		case stringLiteral:
			if character == '\n' {
				result.WriteByte('\n')
			} else {
				result.WriteByte(' ')
			}
			if character == '"' && !escaped {
				state = code
			}
			if character == '\\' && !escaped {
				escaped = true
			} else {
				escaped = false
			}
		case charLiteral:
			if character == '\n' {
				result.WriteByte('\n')
			} else {
				result.WriteByte(' ')
			}
			if character == '\'' && !escaped {
				state = code
			}
			if character == '\\' && !escaped {
				escaped = true
			} else {
				escaped = false
			}
		}
	}
	return result.String()
}

func writeExports(output io.Writer, exports []export) error {
	writer := csv.NewWriter(output)
	if err := writer.Write([]string{"header", "symbol", "export_kind"}); err != nil {
		return fmt.Errorf("write export CSV header: %w", err)
	}
	for _, item := range exports {
		kind := "runtime"
		if item.provider {
			kind = "provider"
		}
		if err := writer.Write([]string{item.header, item.symbol, kind}); err != nil {
			return fmt.Errorf("write export CSV: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("write export CSV: %w", err)
	}
	return nil
}

func readManifest(path string) ([]manifestEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open API manifest %q: %w", path, err)
	}

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	records, readErr := reader.ReadAll()
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("parse API manifest %q: %w", path, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close API manifest %q: %w", path, closeErr)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("API manifest %q is empty", path)
	}
	records[0][0] = strings.TrimPrefix(records[0][0], "\uFEFF")
	expectedHeader := []string{"header", "symbol", "classification"}
	if !slices.Equal(records[0], expectedHeader) {
		return nil, fmt.Errorf(
			"API manifest %q has header %q; want %q",
			path, records[0], expectedHeader,
		)
	}

	knownHeaders := make(map[string]bool, len(stableHeaders))
	for _, header := range stableHeaders {
		knownHeaders[header] = true
	}
	seen := make(map[string]int)
	entries := make([]manifestEntry, 0, len(records)-1)
	for index, record := range records[1:] {
		row := index + 2
		if len(record) != 3 {
			return nil, fmt.Errorf(
				"API manifest %q row %d has %d fields; want 3",
				path, row, len(record),
			)
		}
		entry := manifestEntry{
			header: strings.TrimSpace(record[0]),
			symbol: strings.TrimSpace(record[1]),
			class:  classification(strings.TrimSpace(record[2])),
		}
		if !knownHeaders[entry.header] {
			return nil, fmt.Errorf(
				"API manifest %q row %d has unknown header %q",
				path, row, entry.header,
			)
		}
		if !validSymbol.MatchString(entry.symbol) {
			return nil, fmt.Errorf(
				"API manifest %q row %d has invalid symbol %q",
				path, row, entry.symbol,
			)
		}
		if previous, ok := seen[entry.symbol]; ok {
			return nil, fmt.Errorf(
				"API manifest %q repeats %s on rows %d and %d",
				path, entry.symbol, previous, row,
			)
		}
		seen[entry.symbol] = row
		if _, ok := expectedClassCounts[entry.class]; !ok {
			return nil, fmt.Errorf(
				"API manifest %q row %d classifies %s as %q; want direct, semantic, omitted, or provider",
				path, row, entry.symbol, entry.class,
			)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func verifyAccounting(exports []export, entries []manifestEntry) error {
	manifestBySymbol := make(map[string]manifestEntry, len(entries))
	counts := make(map[classification]int)
	for _, entry := range entries {
		manifestBySymbol[entry.symbol] = entry
		counts[entry.class]++
	}

	exportBySymbol := make(map[string]export, len(exports))
	var problems []string
	for _, item := range exports {
		exportBySymbol[item.symbol] = item
		entry, ok := manifestBySymbol[item.symbol]
		if !ok {
			problems = append(problems,
				fmt.Sprintf("%s (%s) is missing from the manifest", item.symbol, item.header))
			continue
		}
		if entry.header != item.header {
			problems = append(problems,
				fmt.Sprintf("%s is declared in %s but the manifest says %s",
					item.symbol, item.header, entry.header))
		}
		if item.provider && entry.class != classProvider {
			problems = append(problems,
				fmt.Sprintf("%s uses WASMEDGE_CAPI_PLUGIN_EXPORT but is classified %s",
					item.symbol, entry.class))
		}
		if !item.provider && entry.class == classProvider {
			problems = append(problems,
				fmt.Sprintf("%s uses WASMEDGE_CAPI_EXPORT but is classified provider",
					item.symbol))
		}
	}
	for _, entry := range entries {
		if _, ok := exportBySymbol[entry.symbol]; !ok {
			problems = append(problems,
				fmt.Sprintf("%s (%s) is in the manifest but not the SDK headers",
					entry.symbol, entry.header))
		}
	}

	for _, class := range []classification{
		classDirect, classSemantic, classOmitted, classProvider,
	} {
		if counts[class] != expectedClassCounts[class] {
			problems = append(problems,
				fmt.Sprintf("%s count is %d; want %d",
					class, counts[class], expectedClassCounts[class]))
		}
	}
	providerCount := 0
	for _, item := range exports {
		if item.provider {
			providerCount++
		}
	}
	if len(exports)-providerCount != 290 {
		problems = append(problems,
			fmt.Sprintf("runtime export count is %d; want 290", len(exports)-providerCount))
	}
	if len(exports) != 291 {
		problems = append(problems,
			fmt.Sprintf("total export count is %d; want 291", len(exports)))
	}
	if providerCount != 1 {
		problems = append(problems,
			fmt.Sprintf("provider export count is %d; want 1", providerCount))
	}

	if len(problems) != 0 {
		slices.Sort(problems)
		return fmt.Errorf(
			"API manifest does not match the SDK:\n  - %s",
			strings.Join(problems, "\n  - "),
		)
	}
	return nil
}

func findNativeReferences(repoRoot string) (map[string]bool, error) {
	references := make(map[string]bool)
	headerNames := make(map[string]bool, len(stableHeaders))
	for _, header := range stableHeaders {
		headerNames[header] = true
	}

	err := filepath.WalkDir(repoRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "testdata", "vendor":
				if path != repoRoot {
					return filepath.SkipDir
				}
			}
			return nil
		}

		switch strings.ToLower(filepath.Ext(path)) {
		case ".go":
			if strings.HasSuffix(entry.Name(), "_test.go") {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read Go source %q: %w", path, err)
			}
			file, err := parser.ParseFile(
				token.NewFileSet(), path, source, parser.SkipObjectResolution,
			)
			if err != nil {
				return fmt.Errorf("parse Go source %q: %w", path, err)
			}
			ast.Inspect(file, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				packageName, ok := selector.X.(*ast.Ident)
				if ok && packageName.Name == "C" &&
					validSymbol.MatchString(selector.Sel.Name) {
					references[selector.Sel.Name] = true
				}
				return true
			})
		case ".c", ".h":
			// A vendored or copied SDK header must not satisfy the binding check.
			if headerNames[entry.Name()] {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read native source %q: %w", path, err)
			}
			code := stripCLiterals(stripCComments(string(source)))
			for _, symbol := range symbolToken.FindAllString(code, -1) {
				references[symbol] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan repository %q for native API references: %w", repoRoot, err)
	}
	return references, nil
}
