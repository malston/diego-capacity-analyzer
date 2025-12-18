# Adding Localhost to CF CORS Allowed Origins

This guide shows how to configure Cloud Foundry to allow CORS requests from localhost, enabling your React app to connect directly to the CF API.

## Overview

You need to modify the **gorouter** configuration in your CF deployment to add localhost to the allowed CORS origins.

## Method 1: Ops Manager (TAS)

### For Tanzu Application Service via Ops Manager

1. **Login to Ops Manager**
   ```
   https://pcf.YOUR-DOMAIN.com
   ```

2. **Navigate to TAS Tile**
   - Click on the "Tanzu Application Service" (or "Pivotal Application Service") tile

3. **Go to Networking Section**
   - Click on **"Networking"** in the left sidebar

4. **Find CORS Settings**
   - Scroll to **"CORS allowed origins"** or **"Cross-Origin Resource Sharing"** section

5. **Add Localhost**
   
   Add these entries (one per line):
   ```
   http://localhost:3000
   http://127.0.0.1:3000
   ```

   If you want to allow all ports during development:
   ```
   http://localhost:*
   http://127.0.0.1:*
   ```

6. **Save and Apply Changes**
   - Click **"Save"**
   - Return to Ops Manager dashboard
   - Click **"Review Pending Changes"**
   - Select only the TAS tile (to speed up apply)
   - Click **"Apply Changes"**

7. **Wait for Deployment**
   - This will restart the gorouter instances
   - Takes ~10-20 minutes typically

### Verify the Configuration

Once deployed, test with curl:

```bash
# This should return CORS headers
curl -I -X OPTIONS \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET" \
  https://api.sys.YOUR-DOMAIN.com/v3/info
```

Look for:
```
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
```

## Method 2: BOSH Deployment (Open Source CF)

### For CF deployed via BOSH

1. **Find Your CF Deployment**
   ```bash
   bosh deployments
   # Note the cf deployment name, usually "cf"
   ```

2. **Download Current Manifest**
   ```bash
   bosh -d cf manifest > cf-manifest.yml
   ```

3. **Edit the Manifest**
   
   Find the `router` job properties and add CORS configuration:

   ```yaml
   instance_groups:
   - name: router
     jobs:
     - name: gorouter
       properties:
         router:
           cors_allowed_origins:
           - "http://localhost:3000"
           - "http://127.0.0.1:3000"
           # Or for all localhost ports:
           - "http://localhost:*"
           - "http://127.0.0.1:*"
   ```

   **Full example with context:**
   ```yaml
   instance_groups:
   - name: router
     jobs:
     - name: gorouter
       properties:
         router:
           status:
             port: 8080
             user: router_status
             password: ((router_status_password))
           # Add this section
           cors_allowed_origins:
           - "http://localhost:3000"
           - "http://127.0.0.1:3000"
   ```

4. **Deploy the Changes**
   ```bash
   bosh -d cf deploy cf-manifest.yml
   ```

5. **Monitor Deployment**
   ```bash
   bosh -d cf instances
   # Watch until router instances are running
   ```

## Method 3: Operations Files (Cloud Config)

### Using Ops Files with BOSH

Create an ops file: `ops-files/enable-localhost-cors.yml`

```yaml
---
# Enable CORS for localhost development

- type: replace
  path: /instance_groups/name=router/jobs/name=gorouter/properties/router/cors_allowed_origins?
  value:
  - "http://localhost:3000"
  - "http://127.0.0.1:3000"
```

Deploy with the ops file:
```bash
bosh -d cf deploy cf-deployment.yml \
  -o ops-files/enable-localhost-cors.yml \
  -o other-ops-files.yml
```

## Method 4: Platform Automation (for automated deployments)

### If using Platform Automation Toolkit

1. **Update your vars file** (`cf-vars.yml`):
   ```yaml
   router_cors_allowed_origins:
   - "http://localhost:3000"
   - "http://127.0.0.1:3000"
   ```

2. **Update your config** (`cf-config.yml`):
   ```yaml
   product-properties:
     .properties.router_cors_allowed_origins:
       value:
       - "http://localhost:3000"
       - "http://127.0.0.1:3000"
   ```

3. **Run your pipeline** to apply changes

## Method 5: cf-deployment with Custom Operations

### For standard cf-deployment

1. **Create custom ops file**: `enable-localhost-cors.yml`
   ```yaml
   - type: replace
     path: /instance_groups/name=router/jobs/name=gorouter/properties/router/cors_allowed_origins?
     value:
     - http://localhost:3000
     - http://127.0.0.1:3000
   ```

2. **Include in your deployment script**:
   ```bash
   bosh -d cf deploy cf-deployment/cf-deployment.yml \
     -o cf-deployment/operations/use-compiled-releases.yml \
     -o enable-localhost-cors.yml \
     --vars-store cf-vars.yml
   ```

## Verification Steps

### 1. Check Gorouter Configuration

SSH into a router VM:
```bash
bosh -d cf ssh router/0
```

Check the gorouter config:
```bash
cat /var/vcap/jobs/gorouter/config/gorouter.yml | grep -A 5 cors
```

Should show:
```yaml
cors_allowed_origins:
- http://localhost:3000
- http://127.0.0.1:3000
```

### 2. Test with Browser

Open your React app at `http://localhost:3000` and try clicking "Using Mock Data" to switch to live data.

### 3. Test with curl

```bash
# Preflight request (OPTIONS)
curl -X OPTIONS \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET" \
  -H "Access-Control-Request-Headers: Authorization" \
  https://api.sys.YOUR-DOMAIN.com/v3/info \
  -v
```

Look for these headers in the response:
```
< Access-Control-Allow-Origin: http://localhost:3000
< Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS, PATCH
< Access-Control-Allow-Headers: Authorization, Content-Type
```

### 4. Test with JavaScript

In browser console (F12):
```javascript
fetch('https://api.sys.YOUR-DOMAIN.com/v3/info', {
  headers: {
    'Authorization': 'Bearer YOUR_TOKEN'
  }
})
.then(r => r.json())
.then(d => console.log('Success!', d))
.catch(e => console.error('Failed:', e));
```

## Important Security Notes

### ⚠️ Development Only
Adding localhost to CORS origins is typically **only for development**. 

### 🔒 Production Recommendations

For production, you should:

1. **Deploy frontend to proper domain**:
   ```yaml
   cors_allowed_origins:
   - "https://capacity-analyzer.sys.YOUR-DOMAIN.com"
   ```

2. **Use specific origins** (not wildcards):
   ```yaml
   # BAD - Too permissive
   cors_allowed_origins:
   - "*"
   
   # GOOD - Specific domains
   cors_allowed_origins:
   - "https://capacity-analyzer.sys.example.com"
   - "https://dashboard.example.com"
   ```

3. **Remove localhost** when going to production:
   ```yaml
   # Remove these for production:
   # - "http://localhost:*"
   # - "http://127.0.0.1:*"
   ```

## Troubleshooting

### Changes Not Taking Effect

1. **Clear browser cache** (Ctrl+Shift+Delete)
2. **Hard refresh** the page (Ctrl+Shift+R)
3. **Check gorouter logs**:
   ```bash
   bosh -d cf logs router/0 --recent
   ```

### Still Getting CORS Errors

1. **Verify configuration was applied**:
   ```bash
   bosh -d cf ssh router/0
   cat /var/vcap/jobs/gorouter/config/gorouter.yml
   ```

2. **Check if gorouter restarted**:
   ```bash
   bosh -d cf instances --ps
   # Look for router process uptime
   ```

3. **Test directly with curl** (see verification steps above)

4. **Check for HAProxy or load balancer**:
   - Some CF deployments have HAProxy or external load balancers
   - These might also need CORS configuration
   - Check with your infrastructure team

### HAProxy Configuration (if applicable)

If your CF uses HAProxy, you may also need to configure it:

```yaml
instance_groups:
- name: haproxy
  jobs:
  - name: haproxy
    properties:
      ha_proxy:
        cors_allowed_origins:
        - "http://localhost:3000"
        - "http://127.0.0.1:3000"
```

## Alternative: Test Without CORS Changes

If you can't modify CF configuration right now, use the proxy server approach instead:

See `CORS_TROUBLESHOOTING.md` → Solution 1: Backend Proxy

## Rollback

If you need to remove localhost CORS:

### Ops Manager
1. Go back to TAS Tile → Networking
2. Remove the localhost entries
3. Apply Changes

### BOSH
1. Remove the `cors_allowed_origins` section from manifest
2. Redeploy: `bosh -d cf deploy cf-manifest.yml`

## Additional Resources

- **CF Documentation**: https://docs.cloudfoundry.org/concepts/http-routing.html
- **Gorouter Source**: https://github.com/cloudfoundry/gorouter
- **CORS Specification**: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS

## Quick Reference Card

```bash
# Check current config
bosh -d cf ssh router/0
cat /var/vcap/jobs/gorouter/config/gorouter.yml | grep -A 3 cors

# Test CORS with curl
curl -I -X OPTIONS \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET" \
  https://api.sys.YOUR-DOMAIN.com/v3/info

# View router logs
bosh -d cf logs router/0 --recent | grep CORS
```

## Summary Checklist

- [ ] Identified CF deployment method (Ops Manager / BOSH / Platform Automation)
- [ ] Added localhost origins to gorouter configuration
- [ ] Applied/deployed changes
- [ ] Verified gorouter config on VM
- [ ] Tested with curl
- [ ] Tested with browser
- [ ] Documented the change for your team
- [ ] Planned to remove localhost for production

---

**Need help?** Check the exact CF version and deployment method, and I can provide more specific guidance!
