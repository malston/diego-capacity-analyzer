// ABOUTME: Clickable step progress indicator for wizard navigation
// ABOUTME: Shows completed and current states - all steps are navigable

import { Check } from "lucide-react";

const StepIndicator = ({ steps, currentStep, completedSteps, onStepClick }) => {
  const isCompleted = (step) => completedSteps.includes(step.id);
  const isCurrent = (index) => index === currentStep;

  return (
    <nav aria-label="Progress" className="mb-6">
      <ol className="flex items-center justify-between">
        {steps.map((step, index) => {
          const completed = isCompleted(step);
          const current = isCurrent(index);

          return (
            <li key={step.id} className="flex-1 flex items-center">
              <button
                type="button"
                onClick={() => onStepClick(index)}
                aria-current={current ? "step" : undefined}
                data-completed={completed ? "true" : undefined}
                className={`
                  flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all
                  hover:bg-slate-700/50 cursor-pointer
                  ${
                    current
                      ? "text-cyan-400 bg-cyan-500/10 ring-2 ring-cyan-500/50"
                      : completed
                        ? "text-cyan-400"
                        : "text-gray-400"
                  }
                `}
              >
                <span
                  className={`
                    flex items-center justify-center w-6 h-6 rounded-full border-2 text-xs
                    ${
                      current
                        ? "border-cyan-500 bg-cyan-500/20"
                        : completed
                          ? "border-cyan-500 bg-cyan-500"
                          : "border-gray-600"
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
                    isCompleted(step) ? "bg-cyan-500" : "bg-gray-700"
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
