# Acceptance Testing: Security Fixes (Issues #68, #69, #70)

This document provides manual acceptance testing procedures for the critical security fixes in PR #80.

## Prerequisites

1. Backend server running: `make backend-run` from project root
2. Server listening on `http://localhost:8080`
3. `curl` and `jq` installed for testing

## Issue #68: DOS Protection (Request Body Size Limits)

### Objective

Verify that all POST endpoints reject payloads larger than 1MB with a generic error message.

### Test Setup

Generate a 1.5MB oversized payload:

```bash
python3 -c "import json; print(json.dumps({'data': 'x' * 1572864}))" > /tmp/oversized_payload.json
ls -la /tmp/oversized_payload.json  # Should show ~1.5MB
```

### Test Cases

#### Test 68.1: POST /api/v1/infrastructure/manual

```bash
curl -s -X POST http://localhost:8080/api/v1/infrastructure/manual \
  -H "Content-Type: application/json" \
  -d @/tmp/oversized_payload.json
```

**Expected Result:**

```json
{ "error": "Request body too large", "code": 400 }
```

#### Test 68.2: POST /api/v1/infrastructure/state

```bash
curl -s -X POST http://localhost:8080/api/v1/infrastructure/state \
  -H "Content-Type: application/json" \
  -d @/tmp/oversized_payload.json
```

**Expected Result:**

```json
{ "error": "Request body too large", "code": 400 }
```

#### Test 68.3: POST /api/v1/infrastructure/planning

```bash
curl -s -X POST http://localhost:8080/api/v1/infrastructure/planning \
  -H "Content-Type: application/json" \
  -d @/tmp/oversized_payload.json
```

**Expected Result:**

```json
{ "error": "Request body too large", "code": 400 }
```

#### Test 68.4: POST /api/v1/scenario/compare

```bash
curl -s -X POST http://localhost:8080/api/v1/scenario/compare \
  -H "Content-Type: application/json" \
  -d @/tmp/oversized_payload.json
```

**Expected Result:**

```json
{ "error": "Request body too large", "code": 400 }
```

### Verification Criteria

- [ ] All 4 endpoints return HTTP 400
- [ ] All 4 endpoints return exactly `{"error":"Request body too large","code":400}`
- [ ] No internal details (hostnames, IPs, paths) appear in responses

---

## Issue #69: Error Message Sanitization

### Objective

Verify that error responses contain only generic messages without exposing internal infrastructure details.

### Sensitive Patterns to Check For

Error messages must NOT contain:

- Internal hostnames (e.g., `vcenter.internal.corp`, `bosh.acme.local`)
- IP addresses (e.g., `10.0.0.5`, `192.168.1.100`)
- URLs with internal paths (e.g., `https://uaa.internal:8443/oauth/token`)
- Stack traces or Go error chains
- File paths (e.g., `/var/vcap/jobs/...`)
- Credential-related strings (e.g., `password`, `secret`, `token`)
- Connection details (e.g., `dial tcp`, `connection refused`, `x509:`)

### Test Cases

#### Test 69.1: vSphere Connection Error (simulated via invalid config)

When vSphere is misconfigured, the error should be generic:

```bash
curl -s http://localhost:8080/api/v1/infrastructure
```

**Expected Result (if vSphere unreachable):**

```json
{ "error": "Infrastructure service temporarily unavailable", "code": 503 }
```

**NOT acceptable:**

```json
{
  "error": "cannot connect to vcenter.internal.acme.com:443 - dial tcp: connection refused",
  "code": 503
}
```

#### Test 69.2: CF API Authentication Error

When CF API credentials are invalid:

```bash
curl -s http://localhost:8080/api/v1/dashboard
```

**Expected Result (if CF auth fails):**

```json
{ "error": "Authentication service temporarily unavailable", "code": 500 }
```

**NOT acceptable:**

```json
{
  "error": "failed to authenticate: Post \"https://uaa.sys.cf.local/oauth/token\": x509: certificate signed by unknown authority",
  "code": 500
}
```

#### Test 69.3: Invalid JSON Input

```bash
curl -s -X POST http://localhost:8080/api/v1/infrastructure/manual \
  -H "Content-Type: application/json" \
  -d 'not valid json'
```

**Expected Result:**

```json
{ "error": "Invalid JSON", "code": 400 }
```

#### Test 69.4: Valid Request Processing

Verify valid requests still work correctly:

```bash
curl -s -X POST http://localhost:8080/api/v1/infrastructure/manual \
  -H "Content-Type: application/json" \
  -d '{
    "clusters": [{
      "name": "test-cluster",
      "hosts": [{"name": "host1", "memory_gb": 512}],
      "diego_cells": 2,
      "cell_memory_gb": 32
    }]
  }'
```

**Expected Result:** HTTP 200 with infrastructure state JSON (not an error)

### Verification Criteria

- [ ] No error response contains internal hostnames or IP addresses
- [ ] No error response contains URLs with internal paths
- [ ] No error response contains stack traces or raw Go errors
- [ ] No error response contains file system paths
- [ ] No error response exposes authentication details
- [ ] Valid requests continue to work normally

---

## Issue #70: SSH Key Path Traversal Prevention

### Objective

Verify that the BOSH client rejects SSH key paths containing path traversal sequences.

### Test Cases

This vulnerability is in the BOSH proxy configuration code. It cannot be tested via HTTP endpoints directly. Instead, verify via unit tests:

```bash
cd backend && go test ./services/... -v -run "TestValidateSSHKeyPath|TestPathTraversal"
```

**Expected Result:** All path traversal tests pass, including:

- Rejection of paths containing `..`
- Rejection of paths pointing to non-existent files
- Rejection of paths pointing to directories
- Acceptance of valid regular file paths

### Manual Verification

If you have access to modify environment variables, you can verify the validation logs:

1. Set a malicious path:

```bash
export BOSH_ALL_PROXY="ssh+socks5://user@jumpbox?private-key=/etc/../../../etc/passwd"
```

2. Attempt to connect (the connection will fail, but check server logs):

```bash
curl -s http://localhost:8080/api/v1/infrastructure
```

3. Check server logs for validation rejection (should see path traversal detection)

### Verification Criteria

- [ ] Unit tests for ValidateSSHKeyPath pass
- [ ] Paths with `..` are rejected
- [ ] Non-existent paths are rejected
- [ ] Directory paths are rejected
- [ ] Valid file paths are accepted

---

## Automated Test Script

Run all acceptance tests with a single script:

```bash
#!/usr/bin/env bash
set -e

echo "=== Security Acceptance Tests ==="
echo ""

# Setup
echo "Generating oversized payload..."
python3 -c "import json; print(json.dumps({'data': 'x' * 1572864}))" > /tmp/oversized_payload.json

PASS=0
FAIL=0

check_response() {
    local name="$1"
    local expected="$2"
    local actual="$3"

    if [[ "$actual" == *"$expected"* ]]; then
        echo "✅ PASS: $name"
        ((PASS++))
    else
        echo "❌ FAIL: $name"
        echo "   Expected: $expected"
        echo "   Actual: $actual"
        ((FAIL++))
    fi
}

check_no_sensitive() {
    local name="$1"
    local response="$2"
    local patterns=("vcenter" "vsphere" "10.0." "192.168." ".internal" ".local" "password" "secret" "token" "dial tcp" "x509:" "connection refused")

    for pattern in "${patterns[@]}"; do
        if [[ "${response,,}" == *"${pattern,,}"* ]]; then
            echo "❌ FAIL: $name - contains sensitive pattern: $pattern"
            echo "   Response: $response"
            ((FAIL++))
            return
        fi
    done
    echo "✅ PASS: $name - no sensitive data exposed"
    ((PASS++))
}

echo ""
echo "=== Issue #68: DOS Protection Tests ==="

r1=$(curl -s -X POST http://localhost:8080/api/v1/infrastructure/manual -H "Content-Type: application/json" -d @/tmp/oversized_payload.json)
check_response "68.1 /infrastructure/manual oversized" "Request body too large" "$r1"

r2=$(curl -s -X POST http://localhost:8080/api/v1/infrastructure/state -H "Content-Type: application/json" -d @/tmp/oversized_payload.json)
check_response "68.2 /infrastructure/state oversized" "Request body too large" "$r2"

r3=$(curl -s -X POST http://localhost:8080/api/v1/infrastructure/planning -H "Content-Type: application/json" -d @/tmp/oversized_payload.json)
check_response "68.3 /infrastructure/planning oversized" "Request body too large" "$r3"

r4=$(curl -s -X POST http://localhost:8080/api/v1/scenario/compare -H "Content-Type: application/json" -d @/tmp/oversized_payload.json)
check_response "68.4 /scenario/compare oversized" "Request body too large" "$r4"

echo ""
echo "=== Issue #69: Error Sanitization Tests ==="

check_no_sensitive "68.1 response sanitized" "$r1"
check_no_sensitive "68.2 response sanitized" "$r2"
check_no_sensitive "68.3 response sanitized" "$r3"
check_no_sensitive "68.4 response sanitized" "$r4"

r5=$(curl -s -X POST http://localhost:8080/api/v1/infrastructure/manual -H "Content-Type: application/json" -d 'invalid json')
check_response "69.3 Invalid JSON" "Invalid JSON" "$r5"
check_no_sensitive "69.3 response sanitized" "$r5"

echo ""
echo "=== Summary ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"

if [[ $FAIL -gt 0 ]]; then
    exit 1
fi
echo "All acceptance tests passed!"
```

---

## Sign-Off

| Test Area                     | Tester | Date | Result |
| ----------------------------- | ------ | ---- | ------ |
| Issue #68: DOS Protection     |        |      |        |
| Issue #69: Error Sanitization |        |      |        |
| Issue #70: Path Traversal     |        |      |        |

**Notes:**
