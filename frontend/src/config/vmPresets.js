// frontend/src/config/vmPresets.js
// ABOUTME: VM size presets for Diego cell what-if analysis
// ABOUTME: Common configurations used in TAS deployments

export const VM_SIZE_PRESETS = [
  { label: '4 vCPU × 32 GB', cpu: 4, memoryGB: 32 },
  { label: '4 vCPU × 64 GB', cpu: 4, memoryGB: 64 },
  { label: '8 vCPU × 64 GB', cpu: 8, memoryGB: 64 },
  { label: '8 vCPU × 128 GB', cpu: 8, memoryGB: 128 },
  { label: 'Custom...', cpu: null, memoryGB: null },
];

export const DEFAULT_PRESET_INDEX = 0; // 4×32

export const formatCellSize = (cpu, memoryGB) => `${cpu}×${memoryGB}`;
