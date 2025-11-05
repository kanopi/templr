package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"text/template/parse"
)

// LintOptions contains all configuration for lint mode
type LintOptions struct {
	Shared       SharedOptions
	In           string  // single file to lint
	Dir          string  // directory to lint
	Src          string  // source tree to walk and lint
	FailOnWarn   bool    // exit with error on warnings
	Format       string  // output format: text, json, github-actions
	NoUndefCheck bool    // skip undefined variable checking
	Config       *Config // configuration from file
}

// LintIssue represents a single linting issue
type LintIssue struct {
	Severity string // "error", "warn"
	Category string // "parse", "undefined", "function", "guard"
	File     string // file path
	Line     int    // line number (0 if unknown)
	Column   int    // column number (0 if unknown)
	Message  string // human-readable message
}

// LintResult contains the results of a lint operation
type LintResult struct {
	Issues []LintIssue
	Errors int
	Warns  int
}

// RunLintMode executes lint mode
func RunLintMode(opts LintOptions) error {
	result := &LintResult{
		Issues: []LintIssue{},
	}

	// Load data values if provided (for undefined variable checking)
	var values map[string]any
	if !opts.NoUndefCheck && opts.Shared.Data != "" {
		var err error
		values, err = buildValues(".", opts.Shared)
		if err != nil {
			return fmt.Errorf("load data: %w", err)
		}
	}

	// Check required variables if configured
	if opts.Config != nil && len(opts.Config.Lint.RequiredVars) > 0 && values != nil {
		checkRequiredVars(values, opts.Config.Lint.RequiredVars, result)
	}

	// Determine which mode to use
	if opts.In != "" {
		// Lint single file
		if err := lintSingleFile(opts.In, values, opts, result); err != nil {
			return err
		}
	} else if opts.Dir != "" {
		// Lint directory mode
		if err := lintDirectory(opts.Dir, values, opts, result); err != nil {
			return err
		}
	} else if opts.Src != "" {
		// Lint walk mode
		if err := lintWalk(opts.Src, values, opts, result); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("must specify -i, --dir, or --src")
	}

	// Report results
	printLintResults(result, opts)

	// Determine exit code
	if result.Errors > 0 {
		os.Exit(ExitLintError)
	}
	if result.Warns > 0 && opts.FailOnWarn {
		os.Exit(ExitLintWarn)
	}

	return nil
}

// lintSingleFile lints a single template file
func lintSingleFile(path string, values map[string]any, opts LintOptions, result *LintResult) error {
	// Check if file should be excluded
	if opts.Config != nil && shouldExcludeFile(path, opts.Config.Lint.Exclude) {
		return nil
	}

	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	// Create a new template with custom delimiters
	tpl := template.New(filepath.Base(path))
	tpl.Delims(opts.Shared.Ldelim, opts.Shared.Rdelim)
	tpl.Funcs(buildFuncMap(&tpl))

	// Try to parse the template
	_, err = tpl.Parse(string(content))
	if err != nil {
		// Parse error - add as lint issue
		issue := LintIssue{
			Severity: "error",
			Category: "parse",
			File:     path,
			Message:  err.Error(),
		}
		// Try to extract line number from error message
		issue.Line = extractLineNumber(err.Error())
		result.Issues = append(result.Issues, issue)
		result.Errors++
		return nil
	}

	// Check for disallowed functions
	if opts.Config != nil && len(opts.Config.Lint.DisallowFunctions) > 0 {
		checkDisallowedFunctions(tpl, path, opts.Config.Lint.DisallowFunctions, result)
	}

	// If we have values and undefined checking is enabled, check for undefined variables
	if !opts.NoUndefCheck && values != nil {
		checkUndefinedVariables(tpl, path, values, opts, result)
	}

	return nil
}

// lintDirectory lints all templates in a directory
func lintDirectory(dirPath string, values map[string]any, opts LintOptions, result *LintResult) error {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}

	// Find all template files
	pattern := filepath.Join(absDir, "*.tpl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob: %w", err)
	}

	// Add extra extensions
	for _, ext := range opts.Shared.ExtraExts {
		p := filepath.Join(absDir, "*."+ext)
		m, _ := filepath.Glob(p)
		matches = append(matches, m...)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no template files found in %s", dirPath)
	}

	// Parse all templates together (to support includes/defines)
	tpl := template.New("__root__")
	tpl.Delims(opts.Shared.Ldelim, opts.Shared.Rdelim)
	tpl.Funcs(buildFuncMap(&tpl))

	for _, path := range matches {
		content, err := os.ReadFile(path)
		if err != nil {
			result.Issues = append(result.Issues, LintIssue{
				Severity: "error",
				Category: "read",
				File:     path,
				Message:  err.Error(),
			})
			result.Errors++
			continue
		}

		_, err = tpl.New(filepath.Base(path)).Parse(string(content))
		if err != nil {
			issue := LintIssue{
				Severity: "error",
				Category: "parse",
				File:     path,
				Message:  err.Error(),
			}
			issue.Line = extractLineNumber(err.Error())
			result.Issues = append(result.Issues, issue)
			result.Errors++
		}
	}

	// Check for undefined variables in each template
	if !opts.NoUndefCheck && values != nil {
		for _, tmpl := range tpl.Templates() {
			if tmpl.Name() == "__root__" {
				continue
			}
			// Find the file path for this template
			var filePath string
			for _, path := range matches {
				if filepath.Base(path) == tmpl.Name() {
					filePath = path
					break
				}
			}
			checkUndefinedVariables(tmpl, filePath, values, opts, result)

			// Check for disallowed functions in each template
			if opts.Config != nil && len(opts.Config.Lint.DisallowFunctions) > 0 {
				checkDisallowedFunctions(tmpl, filePath, opts.Config.Lint.DisallowFunctions, result)
			}
		}
	}

	return nil
}

// lintWalk recursively walks a directory tree and lints all templates
func lintWalk(srcDir string, values map[string]any, opts LintOptions, result *LintResult) error {
	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}

	// Collect template extensions
	exts := map[string]bool{".tpl": true}
	for _, e := range opts.Shared.ExtraExts {
		exts["."+e] = true
	}

	// Walk the directory tree
	err = filepath.Walk(absSrc, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Check if this is a template file
		ext := filepath.Ext(path)
		if !exts[ext] {
			return nil
		}

		// Lint this file
		return lintSingleFile(path, values, opts, result)
	})

	return err
}

// checkUndefinedVariables checks for undefined variables in a template
func checkUndefinedVariables(tpl *template.Template, path string, values map[string]any, opts LintOptions, result *LintResult) {
	if tpl.Tree == nil {
		return
	}

	// Extract all variable references from the template
	vars := extractVariables(tpl.Tree)

	// Determine severity based on config
	severity := "warn"
	if opts.Config != nil && opts.Config.Lint.FailOnUndefined {
		severity = "error"
	}

	// Check each variable against the values
	for _, varPath := range vars {
		if !checkVariableExists(varPath, values) {
			result.Issues = append(result.Issues, LintIssue{
				Severity: severity,
				Category: "undefined",
				File:     path,
				Message:  fmt.Sprintf("variable %s is undefined", varPath),
			})

			if severity == "error" {
				result.Errors++
			} else {
				result.Warns++
			}
		}
	}
}

// extractVariables extracts all variable references from a template AST
//
//nolint:dupl // Similar to extractFunctionCalls but extracts different data
func extractVariables(tree *parse.Tree) []string {
	vars := make(map[string]bool)

	var walk func(node parse.Node)
	walk = func(node parse.Node) {
		if node == nil {
			return
		}

		switch n := node.(type) {
		case *parse.ActionNode:
			extractFromPipe(n.Pipe, vars)
		case *parse.IfNode:
			extractFromPipe(n.Pipe, vars)
			walkList(n.List, walk)
			if n.ElseList != nil {
				walkList(n.ElseList, walk)
			}
		case *parse.RangeNode:
			extractFromPipe(n.Pipe, vars)
			walkList(n.List, walk)
			if n.ElseList != nil {
				walkList(n.ElseList, walk)
			}
		case *parse.WithNode:
			extractFromPipe(n.Pipe, vars)
			walkList(n.List, walk)
			if n.ElseList != nil {
				walkList(n.ElseList, walk)
			}
		case *parse.ListNode:
			walkList(n, walk)
		case *parse.TemplateNode:
			if n.Pipe != nil {
				extractFromPipe(n.Pipe, vars)
			}
		}
	}

	walk(tree.Root)

	// Convert map to slice
	result := make([]string, 0, len(vars))
	for v := range vars {
		result = append(result, v)
	}
	return result
}

// walkList walks all nodes in a list
func walkList(list *parse.ListNode, walk func(parse.Node)) {
	if list == nil {
		return
	}
	for _, node := range list.Nodes {
		walk(node)
	}
}

// extractFromPipe extracts variable references from a pipe
func extractFromPipe(pipe *parse.PipeNode, vars map[string]bool) {
	if pipe == nil {
		return
	}

	for _, cmd := range pipe.Cmds {
		for _, arg := range cmd.Args {
			extractFromArg(arg, vars)
		}
	}
}

// extractFromArg extracts variable references from an argument
func extractFromArg(arg parse.Node, vars map[string]bool) {
	switch a := arg.(type) {
	case *parse.FieldNode:
		// This is a field access like .field or .nested.field
		path := "." + strings.Join(a.Ident, ".")
		vars[path] = true
	case *parse.ChainNode:
		// This is a method chain
		if a.Node != nil {
			extractFromArg(a.Node, vars)
		}
	case *parse.PipeNode:
		extractFromPipe(a, vars)
	}
}

// checkVariableExists checks if a variable path exists in the values
func checkVariableExists(varPath string, values map[string]any) bool {
	// Remove leading dot
	varPath = strings.TrimPrefix(varPath, ".")

	// Handle special cases
	if varPath == "" || varPath == "Files" || varPath == "Values" {
		return true
	}

	// Split the path and traverse the values
	parts := strings.Split(varPath, ".")
	current := values

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return false
		}

		// If this is the last part, we found it
		if i == len(parts)-1 {
			return true
		}

		// Otherwise, traverse deeper
		switch v := val.(type) {
		case map[string]any:
			current = v
		case map[any]any:
			// Convert to map[string]any
			m := make(map[string]any)
			for k, v := range v {
				if ks, ok := k.(string); ok {
					m[ks] = v
				}
			}
			current = m
		default:
			// Can't traverse further
			return false
		}
	}

	return false
}

// extractLineNumber tries to extract a line number from an error message
func extractLineNumber(errMsg string) int {
	// Go template errors often include "line X" in the message
	// Example: "template: file.tpl:12: unexpected {{end}}"
	parts := strings.Split(errMsg, ":")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if i > 0 && i < len(parts)-1 {
			var line int
			if _, err := fmt.Sscanf(part, "%d", &line); err == nil {
				return line
			}
		}
	}
	return 0
}

// printLintResults prints the lint results to stdout
func printLintResults(result *LintResult, opts LintOptions) {
	switch opts.Format {
	case "json":
		printLintResultsJSON(result)
	case "github-actions":
		printLintResultsGitHubActions(result)
	default:
		printLintResultsText(result, opts.Shared.NoColor)
	}
}

// printLintResultsText prints results in human-readable text format
func printLintResultsText(result *LintResult, noColor bool) {
	if len(result.Issues) == 0 {
		printSuccess("✓ No issues found", noColor)
		return
	}

	for _, issue := range result.Issues {
		var prefix string
		if issue.Severity == "error" {
			prefix = colorize("[lint:error:"+issue.Category+"]", "red", noColor)
		} else {
			prefix = colorize("[lint:warn:"+issue.Category+"]", "yellow", noColor)
		}

		location := issue.File
		if issue.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, issue.Line)
		}

		fmt.Printf("%s %s: %s\n", prefix, location, issue.Message)
	}

	fmt.Println()
	if result.Errors > 0 {
		printError(fmt.Sprintf("✗ Found %d error(s)", result.Errors), noColor)
	}
	if result.Warns > 0 {
		printWarning(fmt.Sprintf("⚠ Found %d warning(s)", result.Warns), noColor)
	}
}

// printLintResultsJSON prints results in JSON format
func printLintResultsJSON(result *LintResult) {
	fmt.Println("{")
	fmt.Printf("  \"errors\": %d,\n", result.Errors)
	fmt.Printf("  \"warnings\": %d,\n", result.Warns)
	fmt.Println("  \"issues\": [")

	for i, issue := range result.Issues {
		comma := ","
		if i == len(result.Issues)-1 {
			comma = ""
		}
		fmt.Printf("    {\"severity\": %q, \"category\": %q, \"file\": %q, \"line\": %d, \"message\": %q}%s\n",
			issue.Severity, issue.Category, issue.File, issue.Line, issue.Message, comma)
	}

	fmt.Println("  ]")
	fmt.Println("}")
}

// printLintResultsGitHubActions prints results in GitHub Actions format
func printLintResultsGitHubActions(result *LintResult) {
	for _, issue := range result.Issues {
		// GitHub Actions annotation format:
		// ::error file={name},line={line},col={col}::{message}
		// ::warning file={name},line={line},col={col}::{message}
		level := issue.Severity
		if level == "warn" {
			level = "warning"
		}

		location := fmt.Sprintf("file=%s", issue.File)
		if issue.Line > 0 {
			location += fmt.Sprintf(",line=%d", issue.Line)
		}

		fmt.Printf("::%s %s::%s\n", level, location, issue.Message)
	}
}

// Helper functions for colored output
func colorize(text, color string, noColor bool) string {
	if noColor {
		return text
	}

	colors := map[string]string{
		"red":    "\033[31m",
		"yellow": "\033[33m",
		"green":  "\033[32m",
		"reset":  "\033[0m",
	}

	if code, ok := colors[color]; ok {
		return code + text + colors["reset"]
	}
	return text
}

func printError(msg string, noColor bool) {
	fmt.Println(colorize(msg, "red", noColor))
}

func printWarning(msg string, noColor bool) {
	fmt.Println(colorize(msg, "yellow", noColor))
}

func printSuccess(msg string, noColor bool) {
	fmt.Println(colorize(msg, "green", noColor))
}

// checkRequiredVars ensures that all required variables are present in values
func checkRequiredVars(values map[string]any, required []string, result *LintResult) {
	for _, varPath := range required {
		if !checkVariableExists(varPath, values) {
			result.Issues = append(result.Issues, LintIssue{
				Severity: "error",
				Category: "required",
				File:     "",
				Message:  fmt.Sprintf("required variable %s is not defined", varPath),
			})
			result.Errors++
		}
	}
}

// shouldExcludeFile checks if a file path matches any exclude patterns
func shouldExcludeFile(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
		// Also try matching against full path
		matched, err = filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// checkDisallowedFunctions inspects template AST for disallowed function calls
func checkDisallowedFunctions(tpl *template.Template, path string, disallowed []string, result *LintResult) {
	if tpl.Tree == nil || len(disallowed) == 0 {
		return
	}

	// Create a map for faster lookup
	disallowMap := make(map[string]bool)
	for _, fn := range disallowed {
		disallowMap[fn] = true
	}

	// Walk the AST and find function calls
	funcs := extractFunctionCalls(tpl.Tree)

	for _, fn := range funcs {
		if disallowMap[fn] {
			result.Issues = append(result.Issues, LintIssue{
				Severity: "error",
				Category: "function",
				File:     path,
				Message:  fmt.Sprintf("disallowed function %q is used", fn),
			})
			result.Errors++
		}
	}
}

// extractFunctionCalls extracts all function calls from a template AST
//
//nolint:dupl // Similar to extractVariables but extracts different data
func extractFunctionCalls(tree *parse.Tree) []string {
	funcs := make(map[string]bool)

	var walk func(node parse.Node)
	walk = func(node parse.Node) {
		if node == nil {
			return
		}

		switch n := node.(type) {
		case *parse.ActionNode:
			extractFuncsFromPipe(n.Pipe, funcs)
		case *parse.IfNode:
			extractFuncsFromPipe(n.Pipe, funcs)
			walkList(n.List, walk)
			if n.ElseList != nil {
				walkList(n.ElseList, walk)
			}
		case *parse.RangeNode:
			extractFuncsFromPipe(n.Pipe, funcs)
			walkList(n.List, walk)
			if n.ElseList != nil {
				walkList(n.ElseList, walk)
			}
		case *parse.WithNode:
			extractFuncsFromPipe(n.Pipe, funcs)
			walkList(n.List, walk)
			if n.ElseList != nil {
				walkList(n.ElseList, walk)
			}
		case *parse.ListNode:
			walkList(n, walk)
		case *parse.TemplateNode:
			if n.Pipe != nil {
				extractFuncsFromPipe(n.Pipe, funcs)
			}
		}
	}

	walk(tree.Root)

	// Convert map to slice
	result := make([]string, 0, len(funcs))
	for fn := range funcs {
		result = append(result, fn)
	}
	return result
}

// extractFuncsFromPipe extracts function names from a pipe
func extractFuncsFromPipe(pipe *parse.PipeNode, funcs map[string]bool) {
	if pipe == nil {
		return
	}

	for _, cmd := range pipe.Cmds {
		if len(cmd.Args) > 0 {
			// First arg might be a function identifier
			if ident, ok := cmd.Args[0].(*parse.IdentifierNode); ok {
				funcs[ident.Ident] = true
			}
		}
	}
}
