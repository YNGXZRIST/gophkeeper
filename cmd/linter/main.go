// Command linter is a multichecker binary: it wires many
// golang.org/x/tools/go/analysis passes plus the project osexit analyzer
// and runs them over packages you name on the command line.
//
// # Multichecker
//
// multichecker.Main from golang.org/x/tools/go/analysis/multichecker is the
// driver entrypoint. It first validates the analyzer list (unique names,
// dependency graph). It registers standard flags (-fix, -c, analyzers on/off),
// parses flags and the remaining args as packages (or a single .cfg file for
// unitchecker). For normal package arguments it delegates to the shared checker,
// which loads typed syntax via go/packages, runs analyzers in an order that
// respects Requires edges, merges diagnostics, and exits non-zero if issues
// were found (unless only warnings, depending on flags). Run "linter help"
// for built-in help and per-analyzer flags (e.g. findcall uses -name).
package main

import (
	"gophkeeper/internal/errcheckanalyzer/osexit"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/gofix"
	"golang.org/x/tools/go/analysis/passes/hostport"
	"golang.org/x/tools/go/analysis/passes/httpmux"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inline"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/pkgfact"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"golang.org/x/tools/go/analysis/passes/waitgroup"
)

func main() {
	checks := getAnalyzers()
	multichecker.Main(
		checks...,
	)
}

func getAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		// custom: forbid os.Exit inside func main in package main (non-test .go files)
		osexit.Analyzer,
		// check for append calls with no values to append (no-op, often a bug)
		appends.Analyzer,
		// report mismatches between assembly files and Go declarations
		asmdecl.Analyzer,
		// check for useless assignments (e.g. x = x)
		assign.Analyzer,
		// check for common mistakes using the sync/atomic package
		atomic.Analyzer,
		// check for common mistakes involving boolean operators
		bools.Analyzer,
		// build SSA-form IR for passes that need it (e.g. nilness); infrastructure
		buildssa.Analyzer,
		// check //go:build and // +build directives
		buildtag.Analyzer,
		// detect some violations of the cgo pointer passing rules
		cgocall.Analyzer,
		// check for unkeyed composite literals
		composite.Analyzer,
		// check for locks erroneously passed by value
		copylock.Analyzer,
		// build a control-flow graph for dependent analyzers; infrastructure
		ctrlflow.Analyzer,
		// check for reflect.DeepEqual used with error values
		deepequalerrors.Analyzer,
		// report common mistakes in defer (e.g. defer f(time.Since(t)) evaluates Since early)
		defers.Analyzer,
		// check Go toolchain directives such as //go:debug
		directive.Analyzer,
		// report passing non-pointer or non-error values to errors.As
		errorsas.Analyzer,
		// find structs that would use less memory if fields were reordered
		fieldalignment.Analyzer,
		// find calls to a function whose name is set via -findcall.name (demo / tooling)
		findcall.Analyzer,
		// report assembly that clobbers the frame pointer before saving it
		framepointer.Analyzer,
		// validate //go:fix inline directives (pairs with inline analyzer)
		gofix.Analyzer,
		// check format of addresses passed to net.Dial
		hostport.Analyzer,
		// report Go 1.22+ http.ServeMux patterns when targeting older Go versions
		httpmux.Analyzer,
		// check for mistakes using HTTP responses (body lifecycle)
		httpresponse.Analyzer,
		// check for impossible interface-to-interface type assertions
		ifaceassert.Analyzer,
		// suggest inlining for functions/constants marked //go:fix inline
		inline.Analyzer,
		// provide ast.Inspector for other passes; infrastructure
		inspect.Analyzer,
		// check references to loop variables from nested functions (go/defer; pre-1.22 footguns)
		loopclosure.Analyzer,
		// check cancel func returned by context.WithCancel (and variants) is called
		lostcancel.Analyzer,
		// check for useless comparisons between functions and nil
		nilfunc.Analyzer,
		// nil pointer dereferences, impossible/tautological nil checks (uses SSA)
		nilness.Analyzer,
		// gather name/value pairs from const declarations named _key_; demo of package facts
		pkgfact.Analyzer,
		// check consistency of Printf format strings and arguments
		printf.Analyzer,
		// check for comparing reflect.Value with == or !=
		reflectvaluecompare.Analyzer,
		// check for possible unintended shadowing of variables
		shadow.Analyzer,
		// check for shifts that equal or exceed the width of the integer
		shift.Analyzer,
		// check for invalid channel arguments to signal.Notify
		sigchanyzer.Analyzer,
		// check correct use of log/slog API
		slog.Analyzer,
		// check the argument type of sort.Slice
		sortslice.Analyzer,
		// check signature of methods of well-known interfaces
		stdmethods.Analyzer,
		// report uses of standard library symbols newer than the module's Go version
		stdversion.Analyzer,
		// check for string(int) and related conversions
		stringintconv.Analyzer,
		// check that struct field tags conform to reflect.StructTag.Get
		structtag.Analyzer,
		// check for calling testing.T methods from the wrong goroutine
		testinggoroutine.Analyzer,
		// check for common mistaken usages of tests and examples
		tests.Analyzer,
		// check time.Format layout strings
		timeformat.Analyzer,
		// report passing non-pointer or non-interface values to unmarshal
		unmarshal.Analyzer,
		// check for unreachable code
		unreachable.Analyzer,
		// check for invalid conversions of uintptr to unsafe.Pointer
		unsafeptr.Analyzer,
		// check for unused results of calls to some functions
		unusedresult.Analyzer,
		// check for unused writes to struct/object fields
		unusedwrite.Analyzer,
		// report packages that use generics (informational)
		usesgenerics.Analyzer,
		// check for mistaken usage of sync.WaitGroup
		waitgroup.Analyzer,
	}
}
