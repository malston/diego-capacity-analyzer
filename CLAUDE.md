# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TAS Capacity Analyzer is a React-based dashboard for analyzing Tanzu Application Service (TAS) / Diego cell capacity and density optimization. Currently uses mock data for demonstration purposes with the intent to integrate with real Cloud Foundry API endpoints.

## Development Commands

```bash
# Install dependencies
npm install

# Start development server (opens at http://localhost:3000)
npm run dev

# Build for production (outputs to dist/)
npm run build

# Preview production build
npm run preview

# Lint code
npm run lint
```

## Technology Stack

- **React 18.2** - UI framework
- **Vite 5** - Build tool and dev server
- **Tailwind CSS 3.3** - Styling (with PostCSS/Autoprefixer)
- **Recharts 2.10** - Data visualization (bar charts, pie charts, line charts)
- **Lucide React** - Icon library
- **Custom Fonts**: JetBrains Mono (monospace), Space Grotesk (titles)

## Architecture

### Component Structure

The application is a single-page app with one main component:

- `main.jsx` - React app entry point
- `App.jsx` - Root component wrapper
- `TASCapacityAnalyzer.jsx` - Main dashboard component containing all UI logic, state management, and calculations

### Data Model

**Mock Data Location**: `TASCapacityAnalyzer.jsx` - `mockData` object (lines 6-26)

**Cell Data Structure**:
```javascript
{
  id: string,              // Unique cell identifier
  name: string,            // Cell name (e.g., 'diego_cell/0')
  memory_mb: number,       // Total cell memory in MB
  allocated_mb: number,    // Memory allocated to apps
  used_mb: number,         // Actual memory used by apps
  cpu_percent: number,     // CPU utilization percentage
  isolation_segment: string // Segment name ('default', 'production', 'development')
}
```

**App Data Structure**:
```javascript
{
  name: string,            // App name
  instances: number,       // Number of instances
  requested_mb: number,    // Memory requested per instance
  actual_mb: number,       // Actual memory used per instance
  isolation_segment: string // Segment name
}
```

### Key Features Implementation

1. **Capacity Metrics** (lines 34-73) - Calculated via `useMemo` hook based on filtered cells/apps by isolation segment
2. **Right-Sizing Recommendations** (lines 76-95) - Apps with >15% memory overhead, sorted by potential savings
3. **What-If Scenario** (lines 284-327) - Memory overcommit ratio slider (1.0x-2.0x) for capacity planning
4. **Isolation Segment Filtering** - Dropdown to view metrics by segment or all segments combined

### Styling Approach

- **Dark theme** with gradient backgrounds and glassmorphic cards
- Custom CSS-in-JS via `<style>` tag in component (lines 119-198)
- Tailwind utility classes for layout and spacing
- Custom classes: `.metric-card`, `.cell-row`, `.progress-bar`, `.status-badge`, `.segment-chip`
- Animations: shimmer effect on progress bars, hover transitions on cards

## Future Integration Notes

### Cloud Foundry API Integration

When ready to replace mock data with real CF API:

1. **Create API service layer**: `src/services/cfApi.js` to handle:
   - OAuth2 authentication with CF UAA
   - Cell capacity fetching (via BOSH, Tanzu Hub, or CF metrics endpoints)
   - App/process data fetching (CF API v3)
   - Stats aggregation and transformation

2. **Environment variables** (`.env`):
   - `REACT_APP_CF_API_URL` - CF API endpoint (e.g., `https://api.sys.example.com`)
   - `REACT_APP_CF_UAA_URL` - UAA endpoint (e.g., `https://login.sys.example.com`)

3. **Required CF API v3 endpoints**:
   - `/v3/isolation_segments` - Get isolation segments
   - `/v3/apps` - Get all apps
   - `/v3/processes` - Get process memory requests
   - `/v3/processes/{guid}/stats` - Get actual memory usage
   - Cell capacity typically from BOSH or monitoring tools (Tanzu Hub, AppMetrics)

### Key Calculation Thresholds

These values may need tuning based on real platform data:

- **Right-sizing threshold**: 15% memory overhead (line 93)
- **Memory buffer recommendation**: 20% above actual usage (line 89)
- **Cell utilization warnings**: >80% high, >60% medium, ≤60% low (line 416)
- **Default overcommit ratio**: 1.0x (line 29)

## Development Notes

- All state management is local to `TASCapacityAnalyzer` component via React hooks
- Charts are responsive via Recharts' `ResponsiveContainer`
- Google Fonts loaded via CDN in component (line 120)
- Vite dev server configured to auto-open browser on port 3000
- No backend or API layer currently - runs entirely client-side
