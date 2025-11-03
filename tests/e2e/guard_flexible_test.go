package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardDetectionAcrossFileTypes(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name            string
		filename        string
		existingContent string
		templateContent string
		shouldOverwrite bool
		description     string
	}{
		{
			name:            "Go file with // guard",
			filename:        "test.go",
			existingContent: "// #templr generated\npackage main\n",
			templateContent: "package new\n",
			shouldOverwrite: true,
			description:     "Should detect // style guard in Go files",
		},
		{
			name:            "Go file without guard",
			filename:        "test.go",
			existingContent: "package main\n",
			templateContent: "package new\n",
			shouldOverwrite: false,
			description:     "Should not overwrite Go file without guard",
		},
		{
			name:            "Shell script with # guard",
			filename:        "test.sh",
			existingContent: "# #templr generated\necho hello\n",
			templateContent: "echo new\n",
			shouldOverwrite: true,
			description:     "Should detect # style guard in shell scripts",
		},
		{
			name:            "Shell script with shebang and guard",
			filename:        "test.sh",
			existingContent: "#!/bin/bash\n# #templr generated\necho hello\n",
			templateContent: "echo new\n",
			shouldOverwrite: true,
			description:     "Should detect guard after shebang",
		},
		{
			name:            "Shell script without guard",
			filename:        "test.sh",
			existingContent: "#!/bin/bash\necho hello\n",
			templateContent: "echo new\n",
			shouldOverwrite: false,
			description:     "Should not overwrite shell script without guard",
		},
		{
			name:            "PHP file with guard after <?php",
			filename:        "test.php",
			existingContent: "<?php\n// #templr generated\necho 'hi';\n",
			templateContent: "<?php\necho 'new';",
			shouldOverwrite: true,
			description:     "Should detect // guard after <?php",
		},
		{
			name:            "PHP file with guard in header block",
			filename:        "test.php",
			existingContent: "<?php // #templr generated ?>\necho 'hi';\n",
			templateContent: "echo 'new';\n",
			shouldOverwrite: true,
			description:     "Should detect guard in <?php header block",
		},
		{
			name:            "HTML file with <!-- guard -->",
			filename:        "test.html",
			existingContent: "<!-- #templr generated -->\n<html></html>\n",
			templateContent: "<html><body></body></html>\n",
			shouldOverwrite: true,
			description:     "Should detect HTML comment guard",
		},
		{
			name:            "HTML file without guard",
			filename:        "test.html",
			existingContent: "<html></html>\n",
			templateContent: "<html><body></body></html>\n",
			shouldOverwrite: false,
			description:     "Should not overwrite HTML without guard",
		},
		{
			name:            "CSS file with /* guard */",
			filename:        "test.css",
			existingContent: "/* #templr generated */\nbody { }\n",
			templateContent: "body { margin: 0; }\n",
			shouldOverwrite: true,
			description:     "Should detect CSS block comment guard",
		},
		{
			name:            "YAML file with # guard",
			filename:        "test.yaml",
			existingContent: "# #templr generated\nkey: value\n",
			templateContent: "key: newvalue\n",
			shouldOverwrite: true,
			description:     "Should detect # guard in YAML",
		},
		{
			name:            "Python file with # guard",
			filename:        "test.py",
			existingContent: "# #templr generated\nprint('hello')\n",
			templateContent: "print('new')\n",
			shouldOverwrite: true,
			description:     "Should detect # guard in Python",
		},
		{
			name:            "Python file with shebang and guard",
			filename:        "test.py",
			existingContent: "#!/usr/bin/env python3\n# #templr generated\nprint('hello')\n",
			templateContent: "print('new')\n",
			shouldOverwrite: true,
			description:     "Should detect guard after Python shebang",
		},
		{
			name:            "JavaScript file with // guard",
			filename:        "test.js",
			existingContent: "// #templr generated\nconsole.log('hi');\n",
			templateContent: "console.log('new');\n",
			shouldOverwrite: true,
			description:     "Should detect // guard in JavaScript",
		},
		{
			name:            "TypeScript file with // guard",
			filename:        "test.ts",
			existingContent: "// #templr generated\nconst x: string = 'hi';\n",
			templateContent: "const x: string = 'new';\n",
			shouldOverwrite: true,
			description:     "Should detect // guard in TypeScript",
		},
		{
			name:            "Rust file with // guard",
			filename:        "test.rs",
			existingContent: "// #templr generated\nfn main() {}\n",
			templateContent: "fn main() { println!(\"new\"); }\n",
			shouldOverwrite: true,
			description:     "Should detect // guard in Rust",
		},
		{
			name:            "JSON file without guard (should not overwrite)",
			filename:        "test.json",
			existingContent: `{"key": "value"}`,
			templateContent: `{"key": "newvalue"}`,
			shouldOverwrite: false,
			description:     "Should not overwrite JSON (no comment support)",
		},
		{
			name:            "Dockerfile with # guard",
			filename:        "Dockerfile",
			existingContent: "# #templr generated\nFROM alpine\n",
			templateContent: "FROM debian\n",
			shouldOverwrite: true,
			description:     "Should detect # guard in Dockerfile",
		},
		{
			name:            "Guard with no space after comment token",
			filename:        "test.sh",
			existingContent: "##templr generated\necho hi\n",
			templateContent: "echo new\n",
			shouldOverwrite: true,
			description:     "Should detect guard without space (##marker)",
		},
		{
			name:            "Guard with no space in Go file",
			filename:        "test.go",
			existingContent: "//#templr generated\npackage main\n",
			templateContent: "package new\n",
			shouldOverwrite: true,
			description:     "Should detect guard without space (//marker)",
		},
		{
			name:            "File with CRLF line endings and guard",
			filename:        "test.sh",
			existingContent: "# #templr generated\r\necho hi\r\n",
			templateContent: "echo new\n",
			shouldOverwrite: true,
			description:     "Should detect guard with CRLF line endings",
		},
		{
			name:            "File with UTF-8 BOM and guard",
			filename:        "test.sh",
			existingContent: "\xEF\xBB\xBF# #templr generated\necho hi\n",
			templateContent: "echo new\n",
			shouldOverwrite: true,
			description:     "Should detect guard after UTF-8 BOM",
		},
		{
			name:            "TOML file with # guard",
			filename:        "config.toml",
			existingContent: "# #templr generated\n[section]\nkey = \"value\"\n",
			templateContent: "[section]\nkey = \"new\"\n",
			shouldOverwrite: true,
			description:     "Should detect # guard in TOML",
		},
		{
			name:            "Markdown file with <!-- guard -->",
			filename:        "README.md",
			existingContent: "<!-- #templr generated -->\n# Title\n",
			templateContent: "# New Title\n",
			shouldOverwrite: true,
			description:     "Should detect HTML comment guard in Markdown",
		},
		{
			name:            "XML file with <!-- guard -->",
			filename:        "config.xml",
			existingContent: "<!-- #templr generated -->\n<root></root>\n",
			templateContent: "<root><item/></root>\n",
			shouldOverwrite: true,
			description:     "Should detect guard in XML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := t.TempDir()
			outFile := filepath.Join(td, tt.filename)
			tplFile := filepath.Join(td, "input.tpl")

			// Write existing file
			if err := os.WriteFile(outFile, []byte(tt.existingContent), 0o644); err != nil {
				t.Fatal(err)
			}

			// Write template
			if err := os.WriteFile(tplFile, []byte(tt.templateContent), 0o644); err != nil {
				t.Fatal(err)
			}

			// Try to render
			_, stderr, _ := run(t, bin, "-in", tplFile, "-out", outFile)

			// Read result
			result, _ := os.ReadFile(outFile)

			if tt.shouldOverwrite {
				// Should have been overwritten - file should be different from original
				// and should NOT contain the old content
				if string(result) == tt.existingContent {
					t.Errorf("%s: expected file to be overwritten, but content unchanged: %q", tt.description, string(result))
				}
				// Check that it doesn't contain old distinctive content (avoid false positives)
				// For most tests, check that specific old content is gone
				if strings.Contains(tt.existingContent, "hi") && strings.Contains(string(result), "'hi'") {
					t.Errorf("%s: expected old content 'hi' to be replaced, but got: %q", tt.description, string(result))
				}
				if strings.Contains(tt.existingContent, "hello") && strings.Contains(string(result), "hello") &&
					!strings.Contains(tt.templateContent, "hello") {
					t.Errorf("%s: expected old content 'hello' to be replaced, but got: %q", tt.description, string(result))
				}
			} else {
				// Should have been skipped (warning in stderr)
				if !strings.Contains(stderr, "[templr:warn:guard]") {
					t.Errorf("%s: expected guard warning, got stderr: %s", tt.description, stderr)
				}
				// Content should be unchanged (original content still there)
				if string(result) != tt.existingContent {
					t.Errorf("%s: expected file to remain unchanged, but it was modified. Original: %q, Got: %q", tt.description, tt.existingContent, string(result))
				}
			}
		})
	}
}

func TestGuardDetectionEdgeCases(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	t.Run("multiple guards in file", func(t *testing.T) {
		td := t.TempDir()
		outFile := filepath.Join(td, "test.sh")
		tplFile := filepath.Join(td, "input.tpl")

		// File with guard in multiple places
		existing := "#!/bin/bash\n# #templr generated\necho 'hi'\n# Some other #templr generated comment\n"
		if err := os.WriteFile(outFile, []byte(existing), 0o644); err != nil {
			t.Fatal(err)
		}

		template := "echo 'new'\n"
		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, _ = run(t, bin, "-in", tplFile, "-out", outFile)

		result, _ := os.ReadFile(outFile)
		// Should detect at least one guard and allow overwrite
		if !strings.Contains(string(result), "new") {
			t.Errorf("expected file to be overwritten when guard is present")
		}
	})

	t.Run("guard in middle of file", func(t *testing.T) {
		td := t.TempDir()
		outFile := filepath.Join(td, "test.go")
		tplFile := filepath.Join(td, "input.tpl")

		// Guard not at the top
		existing := "package main\n\n// #templr generated\nfunc main() {}\n"
		if err := os.WriteFile(outFile, []byte(existing), 0o644); err != nil {
			t.Fatal(err)
		}

		template := "package main\n\nfunc main() { println(\"new\") }\n"
		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, _ = run(t, bin, "-in", tplFile, "-out", outFile)

		result, _ := os.ReadFile(outFile)
		// Should still detect guard even if not at top
		if !strings.Contains(string(result), "new") {
			t.Errorf("expected file to be overwritten when guard is present anywhere")
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		td := t.TempDir()
		outFile := filepath.Join(td, "test.sh")
		tplFile := filepath.Join(td, "input.tpl")

		// Guard with different case (should NOT match by default)
		existing := "# #TEMPLR GENERATED\necho 'hi'\n"
		if err := os.WriteFile(outFile, []byte(existing), 0o644); err != nil {
			t.Fatal(err)
		}

		template := "echo 'new'\n"
		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, stderr, _ := run(t, bin, "-in", tplFile, "-out", outFile)

		result, _ := os.ReadFile(outFile)
		// Should NOT overwrite (case-sensitive guard matching)
		if strings.Contains(string(result), "new") {
			t.Errorf("expected file NOT to be overwritten (guard is case-sensitive)")
		}
		if !strings.Contains(stderr, "[templr:warn:guard]") {
			t.Errorf("expected guard warning for case mismatch")
		}
	})
}
