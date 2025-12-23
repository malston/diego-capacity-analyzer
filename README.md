# TAS Capacity Analyzer

A professional dashboard for analyzing Tanzu Application Service (TAS) / Diego cell capacity, density optimization, and right-sizing recommendations.

![TAS Capacity Analyzer](https://img.shields.io/badge/version-1.3.2-blue.svg)
![React](https://img.shields.io/badge/react-18.2-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## Features

- **Real-time Capacity Monitoring** - Track diego cell memory, CPU, and utilization across all cells
- **Isolation Segment Filtering** - View metrics by isolation segment (default, production, development)
- **What-If Scenario Modeling** - Simulate memory overcommit changes to see potential capacity gains
- **Right-Sizing Recommendations** - Identify over-provisioned apps with specific memory recommendations

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

## Documentation

- **[UI Guide](docs/UI-GUIDE.md)** - Understanding the dashboard metrics and visualizations
- **[Deployment Guide](docs/DEPLOYMENT.md)** - Complete deployment instructions for Cloud Foundry

## Project Structure

```sh
├── backend/             # Go HTTP service
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
