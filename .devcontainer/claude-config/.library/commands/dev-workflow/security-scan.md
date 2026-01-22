---
description: Comprehensive security audit of recent changes
---
# Security Scan
1. **Secret Detection:**
   ```bash
   git diff --cached | grep -E '(api_key|password|secret|token|aws_access)' || echo "✓ No secrets detected"
