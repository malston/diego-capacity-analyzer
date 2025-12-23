// ABOUTME: Lightweight tooltip component using CSS-only approach
// ABOUTME: Displays help text on hover with configurable positioning

import { HelpCircle } from 'lucide-react';

const Tooltip = ({ text, children, position = 'top', showIcon = false }) => {
  const positionClasses = {
    top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
    bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
    left: 'right-full top-1/2 -translate-y-1/2 mr-2',
    right: 'left-full top-1/2 -translate-y-1/2 ml-2',
  };

  const arrowClasses = {
    top: 'top-full left-1/2 -translate-x-1/2 border-t-slate-700 border-x-transparent border-b-transparent',
    bottom: 'bottom-full left-1/2 -translate-x-1/2 border-b-slate-700 border-x-transparent border-t-transparent',
    left: 'left-full top-1/2 -translate-y-1/2 border-l-slate-700 border-y-transparent border-r-transparent',
    right: 'right-full top-1/2 -translate-y-1/2 border-r-slate-700 border-y-transparent border-l-transparent',
  };

  return (
    <span className="relative inline-flex items-center group">
      {children}
      {showIcon && (
        <HelpCircle size={12} className="ml-1 text-slate-500 group-hover:text-slate-400 transition-colors" />
      )}
      <span
        className={`
          absolute z-50 ${positionClasses[position]}
          px-3 py-2 text-xs text-slate-200 bg-slate-700 rounded-lg shadow-lg
          opacity-0 invisible group-hover:opacity-100 group-hover:visible
          transition-all duration-200 ease-out
          min-w-[200px] max-w-[280px] whitespace-normal text-center
          pointer-events-none
        `}
      >
        {text}
        <span
          className={`absolute border-4 ${arrowClasses[position]}`}
        />
      </span>
    </span>
  );
};

export default Tooltip;
