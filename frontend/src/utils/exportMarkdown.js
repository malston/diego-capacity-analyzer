// ABOUTME: Markdown export utility for scenario analysis
// ABOUTME: Generates formatted capacity analysis documents

/**
 * Format a number with commas for readability
 */
function formatNumber(num) {
  if (num === undefined || num === null) return '-';
  return num.toLocaleString();
}

/**
 * Format percentage with one decimal place
 */
function formatPct(num) {
  if (num === undefined || num === null) return '-';
  return `${num.toFixed(1)}%`;
}

/**
 * Get change indicator arrow
 */
function getChangeIndicator(current, proposed, higherIsBetter = true) {
  if (current === proposed) return '→';
  const improved = higherIsBetter ? proposed > current : proposed < current;
  return improved ? '↑' : '↓';
}

/**
 * Format cell size string
 */
function formatCellSize(cpu, memoryGB) {
  return `${cpu} vCPU × ${memoryGB} GB`;
}

/**
 * Generate Markdown report from comparison data
 */
export function generateMarkdownReport(comparison, infrastructureData) {
  const { current, proposed, warnings, delta } = comparison;
  const timestamp = new Date().toISOString().split('T')[0];
  const envName = infrastructureData?.name || 'Environment';

  let md = `# Diego Cell Capacity Analysis

**Environment:** ${envName}
**Generated:** ${timestamp}

---

## Executive Summary

This analysis compares the current Diego cell configuration with a proposed change.

| Aspect | Current | Proposed | Change |
|--------|---------|----------|--------|
| Cell Size | ${formatCellSize(current.cell_cpu || 4, current.cell_memory_gb)} | ${formatCellSize(proposed.cell_cpu || proposed.proposed_cell_cpu || 4, proposed.cell_memory_gb || proposed.proposed_cell_memory_gb)} | - |
| Cell Count | ${formatNumber(current.cell_count)} | ${formatNumber(proposed.cell_count)} | ${delta.capacity_change_gb >= 0 ? '+' : ''}${formatNumber(proposed.cell_count - current.cell_count)} |
| Total Capacity | ${formatNumber(current.app_capacity_gb)} GB | ${formatNumber(proposed.app_capacity_gb)} GB | ${delta.capacity_change_gb >= 0 ? '+' : ''}${formatNumber(delta.capacity_change_gb)} GB |

---

## Detailed Metrics

### Capacity & Utilization

| Metric | Current | Proposed | Change |
|--------|---------|----------|--------|
| App Capacity | ${formatNumber(current.app_capacity_gb)} GB | ${formatNumber(proposed.app_capacity_gb)} GB | ${getChangeIndicator(current.app_capacity_gb, proposed.app_capacity_gb, true)} |
| Cell Utilization | ${formatPct(current.utilization_pct)} | ${formatPct(proposed.utilization_pct)} | ${getChangeIndicator(current.utilization_pct, proposed.utilization_pct, false)} |
| Free Chunks (4GB) | ${formatNumber(current.free_chunks)} | ${formatNumber(proposed.free_chunks)} | ${getChangeIndicator(current.free_chunks, proposed.free_chunks, true)} |

### N-1 Redundancy

| Metric | Current | Proposed | Change |
|--------|---------|----------|--------|
| N-1 Utilization | ${formatPct(current.n1_utilization_pct)} | ${formatPct(proposed.n1_utilization_pct)} | ${getChangeIndicator(current.n1_utilization_pct, proposed.n1_utilization_pct, false)} |

### Fault Tolerance

| Metric | Current | Proposed | Change |
|--------|---------|----------|--------|
| Fault Impact | ${formatNumber(current.fault_impact)} apps/cell | ${formatNumber(proposed.fault_impact)} apps/cell | ${getChangeIndicator(current.fault_impact, proposed.fault_impact, false)} |
| Instances/Cell | ${current.instances_per_cell?.toFixed(1) || '-'} | ${proposed.instances_per_cell?.toFixed(1) || '-'} | ${getChangeIndicator(current.instances_per_cell, proposed.instances_per_cell, false)} |

---

## Recommendations

`;

  // Add warnings as recommendations
  if (warnings && warnings.length > 0) {
    const criticalWarnings = warnings.filter(w => w.severity === 'critical');
    const warningWarnings = warnings.filter(w => w.severity === 'warning');
    const infoWarnings = warnings.filter(w => w.severity === 'info');

    if (criticalWarnings.length > 0) {
      md += `### ⛔ Critical Issues\n\n`;
      criticalWarnings.forEach(w => {
        md += `- **${w.message}**\n`;
      });
      md += '\n';
    }

    if (warningWarnings.length > 0) {
      md += `### ⚠️ Warnings\n\n`;
      warningWarnings.forEach(w => {
        md += `- ${w.message}\n`;
      });
      md += '\n';
    }

    if (infoWarnings.length > 0) {
      md += `### ℹ️ Notes\n\n`;
      infoWarnings.forEach(w => {
        md += `- ${w.message}\n`;
      });
      md += '\n';
    }
  } else {
    md += `✅ No warnings for this configuration.\n\n`;
  }

  // Add cell resilience assessment based on blast radius
  md += `### Cell Failure Resilience\n\n`;
  const blastRadius = proposed.blast_radius_pct?.toFixed(1) || 'N/A';
  if (delta.resilience_change === 'low') {
    md += `✅ **Low risk** - Single cell failure affects only ${blastRadius}% of capacity.\n`;
  } else if (delta.resilience_change === 'moderate') {
    md += `⚠️ **Moderate risk** - Single cell failure affects ${blastRadius}% of capacity.\n`;
  } else if (delta.resilience_change === 'high') {
    md += `🔴 **High risk** - Single cell failure affects ${blastRadius}% of capacity. Consider adding more cells.\n`;
  } else {
    md += `Cell resilience assessment: ${blastRadius}% capacity at risk per cell failure.\n`;
  }

  md += `
---

## Configuration Details

`;

  // Add cluster breakdown if available
  if (infrastructureData?.clusters && infrastructureData.clusters.length > 0) {
    md += `### Cluster Breakdown\n\n`;
    md += `| Cluster | Hosts | Diego Cells | Cell Size |\n`;
    md += `|---------|-------|-------------|----------|\n`;
    infrastructureData.clusters.forEach(c => {
      const cellSize = formatCellSize(c.diego_cell_cpu || c.diego_cell_vcpu || 4, c.diego_cell_memory_gb);
      md += `| ${c.name} | ${c.host_count} | ${c.diego_cell_count} | ${cellSize} |\n`;
    });
    md += '\n';
  }

  md += `---

*Generated by Diego Capacity Analyzer*
`;

  return md;
}

/**
 * Download markdown as a file
 */
export function downloadMarkdown(markdown, filename = 'capacity-analysis.md') {
  const blob = new Blob([markdown], { type: 'text/markdown' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
