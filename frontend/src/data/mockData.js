// ABOUTME: Mock data for TAS Capacity Analyzer dashboard
// ABOUTME: Used for demo mode and development testing

export const mockData = {
  cells: [
    { id: 'cell-01', name: 'diego_cell/0', memory_mb: 16384, allocated_mb: 12288, used_mb: 9830, cpu_percent: 45, isolation_segment: 'default' },
    { id: 'cell-02', name: 'diego_cell/1', memory_mb: 16384, allocated_mb: 14336, used_mb: 11200, cpu_percent: 62, isolation_segment: 'default' },
    { id: 'cell-03', name: 'diego_cell/2', memory_mb: 16384, allocated_mb: 8192, used_mb: 6400, cpu_percent: 28, isolation_segment: 'default' },
    { id: 'cell-04', name: 'diego_cell/3', memory_mb: 32768, allocated_mb: 24576, used_mb: 19660, cpu_percent: 55, isolation_segment: 'production' },
    { id: 'cell-05', name: 'diego_cell/4', memory_mb: 32768, allocated_mb: 28672, used_mb: 22100, cpu_percent: 71, isolation_segment: 'production' },
    { id: 'cell-06', name: 'diego_cell/5', memory_mb: 8192, allocated_mb: 6144, used_mb: 4800, cpu_percent: 38, isolation_segment: 'development' },
  ],
  apps: [
    { name: 'api-gateway', instances: 4, requested_mb: 1024, actual_mb: 780, isolation_segment: 'production' },
    { name: 'auth-service', instances: 3, requested_mb: 512, actual_mb: 420, isolation_segment: 'production' },
    { name: 'payment-processor', instances: 2, requested_mb: 2048, actual_mb: 1650, isolation_segment: 'production' },
    { name: 'web-ui', instances: 6, requested_mb: 768, actual_mb: 580, isolation_segment: 'default' },
    { name: 'background-jobs', instances: 2, requested_mb: 1536, actual_mb: 980, isolation_segment: 'default' },
    { name: 'analytics-engine', instances: 1, requested_mb: 4096, actual_mb: 3200, isolation_segment: 'production' },
    { name: 'notification-service', instances: 3, requested_mb: 512, actual_mb: 380, isolation_segment: 'default' },
    { name: 'dev-app-1', instances: 1, requested_mb: 1024, actual_mb: 450, isolation_segment: 'development' },
    { name: 'dev-app-2', instances: 1, requested_mb: 512, actual_mb: 280, isolation_segment: 'development' },
  ]
};
