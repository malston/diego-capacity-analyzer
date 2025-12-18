# CORS Troubleshooting Guide

## The Problem

When you click "Using Mock Data" to switch to live CF data, you're seeing:
```
Error loading CF data
Failed to fetch
```

This is almost certainly a **CORS (Cross-Origin Resource Sharing)** error.

## What's Happening

Your browser is blocking the request because:
- Your React app runs on `http://localhost:3000`
- Your CF API is at `https://api.sys.YOUR-DOMAIN.com`
- The CF API doesn't allow requests from `localhost`

## Check the Actual Error

### 1. Open Browser DevTools
Press **F12** (or right-click → Inspect)

### 2. Go to Console Tab
You should see an error like:
```
Access to fetch at 'https://api.sys.example.com/v3/apps' from origin 
'http://localhost:3000' has been blocked by CORS policy: No 
'Access-Control-Allow-Origin' header is present on the requested resource.
```

## Test Your Connection

I've added a **"Test Connection"** button:
1. Stay in mock data mode
2. Click the new **"Test Connection"** button
3. This tests if you can reach the CF API at all

**If test fails**: Your `.env` file might have the wrong URLs or CF API is unreachable
**If test succeeds but data load fails**: Definitely CORS

## Solutions

### Solution 1: Backend Proxy (Recommended for Production)

Create a simple Node.js proxy that forwards requests to CF API.

**Quick Express Proxy** (`proxy-server.js`):
```javascript
const express = require('express');
const cors = require('cors');
const fetch = require('node-fetch');

const app = express();
app.use(cors());
app.use(express.json());

const CF_API_URL = 'https://api.sys.YOUR-DOMAIN.com';

app.all('/api/*', async (req, res) => {
  const cfUrl = `${CF_API_URL}${req.path.replace('/api', '')}`;
  
  try {
    const response = await fetch(cfUrl, {
      method: req.method,
      headers: {
        ...req.headers,
        host: undefined,
      },
      body: req.method !== 'GET' ? JSON.stringify(req.body) : undefined,
    });
    
    const data = await response.json();
    res.json(data);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

app.listen(3001, () => console.log('Proxy running on http://localhost:3001'));
```

Then update your `.env`:
```env
VITE_CF_API_URL=http://localhost:3001/api
```

### Solution 2: Configure CF to Allow Localhost

If you have admin access to your CF deployment, configure HAProxy or the API endpoint to allow CORS from localhost.

**In CF deployment manifest** (gorouter or HAProxy config):
```yaml
properties:
  router:
    cors:
      allowed_origins:
      - http://localhost:3000
      - http://127.0.0.1:3000
```

Then redeploy CF.

### Solution 3: Browser Extension (Dev Only)

Install a CORS-unblocking browser extension:
- **Chrome**: "Allow CORS: Access-Control-Allow-Origin"
- **Firefox**: "CORS Everywhere"

⚠️ **Security Warning**: Only use for development! Don't leave these enabled.

### Solution 4: Deploy App to Same Domain

Deploy your React app to the same domain as CF:
- `https://capacity-analyzer.sys.YOUR-DOMAIN.com` → Frontend
- `https://api.sys.YOUR-DOMAIN.com` → CF API

No CORS issues since it's same domain!

### Solution 5: Use CF CLI as Proxy (Testing Only)

For quick testing, you can use the CF CLI to create a tunnel:

```bash
# Install cf-ssh plugin
cf add-plugin-repo CF-Community https://plugins.cloudfoundry.org
cf install-plugin -r CF-Community "Open"

# Create a local proxy through CF
cf ssh-proxy -app-name your-app
```

## Quick Test with curl

Test if CF API is reachable:

```bash
# Test without auth (should work)
curl https://api.sys.YOUR-DOMAIN.com/v3/info

# Test with auth
cf oauth-token  # Get your token
curl -H "Authorization: Bearer YOUR_TOKEN" https://api.sys.YOUR-DOMAIN.com/v3/apps
```

If curl works but browser doesn't → CORS issue
If curl fails → Network/config issue

## Development Workflow

For development, I recommend:

**Quick & Dirty**: Use CORS browser extension
**Better**: Run the Express proxy (5 minutes to set up)
**Production**: Deploy to same domain or implement proper backend

## My Recommendation

Since you're a consultant showing this to customers:

1. **For demos**: Keep using mock data, it looks great!
2. **For real analysis**: Set up the Express proxy (I can help you build this)
3. **For production**: Build a proper backend service with:
   - Authentication handling
   - Token management
   - Rate limiting
   - Caching
   - Error handling

Want me to create the proxy server for you?
