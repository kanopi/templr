package main

import (
	"flag"
	"fmt"
	"os"
)

// runLegacyMode implements backward compatibility for the old flag-based CLI.
// This function parses flags in the old style and routes to the appropriate command.
func runLegacyMode() {
	// Reset flag state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Define all the old flags
	in := flag.String("in", "", "Template file OR entry template name (if -dir is used). Omit for stdin (single-file).")
	out := flag.String("out", "", "Output file path (omit for stdout)")
	data := flag.String("data", "", "Path to base JSON or YAML data file")
	var files stringSlice
	flag.Var(&files, "f", "Additional values files (YAML/JSON). Repeatable.")
	var sets stringSlice
	flag.Var(&sets, "set", "key=value overrides. Repeatable. Supports dotted keys.")

	dir := flag.String("dir", "", "Directory containing *.tpl templates to parse together (multi-file mode)")
	walk := flag.Bool("walk", false, "Render all *.tpl under -src into -dst, mirroring paths and stripping .tpl")
	src := flag.String("src", "", "Templates root for -walk mode")
	dst := flag.String("dst", "", "Output root for -walk mode")

	ldelim := flag.String("ldelim", "{{", "Left delimiter")
	rdelim := flag.String("rdelim", "}}", "Right delimiter")
	strict := flag.Bool("strict", false, "Fail on missing keys")
	dryRun := flag.Bool("dry-run", false, "Preview which files would be rendered (no writes)")
	guard := flag.String("guard", "#templr generated", "Guard string required in existing files to allow overwrite")
	inject := flag.Bool("inject-guard", true, "Automatically insert the guard as a comment into written files (when supported)")
	helpers := flag.String("helpers", "_helpers*.tpl", "Glob pattern of helper templates to load (single-file mode). Set empty to skip.")
	var extraExts stringSlice
	flag.Var(&extraExts, "ext", "Additional template file extensions to treat as templates (e.g., md, txt). Repeatable; do not include the leading dot.")

	showVersion := flag.Bool("version", false, "Print version and exit")
	defaultMissing := flag.String("default-missing", "<no value>", "String to render when a variable/key is missing (works with missingkey=default)")
	noColor := flag.Bool("no-color", false, "Disable colored output (useful for CI/non-ANSI terminals)")

	flag.Parse()

	if *showVersion {
		fmt.Println(getVersion())
		return
	}

	// Build shared options
	shared := SharedOptions{
		Data:           *data,
		Files:          files,
		Sets:           sets,
		Strict:         *strict,
		DryRun:         *dryRun,
		Guard:          *guard,
		InjectGuard:    *inject,
		DefaultMissing: *defaultMissing,
		NoColor:        *noColor,
		Ldelim:         *ldelim,
		Rdelim:         *rdelim,
		ExtraExts:      extraExts,
	}

	// Route to appropriate mode
	var err error

	if *walk {
		// Walk mode
		opts := WalkOptions{
			Shared: shared,
			Src:    *src,
			Dst:    *dst,
		}
		err = RunWalkMode(opts)
	} else if *dir != "" {
		// Dir mode
		opts := DirOptions{
			Shared: shared,
			Dir:    *dir,
			In:     *in,
			Out:    *out,
		}
		err = RunDirMode(opts)
	} else {
		// Render mode (single-file)
		opts := RenderOptions{
			Shared:  shared,
			In:      *in,
			Out:     *out,
			Helpers: *helpers,
		}
		err = RunRenderMode(opts)
	}

	if err != nil {
		// Use legacy error formatting for backward compatibility
		errMsg := err.Error()
		if contains(errMsg, "requires") || contains(errMsg, "key=value") {
			errf(ExitGeneral, "args", "%v", err)
		} else if contains(errMsg, "parse") {
			errf(ExitTemplateError, "parse", "%v", err)
		} else if contains(errMsg, "render") || contains(errMsg, "template") || contains(errMsg, "executing") {
			errf(ExitTemplateError, "render", "%v", err)
		} else if contains(errMsg, "load data") || contains(errMsg, "data") {
			errf(ExitDataError, "data", "%v", err)
		} else if contains(errMsg, "guard") {
			errf(ExitGuardSkipped, "guard", "%v", err)
		} else if contains(errMsg, "helper") {
			errf(ExitTemplateError, "helpers", "%v", err)
		} else {
			errf(ExitGeneral, "error", "%v", err)
		}
	}
}
