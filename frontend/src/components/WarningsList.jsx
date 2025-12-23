// frontend/src/components/WarningsList.jsx
// ABOUTME: Displays scenario warnings with severity colors
// ABOUTME: Critical warnings in red, warnings in yellow

import { AlertTriangle, AlertCircle, Info } from 'lucide-react';

const severityConfig = {
  critical: {
    bg: 'bg-red-950/50',
    border: 'border-red-500/30',
    text: 'text-red-300',
    icon: AlertCircle,
  },
  warning: {
    bg: 'bg-amber-950/50',
    border: 'border-amber-500/30',
    text: 'text-amber-300',
    icon: AlertTriangle,
  },
  info: {
    bg: 'bg-blue-950/50',
    border: 'border-blue-500/30',
    text: 'text-blue-300',
    icon: Info,
  },
};

const WarningsList = ({ warnings }) => {
  if (!warnings || warnings.length === 0) {
    return (
      <div className="bg-emerald-950/50 border border-emerald-500/30 rounded-lg p-4 text-emerald-300">
        ✓ No warnings - proposed configuration looks good
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {warnings.map((warning, index) => {
        const config = severityConfig[warning.severity] || severityConfig.info;
        const Icon = config.icon;
        return (
          <div
            key={index}
            className={`${config.bg} ${config.border} border rounded-lg p-3 flex items-start gap-3`}
          >
            <Icon className={`${config.text} flex-shrink-0 mt-0.5`} size={18} />
            <span className={config.text}>{warning.message}</span>
          </div>
        );
      })}
    </div>
  );
};

export default WarningsList;
