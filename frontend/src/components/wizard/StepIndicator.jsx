// ABOUTME: Clickable step progress indicator for wizard navigation
// ABOUTME: Shows completed, current, and locked states for each step

import { Check } from 'lucide-react';

const StepIndicator = ({ steps, currentStep, completedSteps, onStepClick }) => {
  const isCompleted = (index) => completedSteps.includes(index);
  const isCurrent = (index) => index === currentStep;
  const isClickable = (index) => isCompleted(index) || index <= currentStep;

  return (
    <nav aria-label="Progress" className="mb-6">
      <ol className="flex items-center justify-between">
        {steps.map((step, index) => {
          const completed = isCompleted(index);
          const current = isCurrent(index);
          const clickable = isClickable(index);

          return (
            <li key={step.id} className="flex-1 flex items-center">
              <button
                type="button"
                onClick={() => clickable && onStepClick(index)}
                disabled={!clickable}
                aria-current={current ? 'step' : undefined}
                data-completed={completed ? 'true' : undefined}
                className={`
                  flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all
                  ${current
                    ? 'text-cyan-400 bg-cyan-500/10 ring-2 ring-cyan-500/50'
                    : completed
                      ? 'text-cyan-400 hover:bg-slate-700/50 cursor-pointer'
                      : clickable
                        ? 'text-gray-400 hover:bg-slate-700/50 cursor-pointer'
                        : 'text-gray-600 cursor-not-allowed'
                  }
                `}
              >
                <span
                  className={`
                    flex items-center justify-center w-6 h-6 rounded-full border-2 text-xs
                    ${current
                      ? 'border-cyan-500 bg-cyan-500/20'
                      : completed
                        ? 'border-cyan-500 bg-cyan-500'
                        : 'border-gray-600'
                    }
                  `}
                >
                  {completed ? (
                    <Check size={14} className="text-white" />
                  ) : (
                    index + 1
                  )}
                </span>
                <span>{step.label}</span>
                {!step.required && (
                  <span className="text-xs text-gray-500">(optional)</span>
                )}
              </button>

              {index < steps.length - 1 && (
                <div
                  className={`flex-1 h-0.5 mx-2 ${
                    isCompleted(index) ? 'bg-cyan-500' : 'bg-gray-700'
                  }`}
                />
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
};

export default StepIndicator;
