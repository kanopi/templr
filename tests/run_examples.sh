#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$ROOT/.bin"
OUT_DIR="$ROOT/.out"
GOLDEN_DIR="$ROOT/tests/golden"
EXAMPLES_DIR="$ROOT/playground"

info()  { echo -e "\033[1;34m[info]\033[0m $*"; }
pass()  { echo -e "\033[1;32m[pass]\033[0m $*"; }
fail()  { echo -e "\033[1;31m[fail]\033[0m $*"; exit 1; }

mkdir -p "$BIN_DIR" "$OUT_DIR" "$GOLDEN_DIR"

info "Building templr..."
go build -o "$BIN_DIR/templr" ./ || fail "build failed"

normalize_paths() {
  sed -e "s|$ROOT|<ROOT>|g"       -e "s|$OUT_DIR|<OUT>|g"       -e "s|$EXAMPLES_DIR|<PLAY>|g"
}

run_and_diff() {
  local name="$1"; shift
  local cmd=( "$@" )

  local out="$OUT_DIR/$name"
  mkdir -p "$(dirname "$out")"

  if [[ " ${cmd[*]} " == *" -out "* ]]; then
    "${cmd[@]}"
  else
    "${cmd[@]}" > "$out"
  fi

  local golden="$GOLDEN_DIR/$name"
  if [[ -n "${UPDATE_GOLDEN:-}" ]]; then
    info "UPDATING golden: $golden"
    mkdir -p "$(dirname "$golden")"
    if [[ " ${cmd[*]} " == *" -out "* ]]; then
      for i in "${!cmd[@]}"; do
        if [[ "${cmd[$i]}" == "-out" ]]; then
          cp "${cmd[$((i+1))]}" "$golden"
        fi
      done
    else
      cp "$out" "$golden"
    fi
    return
  fi

  local actual="$out"
  if [[ " ${cmd[*]} " == *" -out "* ]]; then
    for i in "${!cmd[@]}"; do
      if [[ "${cmd[$i]}" == "-out" ]]; then
        actual="${cmd[$((i+1))]}"
      fi
    done
  fi

  if ! diff -u "$golden" "$actual"; then
    fail "Mismatch for $name"
  fi
  pass "$name"
}

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

info "Running scenarios..."

# Single-file render
run_and_diff "single/out.yaml"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/template.tpl" -data "$EXAMPLES_DIR/values.yaml" -out "$OUT_DIR/single/out.yaml"

# Strict mode
mkdir -p "$OUT_DIR/strict"
set +e
"$BIN_DIR/templr" -in "$EXAMPLES_DIR/strict.tpl" -strict -out "$OUT_DIR/strict/strict.yaml" 2> "$OUT_DIR/strict/err.txt"
if grep -qiE "missing|map has no entry|required" "$OUT_DIR/strict/err.txt"; then
  pass "strict mode fails as expected"
else
  fail "strict mode did not fail as expected"
fi
set -e
run_and_diff "strict/strict.yaml"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/strict.tpl" -strict -data "$EXAMPLES_DIR/vals.yaml" -out "$OUT_DIR/strict/strict.yaml"

# Dir mode
run_and_diff "dir/dirtest.out"   "$BIN_DIR/templr" -dir "$EXAMPLES_DIR/dirtest" -in "$EXAMPLES_DIR/dirtest/main.tpl" -data "$EXAMPLES_DIR/dirtest/values.yaml" -out "$OUT_DIR/dir/dirtest.out"

# Walk mode prune
rm -rf "$OUT_DIR/walk"
"$BIN_DIR/templr" -walk -src "$EXAMPLES_DIR/walk/templates" -dst "$OUT_DIR/walk/out"
[[ -f "$OUT_DIR/walk/out/example/test" ]] || fail "missing rendered file"
[[ ! -f "$OUT_DIR/walk/out/example/test2" ]] || fail "unexpected file created"
[[ ! -d "$OUT_DIR/walk/out/example2" ]] || fail "empty dir not pruned"
pass "walk mode prune/skip"

# Guard behavior
rm -rf "$OUT_DIR/guard" && mkdir -p "$OUT_DIR/guard"
printf "content: original\n" > "$OUT_DIR/guard/file.yaml"
"$BIN_DIR/templr" -in "$EXAMPLES_DIR/guard/tpl.tpl" -out "$OUT_DIR/guard/file.yaml" || true
grep -q "content: original" "$OUT_DIR/guard/file.yaml" || fail "guard overwrite should be skipped"
printf "#templr generated\ncontent: original\n" > "$OUT_DIR/guard/file.yaml"
"$BIN_DIR/templr" -in "$EXAMPLES_DIR/guard/tpl.tpl" -out "$OUT_DIR/guard/file.yaml"
grep -q "content: updated" "$OUT_DIR/guard/file.yaml" || fail "guard overwrite should succeed"
pass "guard behavior"

# Merge + --set
run_and_diff "merge/out.yaml"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/merge/app.tpl" -data "$EXAMPLES_DIR/merge/base.yaml" -f "$EXAMPLES_DIR/merge/staging.yaml" --set replicas=3 -out "$OUT_DIR/merge/out.yaml"

# .Files API
run_and_diff "files/secret.yaml"   "$BIN_DIR/templr" -dir "$EXAMPLES_DIR/files" -in "$EXAMPLES_DIR/files/secret.tpl" -out "$OUT_DIR/files/secret.yaml"

# Custom delimiters
run_and_diff "delims/out.txt"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/delims/vue.tpl" -data "$EXAMPLES_DIR/delims/vals.yaml" -ldelim '<<' -rdelim '>>' -out "$OUT_DIR/delims/out.txt"

# required helper
mkdir -p "$OUT_DIR/req"
set +e
"$BIN_DIR/templr" -in "$EXAMPLES_DIR/req/tpl.tpl" -out "$OUT_DIR/req/out.yaml" 2> "$OUT_DIR/req/err.txt"
grep -q "name is required" "$OUT_DIR/req/err.txt" || fail "required should error"
set -e
run_and_diff "req/out.yaml"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/req/tpl.tpl" --set name=alpha -out "$OUT_DIR/req/out.yaml"

# whitespace-only skip
rm -rf "$OUT_DIR/empty"
"$BIN_DIR/templr" -in "$EXAMPLES_DIR/empty/only-spaces.tpl" -out "$OUT_DIR/empty/out.txt" || true
[[ ! -f "$OUT_DIR/empty/out.txt" ]] || fail "whitespace-only output should be skipped"
pass "whitespace-only skip"

# helpers pre-render
run_and_diff "helpers/demo.txt"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/helpers_demo.tpl" -data "$EXAMPLES_DIR/helpers_values.yaml" -out "$OUT_DIR/helpers/demo.txt"

# default-missing (global fallback)
run_and_diff "missing/global.out.txt"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/default-missing/template.tpl" -data "$EXAMPLES_DIR/default-missing/values.yaml" --default-missing "N/A"

# safe helper (per-variable fallback)
run_and_diff "missing/safe.out.txt"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/safe-helper/tpl.tpl" -data "$EXAMPLES_DIR/safe-helper/values.yaml"

# default-missing with pipelines (edge case: missing vars in pipeline expressions)
run_and_diff "missing/pipeline.out.txt"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/default-missing-pipeline/template.tpl" -data "$EXAMPLES_DIR/default-missing-pipeline/values.yaml" --default-missing "N/A"

# default-missing with nested templates (edge case: missing vars in template definitions)
run_and_diff "missing/nested.out.txt"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/default-missing-nested/template.tpl" -data "$EXAMPLES_DIR/default-missing-nested/values.yaml" --default-missing "N/A"

# default-missing with helpers (edge case: missing vars in _helpers.tpl)
run_and_diff "missing/helpers.out.yaml"   "$BIN_DIR/templr" -dir "$EXAMPLES_DIR/default-missing-helpers" -in "$EXAMPLES_DIR/default-missing-helpers/deployment.tpl" -data "$EXAMPLES_DIR/default-missing-helpers/values.yaml" --default-missing "N/A" -out "$OUT_DIR/missing/helpers.out.yaml"

# stdin -> stdout rendering (no -out): echo template and pipe into templr
run_and_diff "stdin/out.txt" bash -lc "echo 'Hello {{ .name }}' | $BIN_DIR/templr -data $EXAMPLES_DIR/stdin-stdout/values.yaml"

# guard injection placements
run_and_diff "guards/shebang.sh"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/guards/shebang.sh.tpl" -out "$OUT_DIR/guards/shebang.sh"
run_and_diff "guards/php.php"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/guards/php.tpl" -out "$OUT_DIR/guards/php.php"
run_and_diff "guards/json.json"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/guards/json.tpl" -out "$OUT_DIR/guards/json.json"
run_and_diff "guards/html.html"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/guards/html.tpl" -out "$OUT_DIR/guards/html.html"
run_and_diff "guards/style.css"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/guards/style.css.tpl" -out "$OUT_DIR/guards/style.css"
run_and_diff "guards/app.js"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/guards/app.js.tpl" -out "$OUT_DIR/guards/app.js"
run_and_diff "guards/Dockerfile"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/guards/Dockerfile.tpl" -out "$OUT_DIR/guards/Dockerfile"

# helpers glob
run_and_diff "helpers_glob/with_helpers.txt"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/helpers_glob/consumer.tpl" -data "$EXAMPLES_DIR/helpers_glob/values.yaml" -out "$OUT_DIR/helpers_glob/with_helpers.txt"
run_and_diff "helpers_glob/without_helpers.txt"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/helpers_glob/consumer.tpl" -data "$EXAMPLES_DIR/helpers_glob/values.yaml" -helpers "" -out "$OUT_DIR/helpers_glob/without_helpers.txt"

# mutations
run_and_diff "mutations/out.txt"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/mutations/demo.tpl" -out "$OUT_DIR/mutations/out.txt"

# Files.Glob
run_and_diff "filesglob/list.txt"   "$BIN_DIR/templr" -dir "$EXAMPLES_DIR/filesglob" -in "$EXAMPLES_DIR/filesglob/list.tpl" -out "$OUT_DIR/filesglob/list.txt"

# dry-run messaging
mkdir -p "$OUT_DIR/dryrun"
"$BIN_DIR/templr" -in "$EXAMPLES_DIR/dryrun/basic.tpl" -out "$OUT_DIR/dryrun/preview.txt" --dry-run | normalize_paths > "$OUT_DIR/dryrun/stdout.txt"
if [[ -n "${UPDATE_GOLDEN:-}" ]]; then
  mkdir -p "$GOLDEN_DIR/dryrun"
  cp "$OUT_DIR/dryrun/stdout.txt" "$GOLDEN_DIR/dryrun/stdout.txt"
else
  if ! diff -u "$GOLDEN_DIR/dryrun/stdout.txt" "$OUT_DIR/dryrun/stdout.txt"; then
    fail "Mismatch for dry-run output"
  fi
fi
pass "dry-run messaging"

# stdout (no -out)
run_and_diff "stdout/out.txt"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/stdout/out.tpl"

# DEFAULTS: single-file
run_and_diff "defaults/sf.out.yaml"   "$BIN_DIR/templr" -in "$EXAMPLES_DIR/defaults_sf/template.tpl" -out "$OUT_DIR/defaults/sf.out.yaml"

# DEFAULTS: dir mode
run_and_diff "defaults/dir.out.yaml"   "$BIN_DIR/templr" -dir "$EXAMPLES_DIR/defaults_dir" -in "$EXAMPLES_DIR/defaults_dir/main.tpl" -out "$OUT_DIR/defaults/dir.out.yaml"

# DEFAULTS: walk mode
rm -rf "$OUT_DIR/defaults_walk"
"$BIN_DIR/templr" -walk -src "$EXAMPLES_DIR/defaults_walk/templates" -dst "$OUT_DIR/defaults_walk/out"
run_and_diff "defaults_walk/a" cat "$OUT_DIR/defaults_walk/out/a"
run_and_diff "defaults_walk/b" cat "$OUT_DIR/defaults_walk/out/b"

# EXTENSIONS: walk mode with -ext md -ext txt
rm -rf "$OUT_DIR/ext_walk"
"$BIN_DIR/templr" -walk -src "$EXAMPLES_DIR/ext_walk/templates" -dst "$OUT_DIR/ext_walk/out" -ext md -ext txt
run_and_diff "ext_walk/README" cat "$OUT_DIR/ext_walk/out/README"
run_and_diff "ext_walk/info" cat "$OUT_DIR/ext_walk/out/info"

# EXTENSIONS: dir mode parse .md; render to .md
run_and_diff "ext_dir/readme.md"   "$BIN_DIR/templr" -dir "$EXAMPLES_DIR/ext_dir" -in "$EXAMPLES_DIR/ext_dir/README.md" -out "$OUT_DIR/ext_dir/readme.md" -ext md

pass "ALL TESTS PASSED"
