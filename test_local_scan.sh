#!/bin/bash

# Test local scanning functionality by using the discovery and detection packages directly
# This demonstrates the core PI detection capability

echo "🧪 Testing PI Scanner Local Capability"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo

# Build the test
go run -mod=mod test_local_demo.go /tmp/test-bank-repo
