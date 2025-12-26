// ABOUTME: Multi-resource bottleneck display component
// ABOUTME: Shows resource exhaustion ordering with highlighted constraint

import { AlertTriangle, HardDrive, Cpu, Database } from 'lucide-react';

// Get icon for resource type
const getResourceIcon = (type) => {
  switch (type) {
    case 'memory':
      return HardDrive;
    case 'cpu':
      return Cpu;
    case 'disk':
      return Database;
    default:
      return HardDrive;
  }
};

// Get utilization status color
const getUtilizationColor = (utilization) => {
  if (utilization >= 85) {
    return {
      text: 'text-red-400',
      bg: 'bg-red-500',
      bgLight: 'bg-red-500/20',
    };
  } else if (utilization >= 70) {
    return {
      text: 'text-amber-400',
      bg: 'bg-amber-500',
      bgLight: 'bg-amber-500/20',
    };
  }
  return {
    text: 'text-cyan-400',
    bg: 'bg-cyan-500',
    bgLight: 'bg-cyan-500/20',
  };
};

const BottleneckCard = ({ resources = [] }) => {
  // Sort resources by utilization (highest first)
  const sortedResources = [...resources].sort((a, b) => b.utilization - a.utilization);
  const constrainingResource = sortedResources[0];

  if (resources.length === 0) {
    return (
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
        <h3 className="text-lg font-semibold mb-4 text-gray-200 flex items-center gap-2">
          <AlertTriangle size={18} className="text-amber-400" />
          Resource Exhaustion Order
        </h3>
        <p className="text-gray-400 text-sm">No resources to analyze</p>
      </div>
    );
  }

  return (
    <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
      <h3 className="text-lg font-semibold mb-4 text-gray-200 flex items-center gap-2">
        <AlertTriangle size={18} className="text-amber-400" />
        Resource Exhaustion Order
      </h3>

      <ul className="space-y-3">
        {sortedResources.map((resource, index) => {
          const isConstraint = index === 0;
          const Icon = getResourceIcon(resource.type);
          const colors = getUtilizationColor(resource.utilization);

          return (
            <li
              key={resource.name}
              role="listitem"
              data-constraint={isConstraint}
              className={`rounded-lg p-4 border transition-all ${
                isConstraint
                  ? 'bg-amber-500/10 border-amber-500/50'
                  : 'bg-slate-700/30 border-slate-600/30'
              }`}
            >
              <div className="flex items-center gap-3">
                {/* Ranking */}
                <div
                  className={`w-8 h-8 rounded-full flex items-center justify-center font-bold text-sm ${
                    isConstraint ? 'bg-amber-500/30 text-amber-400' : 'bg-slate-600/50 text-gray-400'
                  }`}
                >
                  {index + 1}
                </div>

                {/* Resource Icon and Name */}
                <div className="flex items-center gap-2 flex-1">
                  <Icon size={16} className={colors.text} />
                  <span className={`font-medium ${isConstraint ? 'text-amber-200' : 'text-gray-300'}`}>
                    {resource.name}
                  </span>
                  {isConstraint && (
                    <span className="text-xs text-amber-400 px-2 py-0.5 bg-amber-500/20 rounded">
                      ← Closest to limit
                    </span>
                  )}
                </div>

                {/* Utilization Value */}
                <span className={`font-mono font-bold ${colors.text}`}>
                  {resource.utilization}%
                </span>
              </div>

              {/* Progress Bar */}
              <div className="mt-2 ml-11">
                <div
                  role="progressbar"
                  aria-valuenow={resource.utilization}
                  aria-valuemin={0}
                  aria-valuemax={100}
                  className="h-2 bg-slate-700 rounded-full overflow-hidden"
                >
                  <div
                    className={`h-full ${colors.bg} transition-all duration-500`}
                    style={{ width: `${resource.utilization}%` }}
                  />
                </div>
              </div>
            </li>
          );
        })}
      </ul>

      {/* Recommendation */}
      {constrainingResource && (
        <div className="mt-4 p-3 bg-slate-700/30 rounded-lg border border-slate-600/30">
          <p className="text-sm text-gray-300">
            <span className="font-medium text-amber-400">{constrainingResource.name}</span>
            {' '}is your constraint. Address this resource before worrying about others.
          </p>
        </div>
      )}
    </div>
  );
};

export default BottleneckCard;
