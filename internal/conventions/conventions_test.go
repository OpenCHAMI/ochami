// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package conventions

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// sourceFile is one parsed Go file. path is slash-separated and relative to
// the tree's root.
type sourceFile struct {
	path string
	fset *token.FileSet
	file *ast.File
	src  []byte
}

// testFunc is one top-level Test function and the text the checks read.
type testFunc struct {
	sf   *sourceFile
	decl *ast.FuncDecl
	name string
	tvar string // name of the *testing.T parameter
	doc  string // doc comment text, joined into one line
	body string // source text of the body
}

// violation is one broken rule at a position in the tree.
type violation struct {
	pos token.Position
	msg string
}

// tree is a parsed directory of Go files.
type tree struct {
	files []*sourceFile
	tests []*testFunc
}

var (
	moduleOnce sync.Once
	moduleTree *tree
	moduleRoot string
	moduleErr  error
)

// loadModule parses the module containing this package once per test binary.
func loadModule(t *testing.T) (*tree, string) {
	t.Helper()
	moduleOnce.Do(func() {
		moduleRoot, moduleErr = findModuleRoot()
		if moduleErr == nil {
			moduleTree, moduleErr = loadTree(moduleRoot)
		}
	})
	if moduleErr != nil {
		t.Fatalf("failed to load module sources: %v", moduleErr)
	}
	return moduleTree, moduleRoot
}

// findModuleRoot walks up from the working directory to the directory that
// holds go.mod.
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found above the working directory")
		}
		dir = parent
	}
}

// loadTree parses every .go file under root, skipping hidden directories,
// vendor, and testdata below root.
func loadTree(root string) (*tree, error) {
	tr := &tree{}
	err := filepath.WalkDir(root, func(file string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if file != root && (strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(file, ".go") {
			return nil
		}
		src, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filepath.ToSlash(rel), src, parser.ParseComments)
		if err != nil {
			return err
		}
		sf := &sourceFile{path: filepath.ToSlash(rel), fset: fset, file: f, src: src}
		tr.files = append(tr.files, sf)
		if strings.HasSuffix(file, "_test.go") {
			tr.tests = append(tr.tests, testFuncs(sf)...)
		}
		return nil
	})
	return tr, err
}

// testFuncs returns the top-level Test functions in sf.
func testFuncs(sf *sourceFile) []*testFunc {
	var tests []*testFunc
	for _, decl := range sf.file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv != nil || fd.Body == nil || !strings.HasPrefix(fd.Name.Name, "Test") {
			continue
		}
		params := fd.Type.Params.List
		if len(params) != 1 || len(params[0].Names) != 1 || !isTestingT(params[0].Type) {
			continue
		}
		tf := &testFunc{
			sf:   sf,
			decl: fd,
			name: fd.Name.Name,
			tvar: params[0].Names[0].Name,
			body: string(sf.src[sf.offset(fd.Body.Lbrace):sf.offset(fd.Body.Rbrace)]),
		}
		if fd.Doc != nil {
			tf.doc = strings.Join(strings.Fields(fd.Doc.Text()), " ")
		}
		tests = append(tests, tf)
	}
	return tests
}

// isTestingT reports whether expr is *testing.T.
func isTestingT(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "testing" && sel.Sel.Name == "T"
}

func (sf *sourceFile) offset(p token.Pos) int { return sf.fset.Position(p).Offset }

func (sf *sourceFile) at(p token.Pos, format string, args ...any) violation {
	return violation{pos: sf.fset.Position(p), msg: fmt.Sprintf(format, args...)}
}

func (tf *testFunc) at(format string, args ...any) violation {
	return tf.sf.at(tf.decl.Name.Pos(), format, args...)
}

// report fails t once per violation, followed by hint.
func report(t *testing.T, vs []violation, hint string) {
	t.Helper()
	sort.Slice(vs, func(i, j int) bool {
		if vs[i].pos.Filename != vs[j].pos.Filename {
			return vs[i].pos.Filename < vs[j].pos.Filename
		}
		return vs[i].pos.Line < vs[j].pos.Line
	})
	for _, v := range vs {
		t.Errorf("%s:%d: %s", v.pos.Filename, v.pos.Line, v.msg)
	}
	if len(vs) > 0 {
		t.Log(hint)
	}
}

var testNamePattern = regexp.MustCompile(`^Test[A-Z][A-Za-z0-9]*(_[A-Za-z][A-Za-z0-9]*){0,2}$`)

// checkTestNames flags Test functions not named TestSubject_Case. A subject
// may name a method (TestType_Method_Case).
func checkTestNames(tr *tree) []violation {
	var vs []violation
	for _, tf := range tr.tests {
		if !testNamePattern.MatchString(tf.name) {
			vs = append(vs, tf.at("%s is not named TestSubject_Case", tf.name))
		}
	}
	return vs
}

// checkBareSubjects flags a TestSubject with no _Case when the same package
// also has a TestSubject_Case.
func checkBareSubjects(tr *tree) []violation {
	cased := map[string]bool{}
	for _, tf := range tr.tests {
		if subject, _, ok := strings.Cut(tf.name, "_"); ok {
			cased[path.Dir(tf.sf.path)+":"+subject] = true
		}
	}
	var vs []violation
	for _, tf := range tr.tests {
		if !strings.Contains(tf.name, "_") && cased[path.Dir(tf.sf.path)+":"+tf.name] {
			vs = append(vs, tf.at("%s has no _Case, but its package has other %s_ tests", tf.name, tf.name))
		}
	}
	return vs
}

var genericSuffix = regexp.MustCompile(`_(more|edge|extra|coverage|regressions?|branches)_test\.go$`)

// checkTestFileSuffixes flags test files with a generic suffix instead of
// _test.go, _errors_test.go, or _boundary_test.go.
func checkTestFileSuffixes(tr *tree) []violation {
	var vs []violation
	for _, sf := range tr.files {
		if m := genericSuffix.FindStringSubmatch(sf.path); m != nil {
			vs = append(vs, sf.at(sf.file.Package, "test file suffix _%s_test.go is generic", m[1]))
		}
	}
	return vs
}

// checkDocsStartWithName flags Test functions whose doc comment is missing
// or doesn't start with the function's name.
func checkDocsStartWithName(tr *tree) []violation {
	var vs []violation
	for _, tf := range tr.tests {
		if first, _, _ := strings.Cut(tf.doc, " "); first != tf.name {
			vs = append(vs, tf.at("%s's doc comment doesn't start with its name", tf.name))
		}
	}
	return vs
}

var (
	coverageJargon = regexp.MustCompile(`(?i)\barms?\b|\bexercis(e|es|ed|ing)\b|\bfires?\b|\bbranch(es)?\b|\bhandl(es|ing)\b|\bworks? correctly\b|\bgracefully\b|\bdoes the same\b`)
	testReference  = regexp.MustCompile(`\bTest[A-Z_]\w*`)
)

// checkDocWording flags Test doc comments that describe coverage mechanics
// instead of behavior, or that refer to other tests.
func checkDocWording(tr *tree) []violation {
	var vs []violation
	for _, tf := range tr.tests {
		if m := coverageJargon.FindString(tf.doc); m != "" {
			vs = append(vs, tf.at("%s's doc says %q; state the behavior it verifies", tf.name, m))
		}
		for _, ref := range testReference.FindAllString(tf.doc, -1) {
			if ref != tf.name {
				vs = append(vs, tf.at("%s's doc refers to %s; describe this test on its own", tf.name, ref))
			}
		}
	}
	return vs
}

var (
	methodClaim = regexp.MustCompile(`\b(?:issues|sends|requests|verifies)\s+(?:a single |one |an? |the )?(GET|POST|PUT|PATCH|DELETE)\b(?:\s+(?:to\s+)?(/[\w./{}-]*[\w}]))?`)
	pathParam   = regexp.MustCompile(`\{[^}]*\}`)
)

// checkClaimedMethods flags Test docs that say the code sends an HTTP method
// (and path) the body never checks.
func checkClaimedMethods(tr *tree) []violation {
	var vs []violation
	for _, tf := range tr.tests {
		for _, m := range methodClaim.FindAllStringSubmatch(tf.doc, -1) {
			method, route := m[1], m[2]
			constant := "Method" + method[:1] + strings.ToLower(method[1:])
			// A string literal that starts with the method ("GET" or
			// "GET /path") or the http.MethodX constant counts.
			if !strings.Contains(tf.body, `"`+method) && !strings.Contains(tf.body, constant) {
				vs = append(vs, tf.at("%s's doc claims a %s request, but the test never checks the method", tf.name, method))
			}
			if segment := longestSegment(route); segment != "" && !strings.Contains(tf.body, segment) {
				vs = append(vs, tf.at("%s's doc claims %s %s, but the test never checks the path", tf.name, method, route))
			}
		}
	}
	return vs
}

// longestSegment returns the longest literal segment of a route, ignoring
// {parameters}.
func longestSegment(route string) string {
	longest := ""
	for _, s := range strings.Split(pathParam.ReplaceAllString(route, ""), "/") {
		if len(s) > len(longest) {
			longest = s
		}
	}
	return longest
}

// codeConstant matches an exit code constant's name, such as CodeUsage, and
// also the CodeName function, which isCodeConstant rules out.
var codeConstant = regexp.MustCompile(`\bCode[A-Z]\w*\b`)

// isCodeConstant reports whether name is an exit code constant's name.
func isCodeConstant(name string) bool {
	return name != "CodeName" && codeConstant.FindString(name) == name
}

// checkClaimedCodes flags Test docs that name exit codes when the body
// references none of them.
func checkClaimedCodes(tr *tree) []violation {
	var vs []violation
	for _, tf := range tr.tests {
		var named []string
		for _, c := range codeConstant.FindAllString(tf.doc, -1) {
			if isCodeConstant(c) {
				named = append(named, c)
			}
		}
		if len(named) == 0 {
			continue
		}
		found := false
		for _, c := range named {
			if regexp.MustCompile(`\b` + c + `\b`).MatchString(tf.body) {
				found = true
				break
			}
		}
		if !found {
			vs = append(vs, tf.at("%s's doc names %s, but the test checks none of them", tf.name, strings.Join(named, ", ")))
		}
	}
	return vs
}

var assertMethods = map[string]bool{
	"Error": true, "Errorf": true, "Fatal": true, "Fatalf": true, "Fail": true, "FailNow": true,
	"Run": true, "Skip": true, "Skipf": true, "SkipNow": true,
}

// checkAsserts flags Test functions that can't fail: they never report an
// error, run a subtest, skip, or pass t to a helper.
func checkAsserts(tr *tree) []violation {
	var vs []violation
	for _, tf := range tr.tests {
		asserts := false
		ast.Inspect(tf.decl.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || asserts {
				return !asserts
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && isIdent(sel.X, tf.tvar) && assertMethods[sel.Sel.Name] {
				asserts = true
			}
			for _, arg := range call.Args {
				if isIdent(arg, tf.tvar) {
					asserts = true
				}
			}
			return true
		})
		if !asserts {
			vs = append(vs, tf.at("%s never reports a failure", tf.name))
		}
	}
	return vs
}

func isIdent(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}

// codeRef returns the name of the exit code constant expr refers to
// (cli.CodeX, or CodeX inside package cli), or "".
func codeRef(expr ast.Expr) string {
	var name string
	switch e := expr.(type) {
	case *ast.Ident:
		name = e.Name
	case *ast.SelectorExpr:
		if isIdent(e.X, "cli") {
			name = e.Sel.Name
		}
	}
	if name != "" && isCodeConstant(name) {
		return name
	}
	return ""
}

// isCodeNameCall reports whether expr calls CodeName or cli.CodeName.
func isCodeNameCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name == "CodeName"
	case *ast.SelectorExpr:
		return fn.Sel.Name == "CodeName"
	}
	return false
}

// checkCodeNameMessages flags Errorf and Fatalf calls in tests that print an
// exit code constant with %d but don't also print its CodeName.
func checkCodeNameMessages(tr *tree) []violation {
	var vs []violation
	for _, sf := range tr.files {
		if !strings.HasSuffix(sf.path, "_test.go") {
			continue
		}
		ast.Inspect(sf.file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "Errorf" && sel.Sel.Name != "Fatalf") {
				return true
			}
			format, ok := call.Args[0].(*ast.BasicLit)
			if !ok || !strings.Contains(format.Value, "%d") {
				return true
			}
			code, named := "", false
			for _, arg := range call.Args[1:] {
				if isCodeNameCall(arg) {
					named = true
				} else if c := codeRef(arg); c != "" {
					code = c
				}
			}
			if code != "" && !named {
				vs = append(vs, sf.at(call.Pos(), "message prints %s with %%d but not CodeName(%s)", code, code))
			}
			return true
		})
	}
	return vs
}

// isExitCodeExpr reports whether expr is an ExitCode(...) call or an
// exitCode/ExitCode field or variable.
func isExitCodeExpr(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.CallExpr:
		switch fn := e.Fun.(type) {
		case *ast.Ident:
			return fn.Name == "ExitCode"
		case *ast.SelectorExpr:
			return fn.Sel.Name == "ExitCode"
		}
	case *ast.SelectorExpr:
		return e.Sel.Name == "exitCode" || e.Sel.Name == "ExitCode"
	case *ast.Ident:
		return e.Name == "exitCode"
	}
	return false
}

// checkExitCodeLiterals flags tests that compare an exit code with an
// integer literal instead of a named constant.
func checkExitCodeLiterals(tr *tree) []violation {
	var vs []violation
	for _, sf := range tr.files {
		if !strings.HasSuffix(sf.path, "_test.go") {
			continue
		}
		ast.Inspect(sf.file, func(n ast.Node) bool {
			bin, ok := n.(*ast.BinaryExpr)
			if !ok || (bin.Op != token.EQL && bin.Op != token.NEQ) {
				return true
			}
			for _, pair := range [][2]ast.Expr{{bin.X, bin.Y}, {bin.Y, bin.X}} {
				lit, ok := pair[1].(*ast.BasicLit)
				if ok && lit.Kind == token.INT && isExitCodeExpr(pair[0]) {
					vs = append(vs, sf.at(bin.Pos(), "exit code compared with %s; use a named cli.Code constant", lit.Value))
				}
			}
			return true
		})
	}
	return vs
}

var majorVersion = regexp.MustCompile(`^v\d+$`)

// checkHidesKoanf flags exported declarations in the Go files directly under
// dir whose types mention a koanf package.
func checkHidesKoanf(tr *tree, dir string) []violation {
	var vs []violation
	for _, sf := range tr.files {
		if path.Dir(sf.path) != dir || strings.HasSuffix(sf.path, "_test.go") {
			continue
		}
		koanf := map[string]bool{}
		for _, imp := range sf.file.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil || !strings.Contains(p, "/koanf/") {
				continue
			}
			name := path.Base(p)
			if majorVersion.MatchString(name) {
				name = path.Base(path.Dir(p))
			}
			if imp.Name != nil {
				name = imp.Name.Name
			}
			koanf[name] = true
		}
		if len(koanf) == 0 {
			continue
		}
		mentions := func(n ast.Node) bool {
			found := false
			ast.Inspect(n, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok {
					if id, ok := sel.X.(*ast.Ident); ok && koanf[id.Name] {
						found = true
					}
				}
				return !found
			})
			return found
		}
		flag := func(name *ast.Ident, n ast.Node) {
			if n != nil && mentions(n) {
				vs = append(vs, sf.at(name.Pos(), "exported %s exposes koanf in its type", name.Name))
			}
		}
		for _, decl := range sf.file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name.IsExported() && (d.Recv == nil || receiverExported(d.Recv)) {
					flag(d.Name, d.Type)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							flagExportedType(s, flag)
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if name.IsExported() {
								flag(name, s.Type)
							}
						}
					}
				}
			}
		}
	}
	return vs
}

// receiverExported reports whether a method's receiver type is exported.
func receiverExported(recv *ast.FieldList) bool {
	expr := recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if idx, ok := expr.(*ast.IndexExpr); ok {
		expr = idx.X
	}
	id, ok := expr.(*ast.Ident)
	return ok && id.IsExported()
}

// flagExportedType passes an exported type's exported surface to flag: the
// exported fields of a struct, the methods of an interface, or the whole
// type otherwise.
func flagExportedType(s *ast.TypeSpec, flag func(*ast.Ident, ast.Node)) {
	var fields *ast.FieldList
	switch t := s.Type.(type) {
	case *ast.StructType:
		fields = t.Fields
	case *ast.InterfaceType:
		fields = t.Methods
	default:
		flag(s.Name, s.Type)
		return
	}
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			flag(s.Name, field.Type) // embedded
		}
		for _, name := range field.Names {
			if name.IsExported() {
				flag(name, field.Type)
			}
		}
	}
}

var exitStatusRow = regexp.MustCompile(`\|\s+\*(\d+)\*\s*\n:\s+_(\w+)_`)

// checkExitStatusTable flags differences between the code/name rows of a
// man page's EXIT STATUS table and the exit code contract.
func checkExitStatusTable(file, man string) []violation {
	_, section, ok := strings.Cut(man, "# EXIT STATUS")
	if !ok {
		return []violation{{pos: token.Position{Filename: file}, msg: "no EXIT STATUS section"}}
	}
	section, _, _ = strings.Cut(section, "\n# ")
	documented := map[int]string{}
	for _, m := range exitStatusRow.FindAllStringSubmatch(section, -1) {
		code, err := strconv.Atoi(m[1])
		if err != nil {
			return []violation{{pos: token.Position{Filename: file}, msg: fmt.Sprintf("EXIT STATUS code %q: %v", m[1], err)}}
		}
		documented[code] = m[2]
	}
	var vs []violation
	for code := 0; code < 256; code++ {
		want := cli.CodeName(code)
		if strings.HasPrefix(want, "Code(") {
			want = ""
		}
		if got := documented[code]; got != want {
			vs = append(vs, violation{
				pos: token.Position{Filename: file},
				msg: fmt.Sprintf("EXIT STATUS lists code %d as %q, but the contract names it %q", code, got, want),
			})
		}
	}
	return vs
}

// sourceChecks are the checks that read only Go sources, by name.
var sourceChecks = map[string]func(*tree) []violation{
	"names":          checkTestNames,
	"bare subjects":  checkBareSubjects,
	"file suffixes":  checkTestFileSuffixes,
	"doc names":      checkDocsStartWithName,
	"doc wording":    checkDocWording,
	"claimed method": checkClaimedMethods,
	"claimed codes":  checkClaimedCodes,
	"asserts":        checkAsserts,
	"CodeName":       checkCodeNameMessages,
	"code literals":  checkExitCodeLiterals,
	"koanf":          func(tr *tree) []violation { return checkHidesKoanf(tr, "config") },
}

// TestTestNames_Format verifies every Test function's name has the form
// Test, Subject, then an optional _Case, where Subject may name a method as
// Type_Method.
func TestTestNames_Format(t *testing.T) {
	tr, _ := loadModule(t)
	report(t, checkTestNames(tr), "Name tests TestSubject_Case; see CONTRIBUTING.md.")
}

// TestTestNames_BareSubjectStandsAlone verifies a test named for its subject
// alone, with no _Case, is the only test for that subject in its package.
func TestTestNames_BareSubjectStandsAlone(t *testing.T) {
	tr, _ := loadModule(t)
	report(t, checkBareSubjects(tr), "Add a _Case that names the scenario.")
}

// TestTestFiles_NoGenericSuffix verifies no test file uses a generic suffix
// such as _more_test.go or _edge_test.go.
func TestTestFiles_NoGenericSuffix(t *testing.T) {
	tr, _ := loadModule(t)
	report(t, checkTestFileSuffixes(tr), "Use <subject>_test.go, _errors_test.go, or _boundary_test.go.")
}

// TestTestDocs_StartWithName verifies every Test function has a doc comment
// that starts with its name.
func TestTestDocs_StartWithName(t *testing.T) {
	tr, _ := loadModule(t)
	report(t, checkDocsStartWithName(tr), "Start the doc with the test's name and state the behavior it verifies.")
}

// TestTestDocs_DescribeBehavior verifies Test doc comments describe behavior
// rather than coverage mechanics and don't refer to other tests.
func TestTestDocs_DescribeBehavior(t *testing.T) {
	tr, _ := loadModule(t)
	report(t, checkDocWording(tr), "Describe the behavior the test verifies, without coverage jargon or references to other tests.")
}

// TestTestDocs_ClaimedMethodsChecked verifies a Test doc that says the code
// issues, sends, or requests an HTTP method (and path) belongs to a test whose
// body checks that method (and path).
func TestTestDocs_ClaimedMethodsChecked(t *testing.T) {
	tr, _ := loadModule(t)
	report(t, checkClaimedMethods(tr), "Assert r.Method and r.URL.Path in the handler, or reword the doc.")
}

// TestTestDocs_ClaimedCodesChecked verifies a Test doc that names exit code
// constants belongs to a test whose body references at least one of them.
func TestTestDocs_ClaimedCodesChecked(t *testing.T) {
	tr, _ := loadModule(t)
	report(t, checkClaimedCodes(tr), "Assert the exit code the doc names, or reword the doc.")
}

// TestTests_Assert verifies every Test function can fail: it reports an
// error, runs a subtest, skips, or passes t to a helper.
func TestTests_Assert(t *testing.T) {
	tr, _ := loadModule(t)
	report(t, checkAsserts(tr), "Assert the behavior the test's doc describes.")
}

// TestExitCodeMessages_IncludeCodeName verifies test failure messages that
// print an exit code constant with %d also print its cli.CodeName.
func TestExitCodeMessages_IncludeCodeName(t *testing.T) {
	tr, _ := loadModule(t)
	report(t, checkCodeNameMessages(tr), `Use "want %d (%s)", code, cli.CodeName(code).`)
}

// TestExitCodeAssertions_UseNamedConstants verifies tests compare exit codes
// with cli.Code constants rather than integer literals.
func TestExitCodeAssertions_UseNamedConstants(t *testing.T) {
	tr, _ := loadModule(t)
	report(t, checkExitCodeLiterals(tr), "Compare with cli.CodeSuccess, cli.CodeUsage, and so on.")
}

// TestExitStatusDoc_MatchesConstants verifies the EXIT STATUS table in
// ochami(1) lists exactly the exit codes cli.CodeName knows, under the same
// names.
func TestExitStatusDoc_MatchesConstants(t *testing.T) {
	_, root := loadModule(t)
	const file = "man/ochami.1.sc"
	man, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		t.Fatalf("failed to read %s: %v", file, err)
	}
	report(t, checkExitStatusTable(file, string(man)), "Keep ochami(1)'s EXIT STATUS table in step with internal/cli/errors.go.")
}

// TestConfigAPI_HidesKoanf verifies no exported function, method, type,
// field, or variable in pkg/config mentions a koanf type.
func TestConfigAPI_HidesKoanf(t *testing.T) {
	tr, _ := loadModule(t)
	report(t, checkHidesKoanf(tr, "pkg/config"), "Keep koanf behind config.Effective; see pkg/config/doc.go.")
}

// TestChecks_FlagFixtures verifies every check reports the deliberate
// violations in testdata/violations, so a check that matches nothing fails
// here rather than passing silently.
func TestChecks_FlagFixtures(t *testing.T) {
	tr, err := loadTree(filepath.Join("testdata", "violations"))
	if err != nil {
		t.Fatalf("failed to load fixtures: %v", err)
	}
	for name, check := range sourceChecks {
		if len(check(tr)) == 0 {
			t.Errorf("%s check reported nothing for testdata/violations", name)
		}
	}
	man := "# EXIT STATUS\n\n|  *0*\n:  _CodeSuccess_\n|  *1*\n:  _CodeWrong_\n"
	if len(checkExitStatusTable("fixture", man)) == 0 {
		t.Error("EXIT STATUS check reported nothing for a table with a misnamed and missing codes")
	}
}
