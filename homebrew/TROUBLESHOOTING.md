# Installation Issues

## Homebrew Installation

If you're experiencing issues with the Homebrew installation, please try the following troubleshooting steps:

### Common Issues

1. **Formula not found**: Make sure you're using the correct installation command:
   ```bash
   brew install https://raw.githubusercontent.com/mach6/go-covercheck/main/homebrew/go-covercheck.rb
   ```

2. **Build failures**: Ensure you have Go installed:
   ```bash
   brew install go
   ```

3. **Permission issues**: Check that Homebrew has proper permissions:
   ```bash
   brew doctor
   ```

### Updating the Formula

If there's a new version available but the formula hasn't been updated:

1. Check for the latest release on the [releases page](https://github.com/mach6/go-covercheck/releases)
2. The formula should be automatically updated when new releases are tagged
3. If it's not updated, please file an issue

### Manual Installation

If Homebrew installation fails, you can always fall back to:

1. **Go install**: `go install github.com/mach6/go-covercheck/cmd/go-covercheck@latest`
2. **Direct download**: Download from the [releases page](https://github.com/mach6/go-covercheck/releases)
3. **Build from source**: Clone the repository and run `make build`

### Reporting Issues

When reporting Homebrew-related issues, please include:

- Your macOS version
- Homebrew version (`brew --version`)
- Go version (`go version`)
- The exact error message
- Steps to reproduce the issue