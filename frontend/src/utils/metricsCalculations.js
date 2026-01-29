// ABOUTME: Pure functions for calculating dashboard metrics
// ABOUTME: Extracted for testability and reuse

/**
 * Calculate cell metrics from filtered cells data
 */
export function calculateCellMetrics(cells) {
  if (!cells || cells.length === 0) {
    return {
      totalCells: 0,
      totalMemory: 0,
      totalAllocated: 0,
      totalUsed: 0,
      avgCpu: 0,
      utilizationPercent: 0,
      allocationPercent: 0,
    };
  }

  const totalMemory = cells.reduce((sum, c) => sum + c.memory_mb, 0);
  const totalAllocated = cells.reduce((sum, c) => sum + c.allocated_mb, 0);
  const totalUsed = cells.reduce((sum, c) => sum + c.used_mb, 0);
  const avgCpu =
    cells.reduce((sum, c) => sum + c.cpu_percent, 0) / cells.length;

  return {
    totalCells: cells.length,
    totalMemory,
    totalAllocated,
    totalUsed,
    avgCpu,
    utilizationPercent: totalMemory > 0 ? (totalUsed / totalMemory) * 100 : 0,
    allocationPercent:
      totalMemory > 0 ? (totalAllocated / totalMemory) * 100 : 0,
  };
}

/**
 * Calculate app memory metrics from filtered apps data
 */
export function calculateAppMetrics(apps) {
  if (!apps || apps.length === 0) {
    return {
      totalApps: 0,
      totalInstances: 0,
      totalRequested: 0,
      totalUsed: 0,
      unusedMemory: 0,
      unusedPercent: 0,
    };
  }

  const totalRequested = apps.reduce(
    (sum, a) => sum + a.requested_mb * a.instances,
    0,
  );
  const totalUsed = apps.reduce((sum, a) => sum + a.actual_mb * a.instances, 0);
  const totalInstances = apps.reduce((sum, a) => sum + a.instances, 0);
  const unusedMemory = totalRequested - totalUsed;

  return {
    totalApps: apps.length,
    totalInstances,
    totalRequested,
    totalUsed,
    unusedMemory,
    unusedPercent:
      totalRequested > 0 ? (unusedMemory / totalRequested) * 100 : 0,
  };
}

/**
 * Calculate what-if scenario metrics
 * @param {number} totalMemory - Total cell memory in MB
 * @param {number} overcommitRatio - Memory overcommit multiplier (e.g., 1.5 = 150%)
 * @param {number} currentInstances - Current number of app instances
 * @param {number} avgInstanceSize - Average memory per instance in MB (calculated from actual apps)
 */
export function calculateWhatIfMetrics(
  totalMemory,
  overcommitRatio,
  currentInstances,
  avgInstanceSize,
) {
  const newCapacity = totalMemory * overcommitRatio;
  // Use actual average or fall back to a reasonable default if no instances exist
  const effectiveAvgSize = avgInstanceSize > 0 ? avgInstanceSize : 512;
  const potentialInstances = Math.floor(newCapacity / effectiveAvgSize);

  return {
    newCapacity,
    potentialInstances,
    additionalInstances: potentialInstances - currentInstances,
    avgInstanceSize: effectiveAvgSize,
  };
}

/**
 * Calculate right-sizing recommendations for apps
 */
export function calculateRecommendations(apps, thresholdPercent = 15) {
  if (!apps || apps.length === 0) {
    return [];
  }

  return apps
    .map((app) => {
      const overhead = app.requested_mb - app.actual_mb;
      const overheadPercent =
        app.requested_mb > 0 ? (overhead / app.requested_mb) * 100 : 0;
      const recommended = Math.ceil(app.actual_mb * 1.2); // 20% buffer
      const savings = app.requested_mb - recommended;

      return {
        ...app,
        overhead,
        overheadPercent,
        recommended,
        savings: Math.max(0, savings),
        totalSavings: Math.max(0, savings * app.instances),
      };
    })
    .filter((app) => app.overheadPercent > thresholdPercent)
    .sort((a, b) => b.totalSavings - a.totalSavings);
}

/**
 * Filter cells by isolation segment
 */
export function filterBySegment(
  items,
  segment,
  segmentField = "isolation_segment",
) {
  if (segment === "all" || !segment) {
    return items;
  }
  return items.filter((item) => item[segmentField] === segment);
}
