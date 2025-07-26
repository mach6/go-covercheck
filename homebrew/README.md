# Homebrew Formula for go-covercheck

This directory contains the Homebrew formula for `go-covercheck`.

## Installation

### Option 1: Direct Formula Installation

```bash
brew install https://raw.githubusercontent.com/mach6/go-covercheck/main/homebrew/go-covercheck.rb
```

### Option 2: Create a Custom Tap

To create your own tap and install from there:

1. Create a repository named `homebrew-tap` 
2. Copy the `go-covercheck.rb` formula to the `Formula/` directory in that repository
3. Install using: `brew install your-username/tap/go-covercheck`

### Option 3: Local Installation

For testing or development:

```bash
brew install --build-from-source ./homebrew/go-covercheck.rb
```

## Updating the Formula

When a new version is released, update the formula by:

1. Changing the `url` to point to the new version tag
2. Updating the `sha256` checksum (you can get this with `curl -sL <new-url> | sha256sum`)
3. The version will be automatically extracted from the URL

## Testing

Test the formula with:

```bash
brew install --build-from-source ./homebrew/go-covercheck.rb
go-covercheck --version
```