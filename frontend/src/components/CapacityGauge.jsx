// ABOUTME: Circular gauge component for capacity/utilization metrics
// ABOUTME: Shows percentage fill with color-coded status

import React from 'react';

const CapacityGauge = ({
  value,
  max = 100,
  label,
  size = 120,
  thresholds = { warning: 75, critical: 85 },
  inverse = false, // If true, lower is better (like utilization %)
  suffix = '%'
}) => {
  // For utilization metrics, value can exceed 100% (over capacity)
  const isOverCapacity = value > 100 && suffix === '%';
  const displayPercentage = Math.min((value / max) * 100, 100);
  const radius = (size - 16) / 2;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = circumference - (displayPercentage / 100) * circumference;

  // Determine status color
  let status = 'good';
  let color = '#06b6d4'; // cyan-500
  let bgGlow = 'rgba(6, 182, 212, 0.15)';

  // For inverse metrics (utilization), higher value = worse
  // Over capacity is always critical
  if (isOverCapacity) {
    status = 'critical';
    color = '#ef4444'; // red-500
    bgGlow = 'rgba(239, 68, 68, 0.25)';
  } else if (inverse) {
    // For utilization-style metrics: higher = worse
    if (value >= thresholds.critical) {
      status = 'critical';
      color = '#ef4444';
      bgGlow = 'rgba(239, 68, 68, 0.15)';
    } else if (value >= thresholds.warning) {
      status = 'warning';
      color = '#f59e0b';
      bgGlow = 'rgba(245, 158, 11, 0.15)';
    }
  } else {
    // For non-inverse metrics: check thresholds normally
    const effectiveValue = displayPercentage;
    if (effectiveValue >= thresholds.critical) {
      status = 'critical';
      color = '#ef4444';
      bgGlow = 'rgba(239, 68, 68, 0.15)';
    } else if (effectiveValue >= thresholds.warning) {
      status = 'warning';
      color = '#f59e0b';
      bgGlow = 'rgba(245, 158, 11, 0.15)';
    }
  }

  return (
    <div className="flex flex-col items-center gap-2">
      <div
        className="relative"
        style={{ width: size, height: size }}
      >
        {/* Background glow */}
        <div
          className="absolute inset-0 rounded-full blur-xl opacity-50"
          style={{ backgroundColor: bgGlow }}
        />

        {/* SVG Gauge */}
        <svg
          width={size}
          height={size}
          className="transform -rotate-90"
        >
          {/* Background circle */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke="rgba(255,255,255,0.1)"
            strokeWidth="8"
          />

          {/* Progress circle */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke={color}
            strokeWidth="8"
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={isOverCapacity ? 0 : strokeDashoffset}
            style={{
              transition: 'stroke-dashoffset 0.8s ease-out, stroke 0.3s ease',
            }}
          />

          {/* Tick marks */}
          {[0, 25, 50, 75, 100].map((tick) => {
            const angle = (tick / 100) * 360 - 90;
            const radians = (angle * Math.PI) / 180;
            const innerR = radius - 12;
            const outerR = radius - 6;
            return (
              <line
                key={tick}
                x1={size / 2 + innerR * Math.cos(radians)}
                y1={size / 2 + innerR * Math.sin(radians)}
                x2={size / 2 + outerR * Math.cos(radians)}
                y2={size / 2 + outerR * Math.sin(radians)}
                stroke="rgba(255,255,255,0.3)"
                strokeWidth="2"
              />
            );
          })}
        </svg>

        {/* Center value */}
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          {isOverCapacity && (
            <span className="text-[10px] uppercase tracking-wider font-bold text-red-400 mb-0.5">
              OVER
            </span>
          )}
          <span
            className={`font-mono font-bold ${isOverCapacity ? 'text-xl' : 'text-2xl'}`}
            style={{ color }}
          >
            {typeof value === 'number' ? value.toFixed(1) : value}{suffix}
          </span>
        </div>
      </div>

      {/* Label */}
      {label && (
        <span className={`text-xs uppercase tracking-wider font-medium ${isOverCapacity ? 'text-red-400' : 'text-gray-400'}`}>
          {label}
        </span>
      )}
    </div>
  );
};

export default CapacityGauge;
