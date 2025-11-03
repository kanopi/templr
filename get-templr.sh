#!/usr/bin/env bash
set -euo pipefail

REPO="kanopi/templr"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
TMP_DIR=$(mktemp -d)
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

# Optional: tag can be provided as first arg or via TEMPLR_TAG env var.
# Examples:
#   ./get-templr.sh v1.2.3
#   TEMPLR_TAG=v1.2.3 ./get-templr.sh
REQ_TAG="${1:-${TEMPLR_TAG:-}}"

usage() {
  cat <<EOF
Usage: $0 [tag]

If [tag] is provided (e.g., v1.2.3), that release will be installed.
Otherwise, the latest release is installed.

You can also set the environment variable TEMPLR_TAG to choose a release:
  TEMPLR_TAG=v1.2.3 $0

You can also set a custom installation directory with:
  INSTALL_DIR=/custom/path $0
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

# Detect OS
case "$OS" in
  linux|darwin|freebsd|windows) ;;
  *) echo "❌ Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture and validate combinations
case "$OS" in
  linux|darwin)
    case "$ARCH" in
      x86_64|amd64) ARCH="amd64" ;;
      aarch64|arm64) ARCH="arm64" ;;
      *)
        echo "❌ Unsupported architecture for $OS: $ARCH" >&2
        exit 1
        ;;
    esac
    ;;
  windows)
    case "$ARCH" in
      x86_64|amd64) ARCH="amd64" ;;
      aarch64|arm64)
        echo "❌ Unsupported architecture for Windows: $ARCH" >&2
        exit 1
        ;;
      *)
        echo "❌ Unsupported architecture for Windows: $ARCH" >&2
        exit 1
        ;;
    esac
    ;;
  *)
    echo "❌ Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

echo "📦 Fetching latest templr release for $OS-$ARCH..."

# Resolve the release tag (use requested tag if provided, else latest)
if [ -n "$REQ_TAG" ]; then
  TAG="$REQ_TAG"
  echo "📌 Using requested tag: $TAG"
  # Optionally verify the tag exists (non-fatal if API hiccups)
  if ! curl -fsSL "https://api.github.com/repos/$REPO/releases/tags/$TAG" >/dev/null 2>&1; then
    echo "❌ Tag '$TAG' not found in repository $REPO." >&2
    exit 1
  fi
else
  if command -v jq >/dev/null 2>&1; then
    TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | jq -r '.tag_name // empty')
  else
    TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)
  fi
  if [ -z "$TAG" ]; then
    echo "❌ Failed to determine latest release tag." >&2
    exit 1
  fi
  echo "🕘 No tag specified; using latest: $TAG"
fi

# Construct download URL
ASSET="templr-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  ASSET="${ASSET}.exe"
fi
URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"
DEST="${INSTALL_DIR}/templr"
if [ "$OS" = "windows" ]; then
  DEST="${DEST}.exe"
fi

echo "⬇️  Downloading $URL"
curl -fsSL "$URL" -o "${TMP_DIR}/templr"
if [ "$OS" = "windows" ]; then
  mv "${TMP_DIR}/templr" "${TMP_DIR}/templr.exe"
fi

chmod +x "${TMP_DIR}/templr" || true
if [ "$OS" = "windows" ]; then
  chmod +x "${TMP_DIR}/templr.exe" || true
fi

if [ "$OS" = "windows" ]; then
  SRC_FILE="${TMP_DIR}/templr.exe"
else
  SRC_FILE="${TMP_DIR}/templr"
fi

if [ -w "$(dirname "$DEST")" ]; then
  mv "$SRC_FILE" "$DEST"
else
  sudo mv "$SRC_FILE" "$DEST"
fi

echo "✅ templr ${TAG} installed to ${DEST}"
"$DEST" -version