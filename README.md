# TAS Capacity Analyzer

A professional dashboard for analyzing Tanzu Application Service (TAS) / Diego cell capacity, density optimization, and right-sizing recommendations.

![TAS Capacity Analyzer](https://img.shields.io/badge/version-1.2.0-blue.svg)
![React](https://img.shields.io/badge/react-18.2-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## Architecture

- **Backend:** Go HTTP service (CF app) - proxies CF API, queries BOSH for cell metrics
- **Frontend:** React SPA (CF app with static buildpack) - dashboard UI

## Features

- **Real-time Capacity Monitoring** - Track diego cell memory, CPU, and utilization across all cells
- **Isolation Segment Filtering** - View metrics by isolation segment (default, production, development)
- **What-If Scenario Modeling** - Simulate memory overcommit changes to see potential capacity gains
- **Right-Sizing Recommendations** - Identify over-provisioned apps with specific memory recommendations
- **Interactive Visualizations** - Bar charts, pie charts, and detailed tables with live data
- **Professional UI** - Dark theme with technical typography and smooth animations

## Quick Start (Local Development)

### Backend

```bash
cd backend
export CF_API_URL=https://api.sys.example.com
export CF_USERNAME=admin
export CF_PASSWORD=secret
go run main.go
```

### Frontend

```bash
cd frontend
echo "VITE_API_URL=http://localhost:8080" > .env
npm install
npm run dev
```

## Deployment to Cloud Foundry

For complete deployment instructions, see **[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)**.

### Quick Deploy

1. **Deploy Backend**

```bash
cd backend
# Update manifest.yml with your values
cf push
cf set-env capacity-backend CF_USERNAME admin
cf set-env capacity-backend CF_PASSWORD <password>
# ... set other env vars (see deployment guide)
cf restage capacity-backend
```

1. **Deploy Frontend**

```bash
cd frontend
# Update .env with backend URL
echo "VITE_API_URL=https://capacity-backend.apps.example.com" > .env
npm run build
cf push
```

1. **Access UI**

```bash
cf app capacity-ui  # Get URL
open https://capacity-ui.apps.example.com
```

## Architecture Diagram

```console
Frontend (React)  →  Backend (Go)  →  CF API v3
                             ↓
                          BOSH API (Diego cells)
                             ↓
                      In-Memory Cache (5min TTL)
```

## Project Structure

```sh
├── backend/              # Go HTTP service
│   ├── main.go
│   ├── config/          # Configuration loader
│   ├── models/          # Data models
│   ├── cache/           # In-memory cache
│   ├── services/        # CF/BOSH API clients
│   ├── handlers/        # HTTP handlers
│   └── manifest.yml     # CF deployment manifest
│
├── frontend/            # React SPA
│   ├── src/
│   ├── index.html
│   ├── package.json
│   └── manifest.yml     # CF deployment manifest
│
└── docs/                # Documentation
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
