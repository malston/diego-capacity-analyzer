// frontend/src/config/vmPresets.js
// ABOUTME: VM size presets for Diego cell what-if analysis
// ABOUTME: Common configurations used in TAS deployments

export const VM_SIZE_PRESETS = [
  { label: '4 vCPU × 16 GB', cpu: 4, memoryGB: 16 },
  { label: '4 vCPU × 32 GB', cpu: 4, memoryGB: 32 },
  { label: '4 vCPU × 64 GB', cpu: 4, memoryGB: 64 },
  { label: '8 vCPU × 64 GB', cpu: 8, memoryGB: 64 },
  { label: '8 vCPU × 128 GB', cpu: 8, memoryGB: 128 },
  { label: 'Custom...', cpu: null, memoryGB: null },
];

export const DEFAULT_PRESET_INDEX = 1; // 4×32 (index shifted after adding 4×16)

export const formatCellSize = (cpu, memoryGB) => `${cpu}×${memoryGB}`;

/**
 * Find matching preset index for given CPU/memory, or return Custom index
 * @param {number} cpu - vCPU count
 * @param {number} memoryGB - Memory in GB
 * @returns {{ presetIndex: number, isCustom: boolean }}
 */
export const findMatchingPreset = (cpu, memoryGB) => {
  if (!cpu || !memoryGB) {
    return { presetIndex: VM_SIZE_PRESETS.length - 1, isCustom: true };
  }

  const exactMatch = VM_SIZE_PRESETS.findIndex(
    (p) => p.cpu === cpu && p.memoryGB === memoryGB
  );

  // Found match and it's not the Custom preset (last one)
  if (exactMatch !== -1 && exactMatch !== VM_SIZE_PRESETS.length - 1) {
    return { presetIndex: exactMatch, isCustom: false };
  }

  // Return Custom (last preset)
  return { presetIndex: VM_SIZE_PRESETS.length - 1, isCustom: true };
};
