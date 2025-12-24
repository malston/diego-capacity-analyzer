// ABOUTME: Toast notification component for success/error feedback
// ABOUTME: Auto-dismisses after timeout, accessible with ARIA live regions

import { CheckCircle, XCircle, X } from 'lucide-react';

const Toast = ({ message, variant = 'success', onDismiss }) => {
  const isError = variant === 'error';

  const styles = isError
    ? 'bg-red-500/20 border-red-500/30 text-red-300'
    : 'bg-emerald-500/20 border-emerald-500/30 text-emerald-300';

  const Icon = isError ? XCircle : CheckCircle;

  return (
    <div
      role={isError ? 'alert' : 'status'}
      aria-live={isError ? 'assertive' : 'polite'}
      className={`flex items-center gap-3 px-4 py-3 rounded-lg border shadow-lg ${styles}`}
    >
      <Icon size={18} aria-hidden="true" />
      <span className="flex-1 text-sm font-medium">{message}</span>
      <button
        onClick={onDismiss}
        className="text-current opacity-60 hover:opacity-100 transition-opacity"
        aria-label="Dismiss notification"
      >
        <X size={16} aria-hidden="true" />
      </button>
    </div>
  );
};

export default Toast;
