# Installation Guide

This guide covers all the ways to install templr on your system.

## Quick Install

### macOS/Linux via Homebrew (Recommended)

The easiest way to install templr on macOS or Linux:

```bash
brew tap kanopi/templr
brew install templr
```

To upgrade to the latest version:

```bash
brew upgrade templr
```

### One-Line Install Script

Install the latest version with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/kanopi/templr/main/get-templr.sh | bash
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/kanopi/templr/main/get-templr.sh | bash -s 1.2.3
```

This script:
- Detects your operating system and architecture
- Downloads the appropriate binary
- Installs it to `/usr/local/bin`
- Verifies the installation

## Other Installation Methods

### Download Pre-Built Binary

Download the latest release for your platform:

1. Visit the [GitHub Releases](https://github.com/kanopi/templr/releases) page
2. Download the appropriate archive for your OS and architecture:
   - `templr_Darwin_x86_64.tar.gz` - macOS Intel
   - `templr_Darwin_arm64.tar.gz` - macOS Apple Silicon
   - `templr_Linux_x86_64.tar.gz` - Linux Intel/AMD
   - `templr_Linux_arm64.tar.gz` - Linux ARM
   - `templr_Windows_x86_64.zip` - Windows Intel/AMD
3. Extract the archive:
   ```bash
   tar -xzf templr_*.tar.gz  # macOS/Linux
   # or
   unzip templr_*.zip        # Windows
   ```
4. Move the binary to your PATH:
   ```bash
   # macOS/Linux
   sudo mv templr /usr/local/bin/
   sudo chmod +x /usr/local/bin/templr

   # Windows (PowerShell as Administrator)
   Move-Item templr.exe C:\Windows\System32\
   ```

### Using Docker

Run templr without installing it locally using the official Docker image:

#### Walk Mode (Render Directory Tree)

```bash
docker run --rm -v $(pwd):/work -w /work kanopi/templr walk --src /work/templates --dst /work/out
```

#### Single File Rendering

```bash
docker run --rm -v $(pwd):/work -w /work kanopi/templr render -in /work/template.tpl -data /work/values.yaml -out /work/output.yaml
```

#### Lint Templates

```bash
docker run --rm -v $(pwd):/work -w /work kanopi/templr lint --src /work/templates -d /work/values.yaml
```

#### Docker Tips

- Use `--rm` to automatically remove the container after execution
- Mount your working directory with `-v $(pwd):/work`
- Set working directory with `-w /work`
- All paths must be absolute inside the container (e.g., `/work/templates`)

### Building from Source

If you want to build templr from source:

#### Prerequisites

- Go 1.21 or later
- Git

#### Build Steps

```bash
# Clone the repository
git clone https://github.com/kanopi/templr.git
cd templr

# Build the binary
go build -o templr .

# (Optional) Install to your PATH
sudo mv templr /usr/local/bin/

# Verify the build
templr version
```

#### Build with Make

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint

# Build release binaries for all platforms
goreleaser build --snapshot --clean
```

## Verification

After installation, verify templr is working correctly:

### Check Version

```bash
templr version
```

Expected output: `dev` (if built from source) or version number like `1.0.0`

### Run Help

```bash
templr --help
```

You should see the usage information and available commands.

### Test Rendering

Create a simple test:

```bash
# Create a template
echo 'Hello {{ .name }}!' > test.tpl

# Render it
templr render -in test.tpl --set name=World

# Expected output: Hello World!
```

If you see "Hello World!", templr is installed correctly!

## Platform-Specific Notes

### macOS

#### Apple Silicon (M1/M2/M3)

Make sure to download the ARM64 version for better performance:

```bash
# Homebrew automatically installs the correct version
brew tap kanopi/templr
brew install templr
```

#### Security Warning

If you see "templr cannot be opened because it is from an unidentified developer":

```bash
# Remove the quarantine attribute
xattr -d com.apple.quarantine /usr/local/bin/templr
```

Or allow it in System Preferences → Security & Privacy.

### Linux

#### Permissions

If you get a "permission denied" error:

```bash
chmod +x templr
```

#### PATH Issues

If `templr` is not found after installation, add it to your PATH:

```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH="$PATH:/usr/local/bin"

# Reload your shell
source ~/.bashrc  # or source ~/.zshrc
```

### Windows

#### PowerShell Execution Policy

If you get an execution policy error:

```powershell
# Run as Administrator
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

#### PATH Configuration

Add templr to your PATH:

1. Search for "Environment Variables" in Windows Settings
2. Edit "Path" under System Variables
3. Add the directory containing `templr.exe`
4. Restart your terminal

## Updating templr

### Homebrew

```bash
brew upgrade templr
```

### Manual Update

1. Download the latest release
2. Replace the existing binary
3. Verify the new version: `templr version`

### Docker

Pull the latest image:

```bash
docker pull kanopi/templr:latest
```

## Uninstallation

### Homebrew

```bash
brew uninstall templr
brew untap kanopi/templr
```

### Manual Installation

```bash
# Remove the binary
sudo rm /usr/local/bin/templr

# (Optional) Remove user config
rm -rf ~/.config/templr
```

### Docker

```bash
# Remove the image
docker rmi kanopi/templr
```

## Troubleshooting

### Command Not Found

**Problem**: `bash: templr: command not found`

**Solution**:
- Verify templr is in your PATH: `which templr`
- Check installation location: `ls -l /usr/local/bin/templr`
- Add to PATH if needed (see platform-specific notes above)

### Permission Denied

**Problem**: `permission denied: ./templr`

**Solution**:
```bash
chmod +x templr
```

### Version Mismatch

**Problem**: `templr version` shows old version after update

**Solution**:
- Check if multiple versions exist: `which -a templr`
- Remove old versions
- Clear shell hash: `hash -r`

### Docker Volume Issues

**Problem**: Templates not found in Docker

**Solution**:
- Ensure you're mounting the correct directory
- Use absolute paths inside container: `/work/templates` not `./templates`
- Check file permissions on host

## Next Steps

- [Quick Start Guide](README.md#quick-start)
- [CLI Reference](cli-reference.md)
- [Templating Guide](templating-guide.md)
- [Configuration Files](configuration.md)

## Getting Help

- [Documentation Hub](README.md)
- [GitHub Issues](https://github.com/kanopi/templr/issues)
- [GitHub Discussions](https://github.com/kanopi/templr/discussions)
