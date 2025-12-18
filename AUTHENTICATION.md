# CF API Authentication - Setup Guide

This document explains how to set up and use the Cloud Foundry API authentication in the TAS Capacity Analyzer.

## Quick Start

### 1. Configure Environment Variables

Copy the example environment file and configure it:

```bash
cp .env.example .env
```

Edit `.env` with your CF environment details:

```env
VITE_CF_API_URL=https://api.sys.YOUR-DOMAIN.com
VITE_CF_UAA_URL=https://login.sys.YOUR-DOMAIN.com
```

### 2. Install Dependencies

```bash
npm install
```

### 3. Run the Application

```bash
npm run dev
```

The app will open at `http://localhost:3000` and show the login screen.

### 4. Login

Enter your Cloud Foundry credentials:
- **Username**: Your CF username (usually an email)
- **Password**: Your CF password

The app will authenticate with CF UAA and store the access token in your browser session.

## How Authentication Works

### OAuth2 Flow

The app uses the OAuth2 **password grant** flow to authenticate with Cloud Foundry UAA:

1. User enters credentials in the login form
2. App sends credentials to CF UAA `/oauth/token` endpoint
3. UAA validates credentials and returns an access token
4. Access token is stored in browser `sessionStorage`
5. Token is used for all subsequent CF API calls
6. Token automatically refreshes when it expires

### Token Management

- **Access Token**: Valid for ~12 hours (configurable in CF)
- **Refresh Token**: Used to get a new access token without re-login
- **Storage**: Tokens stored in browser `sessionStorage` (cleared on tab close)
- **Auto-Refresh**: Tokens automatically refresh 60 seconds before expiry

### Security Considerations

**Current Implementation (Client-Side Auth)**:
- ✅ Suitable for internal tools and demos
- ✅ No backend required
- ❌ Credentials flow through browser
- ❌ Token visible in browser storage

**For Production Use**, consider:
- **Backend Proxy**: Add a backend service to handle authentication
- **Authorization Code Flow**: More secure OAuth2 flow
- **Session Management**: Server-side session storage
- **Rate Limiting**: Protect against brute force attacks

## Authentication Service API

### cfAuth Service

Located in `src/services/cfAuth.js`

```javascript
import { cfAuth } from './services/cfAuth';

// Login
await cfAuth.login(username, password);

// Check if authenticated
const isAuth = cfAuth.isAuthenticated();

// Get current token (auto-refreshes if needed)
const token = await cfAuth.getToken();

// Get user info from token
const user = cfAuth.getUserInfo();

// Logout
cfAuth.logout();
```

### cfApi Service

Located in `src/services/cfApi.js`

```javascript
import { cfApi } from './services/cfApi';

// Get all applications
const apps = await cfApi.getApplications();

// Get apps with isolation segment names
const appsWithSegments = await cfApi.getAppsWithSegments();

// Get isolation segments
const segments = await cfApi.getIsolationSegments();

// Get CF info (no auth required)
const info = await cfApi.getInfo();
```

## React Components

### AuthProvider Context

Wrap your app with `AuthProvider` to enable authentication:

```javascript
import { AuthProvider } from './contexts/AuthContext';

function App() {
  return (
    <AuthProvider>
      <YourComponents />
    </AuthProvider>
  );
}
```

### useAuth Hook

Access authentication state in components:

```javascript
import { useAuth } from './contexts/AuthContext';

function MyComponent() {
  const { isAuthenticated, user, login, logout, loading, error } = useAuth();
  
  if (loading) return <Loading />;
  if (!isAuthenticated) return <Login />;
  
  return <div>Hello {user.username}</div>;
}
```

## Troubleshooting

### CORS Errors

If you see CORS errors in the browser console:

```
Access to fetch at 'https://api.sys.example.com/v3/apps' from origin 'http://localhost:3000' has been blocked by CORS policy
```

**Solutions**:
1. **Use CF CLI Proxy**: Run `cf api` with the API URL and use the CLI as a proxy
2. **Add CORS Headers**: Configure CF/HAProxy to allow localhost in CORS headers
3. **Use Backend Proxy**: Create a backend service to proxy CF API requests

### Authentication Failures

If authentication fails:

1. **Check CF credentials**: Verify with `cf login`
2. **Check API URLs**: Ensure VITE_CF_API_URL and VITE_CF_UAA_URL are correct
3. **Check network**: Ensure you can reach the CF API from your machine
4. **Check browser console**: Look for detailed error messages

### Token Expiration

If you see "Not authenticated" errors after being logged in:

- Token may have expired
- Refresh token may be invalid
- Click "Refresh" or logout and login again

### Missing Cell Data

The CF API doesn't directly expose diego cell capacity information. To get cell data:

1. **BOSH API**: Query BOSH director for VM information (requires BOSH credentials)
2. **Healthwatch**: Use VMware Tanzu Healthwatch API
3. **App Metrics**: Use CF App Metrics API
4. **Custom Exporter**: Create a custom metrics exporter

Example BOSH query (server-side):

```javascript
const cells = await cfApi.getDiegoCellsFromBOSH(
  'https://bosh.example.com:25555',
  boshToken,
  'cf-deployment'
);
```

## API Permissions

### Required CF Scopes

The user account needs these OAuth scopes:

- `cloud_controller.read` - Read apps, spaces, orgs
- `cloud_controller.admin` - Admin access (for cell data)

### Check User Scopes

```javascript
const user = cfAuth.getUserInfo();
console.log('User scopes:', user.scopes);
```

## Data Source Toggle

The dashboard includes a toggle between mock data and live CF data:

- **Mock Data Mode**: Uses fake data for demo purposes
- **Live CF Data Mode**: Fetches real data from CF API

Click the "Using Mock Data" / "Live CF Data" button to toggle.

## Next Steps

### Get Diego Cell Data

Since CF API doesn't expose cell data, you have options:

1. **Integrate with BOSH**: See `src/services/cfApi.js` → `getDiegoCellsFromBOSH()`
2. **Use Healthwatch**: Query Healthwatch metrics API
3. **Custom Metrics**: Create a custom metrics endpoint
4. **Keep Mock Cells**: Use real app data with mock cell data

### Add More Features

- Historical data tracking
- Cost estimation based on cloud provider
- Email/Slack alerts for capacity thresholds
- Export reports to PDF/Excel
- Multi-foundation support

### Deploy to Production

See main README.md for deployment options.

## Security Best Practices

1. **Never commit .env files** to git
2. **Use HTTPS** for all CF API communication
3. **Rotate credentials** regularly
4. **Limit user permissions** to read-only where possible
5. **Consider backend proxy** for production deployments
6. **Enable audit logging** for authentication events

## Support

For issues:
- Check browser console for errors
- Verify CF API connectivity with `cf login`
- Review CF UAA logs for authentication failures
- Open an issue in this repository
