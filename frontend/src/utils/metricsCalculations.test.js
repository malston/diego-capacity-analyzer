// ABOUTME: Tests for metrics calculation utilities
// ABOUTME: Covers cell metrics, app metrics, what-if, and recommendations

import { describe, it, expect } from "vitest";
import {
  calculateCellMetrics,
  calculateAppMetrics,
  calculateWhatIfMetrics,
  calculateRecommendations,
  filterBySegment,
} from "./metricsCalculations";

describe("calculateCellMetrics", () => {
  it("returns zero values for empty array", () => {
    const result = calculateCellMetrics([]);
    expect(result.totalCells).toBe(0);
    expect(result.totalMemory).toBe(0);
    expect(result.utilizationPercent).toBe(0);
  });

  it("returns zero values for null input", () => {
    const result = calculateCellMetrics(null);
    expect(result.totalCells).toBe(0);
  });

  it("calculates totals correctly", () => {
    const cells = [
      { memory_mb: 16384, allocated_mb: 12000, used_mb: 8000, cpu_percent: 40 },
      {
        memory_mb: 16384,
        allocated_mb: 14000,
        used_mb: 10000,
        cpu_percent: 60,
      },
    ];

    const result = calculateCellMetrics(cells);

    expect(result.totalCells).toBe(2);
    expect(result.totalMemory).toBe(32768);
    expect(result.totalAllocated).toBe(26000);
    expect(result.totalUsed).toBe(18000);
    expect(result.avgCpu).toBe(50);
  });

  it("calculates utilization percentage correctly", () => {
    const cells = [
      { memory_mb: 10000, allocated_mb: 8000, used_mb: 5000, cpu_percent: 50 },
    ];

    const result = calculateCellMetrics(cells);

    expect(result.utilizationPercent).toBe(50); // 5000/10000 * 100
    expect(result.allocationPercent).toBe(80); // 8000/10000 * 100
  });
});

describe("calculateAppMetrics", () => {
  it("returns zero values for empty array", () => {
    const result = calculateAppMetrics([]);
    expect(result.totalApps).toBe(0);
    expect(result.unusedMemory).toBe(0);
  });

  it("calculates unused memory correctly", () => {
    const apps = [
      { requested_mb: 1024, actual_mb: 800, instances: 2 },
      { requested_mb: 512, actual_mb: 400, instances: 3 },
    ];

    const result = calculateAppMetrics(apps);

    // Total requested: (1024*2) + (512*3) = 2048 + 1536 = 3584
    // Total used: (800*2) + (400*3) = 1600 + 1200 = 2800
    // Unused: 3584 - 2800 = 784
    expect(result.totalRequested).toBe(3584);
    expect(result.totalUsed).toBe(2800);
    expect(result.unusedMemory).toBe(784);
    expect(result.totalInstances).toBe(5);
  });

  it("calculates unused percent correctly", () => {
    const apps = [{ requested_mb: 1000, actual_mb: 750, instances: 1 }];

    const result = calculateAppMetrics(apps);

    expect(result.unusedPercent).toBe(25); // 250/1000 * 100
  });
});

describe("calculateWhatIfMetrics", () => {
  it("calculates new capacity with overcommit ratio", () => {
    const result = calculateWhatIfMetrics(32768, 1.5, 50, 512);

    expect(result.newCapacity).toBe(49152); // 32768 * 1.5
  });

  it("calculates potential instances using provided average instance size", () => {
    // With 1024MB average instance size
    const result = calculateWhatIfMetrics(32768, 1.0, 0, 1024);

    expect(result.potentialInstances).toBe(32); // 32768 / 1024
    expect(result.additionalInstances).toBe(32);
    expect(result.avgInstanceSize).toBe(1024);
  });

  it("falls back to 512MB when avgInstanceSize is zero", () => {
    const result = calculateWhatIfMetrics(32768, 1.0, 0, 0);

    expect(result.potentialInstances).toBe(64); // 32768 / 512 (fallback)
    expect(result.avgInstanceSize).toBe(512);
  });

  it("calculates additional instances correctly with custom average", () => {
    // 32768 * 2.0 = 65536 new capacity
    // 65536 / 256 = 256 potential instances
    // 256 - 100 = 156 additional
    const result = calculateWhatIfMetrics(32768, 2.0, 100, 256);

    expect(result.additionalInstances).toBe(156);
  });

  it("handles realistic workload with mixed instance sizes", () => {
    // Simulate a workload: total requested 8192MB across 10 instances = 819.2 avg
    const avgSize = 819.2;
    const result = calculateWhatIfMetrics(32768, 1.5, 10, avgSize);

    // New capacity: 49152
    // Potential: floor(49152 / 819.2) = 60
    // Additional: 60 - 10 = 50
    expect(result.newCapacity).toBe(49152);
    expect(result.potentialInstances).toBe(60);
    expect(result.additionalInstances).toBe(50);
  });
});

describe("calculateRecommendations", () => {
  it("returns empty array for no apps", () => {
    expect(calculateRecommendations([])).toEqual([]);
    expect(calculateRecommendations(null)).toEqual([]);
  });

  it("filters apps below threshold", () => {
    const apps = [
      {
        name: "low-overhead",
        requested_mb: 1000,
        actual_mb: 900,
        instances: 1,
      }, // 10% overhead
      {
        name: "high-overhead",
        requested_mb: 1000,
        actual_mb: 500,
        instances: 1,
      }, // 50% overhead
    ];

    const result = calculateRecommendations(apps, 15);

    expect(result).toHaveLength(1);
    expect(result[0].name).toBe("high-overhead");
  });

  it("calculates recommended size with 20% buffer", () => {
    const apps = [
      { name: "test-app", requested_mb: 1000, actual_mb: 500, instances: 1 },
    ];

    const result = calculateRecommendations(apps, 0);

    // Recommended: 500 * 1.2 = 600
    expect(result[0].recommended).toBe(600);
    expect(result[0].savings).toBe(400); // 1000 - 600
  });

  it("sorts by total savings descending", () => {
    const apps = [
      {
        name: "small-savings",
        requested_mb: 1000,
        actual_mb: 500,
        instances: 1,
      },
      {
        name: "big-savings",
        requested_mb: 1000,
        actual_mb: 500,
        instances: 10,
      },
    ];

    const result = calculateRecommendations(apps, 0);

    expect(result[0].name).toBe("big-savings");
    expect(result[0].totalSavings).toBe(4000); // 400 * 10
  });
});

describe("filterBySegment", () => {
  const items = [
    { name: "a", isolation_segment: "prod" },
    { name: "b", isolation_segment: "dev" },
    { name: "c", isolation_segment: "prod" },
  ];

  it('returns all items for "all" segment', () => {
    expect(filterBySegment(items, "all")).toHaveLength(3);
  });

  it("returns all items for null/undefined segment", () => {
    expect(filterBySegment(items, null)).toHaveLength(3);
    expect(filterBySegment(items, undefined)).toHaveLength(3);
  });

  it("filters by segment correctly", () => {
    const result = filterBySegment(items, "prod");
    expect(result).toHaveLength(2);
    expect(result.every((i) => i.isolation_segment === "prod")).toBe(true);
  });

  it("supports custom segment field", () => {
    const customItems = [
      { name: "a", segment: "x" },
      { name: "b", segment: "y" },
    ];
    const result = filterBySegment(customItems, "x", "segment");
    expect(result).toHaveLength(1);
  });
});
