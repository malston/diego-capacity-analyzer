# TAS Capacity Analyzer

A professional dashboard for analyzing Tanzu Application Service (TAS) / Diego cell capacity, density optimization, and right-sizing recommendations.

![TAS Capacity Analyzer](https://img.shields.io/badge/version-1.0.0-blue.svg)
![React](https://img.shields.io/badge/react-18.2-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## Features

- **Real-time Capacity Monitoring** - Track diego cell memory, CPU, and utilization across all cells
- **Isolation Segment Filtering** - View metrics by isolation segment (default, production, development)
- **What-If Scenario Modeling** - Simulate memory overcommit changes to see potential capacity gains
- **Right-Sizing Recommendations** - Identify over-provisioned apps with specific memory recommendations
- **Interactive Visualizations** - Bar charts, pie charts, and detailed tables with live data
- **Professional UI** - Dark theme with technical typography and smooth animations

## Getting Started

### Prerequisites

- Node.js 18+ and npm
- Access to a TAS/Cloud Foundry environment (for real data integration)

### Installation

1. Install dependencies:
```bash
npm install
```

2. Start the development server:
```bash
npm run dev
```

The app will open automatically at `http://localhost:3000`

### Build for Production

```bash
npm run build
```

The production build will be created in the `dist/` directory.

## Current State

The dashboard currently uses **mock data** to demonstrate functionality. The mock data includes:
- 6 diego cells with varying memory sizes (8GB, 16GB, 32GB)
- 3 isolation segments (default, production, development)
- 9 sample applications with realistic memory profiles

## Integrating with Real CF API

To connect to real TAS/Cloud Foundry data, you'll need to replace the `mockData` object in `src/TASCapacityAnalyzer.jsx` with actual CF API calls.

### Required CF API Endpoints

#### 1. Get Diego Cell Information

```javascript
// Get isolation segments
const segments = await fetch('https://api.sys.YOUR-DOMAIN.com/v3/isolation_segments', {
  headers: {
    'Authorization': `Bearer ${cfToken}`,
  }
});

// Get cell capacity from BOSH or CF metrics
// Note: Cell capacity is typically retrieved via BOSH or monitoring tools
```

#### 2. Get Application Data

```javascript
// Get all apps
const apps = await fetch('https://api.sys.YOUR-DOMAIN.com/v3/apps', {
  headers: {
    'Authorization': `Bearer ${cfToken}`,
  }
});

// Get processes (for memory requests)
const processes = await fetch('https://api.sys.YOUR-DOMAIN.com/v3/processes', {
  headers: {
    'Authorization': `Bearer ${cfToken}`,
  }
});

// Get process stats (for actual memory usage)
const stats = await fetch(`https://api.sys.YOUR-DOMAIN.com/v3/processes/${processGuid}/stats`, {
  headers: {
    'Authorization': `Bearer ${cfToken}`,
  }
});
```

#### 3. Authentication

You'll need to implement OAuth2 authentication with the CF API:

```javascript
const getToken = async (username, password) => {
  const response = await fetch('https://login.sys.YOUR-DOMAIN.com/oauth/token', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
      'Authorization': 'Basic Y2Y6', // cf client credentials
    },
    body: new URLSearchParams({
      grant_type: 'password',
      username,
      password,
    }),
  });
  
  const data = await response.json();
  return data.access_token;
};
```

### Example Integration Pattern

Create a new file `src/services/cfApi.js`:

```javascript
const CF_API_URL = process.env.REACT_APP_CF_API_URL || 'https://api.sys.example.com';

export const fetchCellData = async (token) => {
  // Fetch from CF metrics or BOSH
  // This is environment-specific - might come from:
  // - Tanzu Hub API
  // - AppMetrics API
  // - BOSH director API
  // - Custom metrics endpoint
  
  const response = await fetch(`${CF_API_URL}/v3/...`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  
  return await response.json();
};

export const fetchAppData = async (token) => {
  const apps = await fetch(`${CF_API_URL}/v3/apps`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  
  const processes = await fetch(`${CF_API_URL}/v3/processes`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  
  // Combine and transform data
  return transformAppData(await apps.json(), await processes.json());
};
```

Then update `TASCapacityAnalyzer.jsx`:

```javascript
import { fetchCellData, fetchAppData } from './services/cfApi';

const TASCapacityAnalyzer = () => {
  const [data, setData] = useState(mockData); // Start with mock
  const [loading, setLoading] = useState(false);
  
  const loadRealData = async () => {
    setLoading(true);
    try {
      const token = getCFToken(); // Implement this
      const [cells, apps] = await Promise.all([
        fetchCellData(token),
        fetchAppData(token),
      ]);
      
      setData({ cells, apps });
    } catch (error) {
      console.error('Failed to load CF data:', error);
    } finally {
      setLoading(false);
    }
  };
  
  // Call loadRealData() on mount or via button click
  // ...
};
```

### Environment Variables

Create a `.env` file in the project root:

```env
REACT_APP_CF_API_URL=https://api.sys.YOUR-DOMAIN.com
REACT_APP_CF_UAA_URL=https://login.sys.YOUR-DOMAIN.com
```

## Data Requirements

For the dashboard to work with real data, you need:

### Cell Data Structure:
```javascript
{
  id: string,           // Unique cell identifier
  name: string,         // Cell name (e.g., 'diego_cell/0')
  memory_mb: number,    // Total cell memory in MB
  allocated_mb: number, // Memory allocated to apps
  used_mb: number,      // Actual memory used by apps
  cpu_percent: number,  // CPU utilization percentage
  isolation_segment: string // Segment name
}
```

### App Data Structure:
```javascript
{
  name: string,         // App name
  instances: number,    // Number of instances
  requested_mb: number, // Memory requested per instance
  actual_mb: number,    // Actual memory used per instance
  isolation_segment: string // Segment name
}
```

## Recommended Monitoring Tools

To get real-time cell and app metrics, consider integrating with:

- **Tanzu Hub** - VMware Tanzu Hub for platform metrics
- **App Metrics** - TAS App Metrics for application-level data
- **BOSH** - Direct BOSH director queries for cell information
- **Prometheus/Grafana** - Custom metrics pipelines
- **CF Metrics Forwarder** - Stream metrics to external systems

## Architecture Options

### Option 1: Frontend-Only (Current)
- React app calls CF API directly
- User authentication via OAuth2
- Runs entirely in browser
- **Pro**: Simple deployment, no backend needed
- **Con**: Exposes CF credentials in browser

### Option 2: Backend Proxy
- Add Node.js/Go backend
- Backend handles CF API authentication
- Frontend calls backend API
- **Pro**: More secure, can cache data
- **Con**: Requires server deployment

### Option 3: CLI Tool + Static Report
- Convert to CLI tool (Go/Python)
- Generate static HTML reports
- Runs in customer environment
- **Pro**: Most secure, no hosting needed
- **Con**: Not real-time, manual execution

## Customization

### Adjusting Thresholds

In `TASCapacityAnalyzer.jsx`, modify:

```javascript
// Right-sizing threshold (currently 15% overhead)
.filter(app => app.overheadPercent > 15)

// Memory buffer for recommendations (currently 20%)
recommendedMb: Math.ceil(app.actual_mb * 1.2)

// Cell utilization warning levels
const status = utilizationPercent > 80 ? 'high' 
  : utilizationPercent > 60 ? 'medium' : 'low';
```

### Adding New Metrics

To add custom metrics:

1. Add to data structure in `mockData`
2. Calculate in the `metrics` useMemo hook
3. Add UI component to display

## Deployment

### Netlify/Vercel
```bash
npm run build
# Deploy dist/ folder
```

### Docker
```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --production
COPY . .
RUN npm run build
CMD ["npm", "run", "preview"]
```

### Static Hosting
```bash
npm run build
# Upload dist/ to S3, GCS, or any static host
```

## Contributing

This tool was built for TAS platform engineers to help optimize diego cell capacity and reduce infrastructure costs.

## License

MIT License - See LICENSE file for details

## Author

**Mark Alston**  
Broadcom/VMware Tanzu Platform Consultant

## Support

For issues or questions:
- Open an issue in this repository
- Contact your Broadcom/VMware representative

## Roadmap

- [ ] Direct CF API integration with authentication
- [ ] Historical trend analysis
- [ ] Cost estimation based on IaaS pricing
- [ ] Export reports to PDF/Excel
- [ ] Multi-foundation support
- [ ] BOSH director integration
- [ ] Slack/email alerts for capacity thresholds
- [ ] Terraform/Platform Automation recommendations
