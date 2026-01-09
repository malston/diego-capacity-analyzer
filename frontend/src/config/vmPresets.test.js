// ABOUTME: Tests for VM preset utilities
// ABOUTME: Covers preset matching logic for auto-population from infrastructure data

import { describe, it, expect } from 'vitest';
import { VM_SIZE_PRESETS, findMatchingPreset } from './vmPresets';

describe('VM_SIZE_PRESETS', () => {
  it('includes 4×16 GB preset for Small Footprint TAS', () => {
    const preset = VM_SIZE_PRESETS.find(p => p.cpu === 4 && p.memoryGB === 16);
    expect(preset).toBeDefined();
    expect(preset.label).toBe('4 vCPU × 16 GB');
  });

  it('has Custom as last preset with null values', () => {
    const lastPreset = VM_SIZE_PRESETS[VM_SIZE_PRESETS.length - 1];
    expect(lastPreset.label).toBe('Custom...');
    expect(lastPreset.cpu).toBeNull();
    expect(lastPreset.memoryGB).toBeNull();
  });
});

describe('findMatchingPreset', () => {
  it('returns exact match for 4×16', () => {
    const result = findMatchingPreset(4, 16);
    expect(result.isCustom).toBe(false);
    expect(VM_SIZE_PRESETS[result.presetIndex].cpu).toBe(4);
    expect(VM_SIZE_PRESETS[result.presetIndex].memoryGB).toBe(16);
  });

  it('returns exact match for 4×32', () => {
    const result = findMatchingPreset(4, 32);
    expect(result.isCustom).toBe(false);
    expect(VM_SIZE_PRESETS[result.presetIndex].memoryGB).toBe(32);
  });

  it('returns exact match for 4×64', () => {
    const result = findMatchingPreset(4, 64);
    expect(result.isCustom).toBe(false);
    expect(VM_SIZE_PRESETS[result.presetIndex].memoryGB).toBe(64);
  });

  it('returns exact match for 8×64', () => {
    const result = findMatchingPreset(8, 64);
    expect(result.isCustom).toBe(false);
    expect(VM_SIZE_PRESETS[result.presetIndex].cpu).toBe(8);
    expect(VM_SIZE_PRESETS[result.presetIndex].memoryGB).toBe(64);
  });

  it('returns exact match for 8×128', () => {
    const result = findMatchingPreset(8, 128);
    expect(result.isCustom).toBe(false);
    expect(VM_SIZE_PRESETS[result.presetIndex].memoryGB).toBe(128);
  });

  it('returns Custom for non-matching CPU/memory combination', () => {
    const result = findMatchingPreset(6, 48);
    expect(result.isCustom).toBe(true);
    expect(result.presetIndex).toBe(VM_SIZE_PRESETS.length - 1);
  });

  it('returns Custom for null values', () => {
    const result = findMatchingPreset(null, null);
    expect(result.isCustom).toBe(true);
  });

  it('returns Custom for zero values', () => {
    const result = findMatchingPreset(0, 0);
    expect(result.isCustom).toBe(true);
  });

  it('returns Custom when only CPU matches', () => {
    const result = findMatchingPreset(4, 48); // 4 CPU but non-standard memory
    expect(result.isCustom).toBe(true);
  });

  it('returns Custom when only memory matches', () => {
    const result = findMatchingPreset(6, 64); // non-standard CPU but 64GB memory
    expect(result.isCustom).toBe(true);
  });
});
