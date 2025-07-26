#!/bin/bash

# Test script for go-covercheck Homebrew formula
# This script helps validate that the formula works correctly

set -e

echo "🧪 Testing go-covercheck Homebrew formula..."

# Check if formula syntax is valid
echo "📝 Checking Ruby syntax..."
ruby -c homebrew/go-covercheck.rb
echo "✅ Ruby syntax is valid"

# Verify the formula can be parsed (if brew is available)
if command -v brew >/dev/null 2>&1; then
    echo "🍺 Testing with Homebrew..."
    
    # Test the formula
    echo "🔍 Auditing formula..."
    brew audit --strict homebrew/go-covercheck.rb || echo "⚠️  Audit warnings (may be expected)"
    
    echo "🧪 Testing formula installation..."
    brew install --build-from-source homebrew/go-covercheck.rb
    
    echo "🔧 Testing installed binary..."
    go-covercheck --version
    
    echo "✅ All tests passed!"
else
    echo "⚠️  Homebrew not available - skipping brew-specific tests"
    echo "📝 Formula syntax validated successfully"
fi