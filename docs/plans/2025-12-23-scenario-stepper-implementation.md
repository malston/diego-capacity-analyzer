# Scenario Stepper Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor ScenarioAnalyzer from a dense form into a linear wizard with clickable step navigation.

**Architecture:** Extract form inputs into three step components (CellConfig, ResourceTypes, Advanced), coordinated by a ScenarioWizard container with a clickable StepIndicator. Info displays remain always visible in ScenarioAnalyzer.

**Tech Stack:** React 18, Vitest, React Testing Library, Tailwind CSS, Lucide icons

---

## Task 1: StepIndicator Component

**Files:**
- Create: `frontend/src/components/wizard/StepIndicator.jsx`
- Create: `frontend/src/components/wizard/StepIndicator.test.jsx`

### Step 1.1: Write the failing test

Create the test file with initial tests:

```jsx
// frontend/src/components/wizard/StepIndicator.test.jsx
// ABOUTME: Tests for wizard step indicator component
// ABOUTME: Covers step rendering, click navigation, and visual states

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import StepIndicator from './StepIndicator';

const STEPS = [
  { id: 'cell-config', label: 'Cell Config', required: true },
  { id: 'resources', label: 'Resources', required: false },
  { id: 'advanced', label: 'Advanced', required: false },
];

describe('StepIndicator', () => {
  const defaultProps = {
    steps: STEPS,
    currentStep: 0,
    completedSteps: [],
    onStepClick: vi.fn(),
  };

  it('renders all step labels', () => {
    render(<StepIndicator {...defaultProps} />);
    expect(screen.getByText('Cell Config')).toBeInTheDocument();
    expect(screen.getByText('Resources')).toBeInTheDocument();
    expect(screen.getByText('Advanced')).toBeInTheDocument();
  });

  it('marks current step as active', () => {
    render(<StepIndicator {...defaultProps} currentStep={1} />);
    const resourcesStep = screen.getByText('Resources').closest('button');
    expect(resourcesStep).toHaveAttribute('aria-current', 'step');
  });

  it('marks completed steps with checkmark', () => {
    render(<StepIndicator {...defaultProps} completedSteps={[0]} />);
    const cellConfigStep = screen.getByText('Cell Config').closest('button');
    expect(cellConfigStep).toHaveAttribute('data-completed', 'true');
  });

  it('calls onStepClick when clicking completed step', async () => {
    const onStepClick = vi.fn();
    render(
      <StepIndicator
        {...defaultProps}
        currentStep={1}
        completedSteps={[0]}
        onStepClick={onStepClick}
      />
    );
    await userEvent.click(screen.getByText('Cell Config'));
    expect(onStepClick).toHaveBeenCalledWith(0);
  });

  it('does not call onStepClick when clicking locked step', async () => {
    const onStepClick = vi.fn();
    render(
      <StepIndicator
        {...defaultProps}
        currentStep={0}
        completedSteps={[]}
        onStepClick={onStepClick}
      />
    );
    await userEvent.click(screen.getByText('Advanced'));
    expect(onStepClick).not.toHaveBeenCalled();
  });
});
```

### Step 1.2: Run test to verify it fails

```bash
cd frontend && npm test -- StepIndicator
```

Expected: FAIL - module not found

### Step 1.3: Write minimal implementation

```jsx
// frontend/src/components/wizard/StepIndicator.jsx
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
```

### Step 1.4: Run test to verify it passes

```bash
cd frontend && npm test -- StepIndicator
```

Expected: PASS (5 tests)

### Step 1.5: Commit

```bash
git add frontend/src/components/wizard/
git commit -m "feat: add StepIndicator component for wizard navigation"
```

---

## Task 2: CellConfigStep Component

**Files:**
- Create: `frontend/src/components/wizard/steps/CellConfigStep.jsx`
- Create: `frontend/src/components/wizard/steps/CellConfigStep.test.jsx`

### Step 2.1: Write the failing test

```jsx
// frontend/src/components/wizard/steps/CellConfigStep.test.jsx
// ABOUTME: Tests for cell configuration step in scenario wizard
// ABOUTME: Covers VM size selection, custom inputs, and cell count

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import CellConfigStep from './CellConfigStep';

describe('CellConfigStep', () => {
  const defaultProps = {
    selectedPreset: 0,
    setSelectedPreset: vi.fn(),
    customCPU: 4,
    setCustomCPU: vi.fn(),
    customMemory: 32,
    setCustomMemory: vi.fn(),
    cellCount: 100,
    setCellCount: vi.fn(),
    equivalentCellSuggestion: null,
    onContinue: vi.fn(),
  };

  it('renders VM size dropdown', () => {
    render(<CellConfigStep {...defaultProps} />);
    expect(screen.getByLabelText(/vm size/i)).toBeInTheDocument();
  });

  it('renders cell count input', () => {
    render(<CellConfigStep {...defaultProps} />);
    expect(screen.getByLabelText(/cell count/i)).toBeInTheDocument();
  });

  it('calls onContinue when Continue button clicked', async () => {
    const onContinue = vi.fn();
    render(<CellConfigStep {...defaultProps} onContinue={onContinue} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(onContinue).toHaveBeenCalled();
  });

  it('disables Continue when cellCount is 0', () => {
    render(<CellConfigStep {...defaultProps} cellCount={0} />);
    expect(screen.getByRole('button', { name: /continue/i })).toBeDisabled();
  });

  it('shows equivalent cells suggestion when provided', () => {
    render(
      <CellConfigStep
        {...defaultProps}
        equivalentCellSuggestion={{
          equivalentCells: 200,
          currentTotalGB: 6400,
        }}
      />
    );
    expect(screen.getByText(/equivalent capacity/i)).toBeInTheDocument();
    expect(screen.getByText(/200 cells/i)).toBeInTheDocument();
  });
});
```

### Step 2.2: Run test to verify it fails

```bash
cd frontend && npm test -- CellConfigStep
```

Expected: FAIL - module not found

### Step 2.3: Write minimal implementation

```jsx
// frontend/src/components/wizard/steps/CellConfigStep.jsx
// ABOUTME: Step 1 of scenario wizard - Diego cell configuration
// ABOUTME: Handles VM size preset selection and cell count input

import { Sparkles, ArrowRight } from 'lucide-react';
import { VM_SIZE_PRESETS } from '../../../config/vmPresets';

const CellConfigStep = ({
  selectedPreset,
  setSelectedPreset,
  customCPU,
  setCustomCPU,
  customMemory,
  setCustomMemory,
  cellCount,
  setCellCount,
  equivalentCellSuggestion,
  onContinue,
}) => {
  const preset = VM_SIZE_PRESETS[selectedPreset];
  const isCustom = preset.cpu === null;
  const canContinue = cellCount > 0;

  return (
    <div className="space-y-6">
      <div>
        <label
          htmlFor="vm-size"
          className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
        >
          VM Size
        </label>
        <select
          id="vm-size"
          value={selectedPreset}
          onChange={(e) => setSelectedPreset(Number(e.target.value))}
          className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
        >
          {VM_SIZE_PRESETS.map((p, i) => (
            <option key={i} value={i}>
              {p.label}
            </option>
          ))}
        </select>
      </div>

      {isCustom && (
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="custom-cpu"
              className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
            >
              vCPU
            </label>
            <input
              id="custom-cpu"
              type="number"
              value={customCPU}
              onChange={(e) => setCustomCPU(Number(e.target.value))}
              min={1}
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
            />
          </div>
          <div>
            <label
              htmlFor="custom-memory"
              className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
            >
              Memory (GB)
            </label>
            <input
              id="custom-memory"
              type="number"
              value={customMemory}
              onChange={(e) => setCustomMemory(Number(e.target.value))}
              min={8}
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
            />
          </div>
        </div>
      )}

      <div>
        <label
          htmlFor="cell-count"
          className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
        >
          Cell Count
        </label>
        <input
          id="cell-count"
          type="number"
          value={cellCount}
          onChange={(e) => setCellCount(Number(e.target.value))}
          min={1}
          className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
        />
        {equivalentCellSuggestion && (
          <button
            type="button"
            onClick={() => setCellCount(equivalentCellSuggestion.equivalentCells)}
            className="mt-2 text-xs text-amber-400 hover:text-amber-300 flex items-center gap-1 transition-colors"
          >
            <Sparkles size={12} />
            For equivalent capacity ({equivalentCellSuggestion.currentTotalGB}GB): use {equivalentCellSuggestion.equivalentCells} cells
          </button>
        )}
      </div>

      <div className="flex justify-end pt-4">
        <button
          type="button"
          onClick={onContinue}
          disabled={!canContinue}
          className="flex items-center gap-2 px-6 py-2.5 bg-cyan-600 text-white rounded-lg hover:bg-cyan-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors font-medium"
        >
          Continue
          <ArrowRight size={16} />
        </button>
      </div>
    </div>
  );
};

export default CellConfigStep;
```

### Step 2.4: Run test to verify it passes

```bash
cd frontend && npm test -- CellConfigStep
```

Expected: PASS (5 tests)

### Step 2.5: Commit

```bash
git add frontend/src/components/wizard/steps/
git commit -m "feat: add CellConfigStep component for VM size and cell count"
```

---

## Task 3: ResourceTypesStep Component

**Files:**
- Create: `frontend/src/components/wizard/steps/ResourceTypesStep.jsx`
- Create: `frontend/src/components/wizard/steps/ResourceTypesStep.test.jsx`

### Step 3.1: Write the failing test

```jsx
// frontend/src/components/wizard/steps/ResourceTypesStep.test.jsx
// ABOUTME: Tests for resource types step in scenario wizard
// ABOUTME: Covers resource toggle buttons and disk input

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ResourceTypesStep from './ResourceTypesStep';

describe('ResourceTypesStep', () => {
  const defaultProps = {
    selectedResources: ['memory'],
    toggleResource: vi.fn(),
    customDisk: 128,
    setCustomDisk: vi.fn(),
    onContinue: vi.fn(),
    onSkip: vi.fn(),
  };

  it('renders resource type buttons', () => {
    render(<ResourceTypesStep {...defaultProps} />);
    expect(screen.getByRole('button', { name: /memory/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /cpu/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /disk/i })).toBeInTheDocument();
  });

  it('shows selected state for active resources', () => {
    render(<ResourceTypesStep {...defaultProps} selectedResources={['memory', 'cpu']} />);
    const memoryBtn = screen.getByRole('button', { name: /memory/i });
    expect(memoryBtn).toHaveAttribute('aria-pressed', 'true');
  });

  it('calls toggleResource when clicking resource button', async () => {
    const toggleResource = vi.fn();
    render(<ResourceTypesStep {...defaultProps} toggleResource={toggleResource} />);
    await userEvent.click(screen.getByRole('button', { name: /cpu/i }));
    expect(toggleResource).toHaveBeenCalledWith('cpu');
  });

  it('shows disk input only when disk is selected', () => {
    const { rerender } = render(<ResourceTypesStep {...defaultProps} selectedResources={['memory']} />);
    expect(screen.queryByLabelText(/disk per cell/i)).not.toBeInTheDocument();

    rerender(<ResourceTypesStep {...defaultProps} selectedResources={['memory', 'disk']} />);
    expect(screen.getByLabelText(/disk per cell/i)).toBeInTheDocument();
  });

  it('calls onSkip when Skip button clicked', async () => {
    const onSkip = vi.fn();
    render(<ResourceTypesStep {...defaultProps} onSkip={onSkip} />);
    await userEvent.click(screen.getByRole('button', { name: /skip/i }));
    expect(onSkip).toHaveBeenCalled();
  });

  it('calls onContinue when Continue button clicked', async () => {
    const onContinue = vi.fn();
    render(<ResourceTypesStep {...defaultProps} onContinue={onContinue} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(onContinue).toHaveBeenCalled();
  });
});
```

### Step 3.2: Run test to verify it fails

```bash
cd frontend && npm test -- ResourceTypesStep
```

Expected: FAIL - module not found

### Step 3.3: Write minimal implementation

```jsx
// frontend/src/components/wizard/steps/ResourceTypesStep.jsx
// ABOUTME: Step 2 of scenario wizard - resource type selection
// ABOUTME: Handles memory/CPU/disk toggle and disk size input

import { ArrowRight } from 'lucide-react';
import { RESOURCE_TYPES } from '../../../config/resourceConfig';

const ResourceTypesStep = ({
  selectedResources,
  toggleResource,
  customDisk,
  setCustomDisk,
  onContinue,
  onSkip,
}) => {
  const showDiskInput = selectedResources.includes('disk');

  return (
    <div className="space-y-6">
      <div>
        <label className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-3">
          Which resources to analyze?
        </label>
        <div className="flex flex-wrap gap-2">
          {RESOURCE_TYPES.map((resource) => {
            const Icon = resource.icon;
            const isSelected = selectedResources.includes(resource.id);
            return (
              <button
                type="button"
                key={resource.id}
                onClick={() => toggleResource(resource.id)}
                aria-pressed={isSelected}
                className={`flex items-center gap-2 px-4 py-2 rounded-lg border transition-all ${
                  isSelected
                    ? 'bg-cyan-600/30 border-cyan-500 text-cyan-300'
                    : 'bg-slate-700/50 border-slate-600 text-gray-400 hover:border-slate-500'
                }`}
              >
                <Icon size={16} />
                {resource.label}
              </button>
            );
          })}
        </div>
      </div>

      {showDiskInput && (
        <div>
          <label
            htmlFor="disk-per-cell"
            className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
          >
            Disk per Cell (GB)
          </label>
          <input
            id="disk-per-cell"
            type="number"
            value={customDisk}
            onChange={(e) => setCustomDisk(Number(e.target.value))}
            min={32}
            className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2.5 text-gray-200 font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors max-w-xs"
          />
        </div>
      )}

      <div className="flex justify-end gap-3 pt-4">
        <button
          type="button"
          onClick={onSkip}
          className="px-6 py-2.5 text-gray-400 hover:text-gray-300 transition-colors font-medium"
        >
          Skip
        </button>
        <button
          type="button"
          onClick={onContinue}
          className="flex items-center gap-2 px-6 py-2.5 bg-cyan-600 text-white rounded-lg hover:bg-cyan-500 transition-colors font-medium"
        >
          Continue
          <ArrowRight size={16} />
        </button>
      </div>
    </div>
  );
};

export default ResourceTypesStep;
```

### Step 3.4: Run test to verify it passes

```bash
cd frontend && npm test -- ResourceTypesStep
```

Expected: PASS (6 tests)

### Step 3.5: Commit

```bash
git add frontend/src/components/wizard/steps/
git commit -m "feat: add ResourceTypesStep component for resource selection"
```

---

## Task 4: AdvancedStep Component

**Files:**
- Create: `frontend/src/components/wizard/steps/AdvancedStep.jsx`
- Create: `frontend/src/components/wizard/steps/AdvancedStep.test.jsx`

### Step 4.1: Write the failing test

```jsx
// frontend/src/components/wizard/steps/AdvancedStep.test.jsx
// ABOUTME: Tests for advanced options step in scenario wizard
// ABOUTME: Covers overhead slider, hypothetical app, and TPS curve

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import AdvancedStep from './AdvancedStep';

describe('AdvancedStep', () => {
  const defaultProps = {
    overheadPct: 7,
    setOverheadPct: vi.fn(),
    useAdditionalApp: false,
    setUseAdditionalApp: vi.fn(),
    additionalApp: { name: 'test-app', instances: 1, memoryGB: 1, diskGB: 1 },
    setAdditionalApp: vi.fn(),
    tpsCurve: [{ cells: 50, tps: 500 }],
    setTPSCurve: vi.fn(),
    onContinue: vi.fn(),
    onSkip: vi.fn(),
  };

  it('renders overhead slider', () => {
    render(<AdvancedStep {...defaultProps} />);
    expect(screen.getByLabelText(/memory overhead/i)).toBeInTheDocument();
  });

  it('displays current overhead percentage', () => {
    render(<AdvancedStep {...defaultProps} overheadPct={10} />);
    expect(screen.getByText(/10%/)).toBeInTheDocument();
  });

  it('renders hypothetical app section', () => {
    render(<AdvancedStep {...defaultProps} />);
    expect(screen.getByText(/hypothetical app/i)).toBeInTheDocument();
  });

  it('shows app inputs when checkbox is checked', async () => {
    render(<AdvancedStep {...defaultProps} useAdditionalApp={true} />);
    expect(screen.getByLabelText(/app name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/instances/i)).toBeInTheDocument();
  });

  it('renders TPS curve section', () => {
    render(<AdvancedStep {...defaultProps} />);
    expect(screen.getByText(/tps performance curve/i)).toBeInTheDocument();
  });

  it('calls onSkip when Skip button clicked', async () => {
    const onSkip = vi.fn();
    render(<AdvancedStep {...defaultProps} onSkip={onSkip} />);
    await userEvent.click(screen.getByRole('button', { name: /skip/i }));
    expect(onSkip).toHaveBeenCalled();
  });

  it('calls onContinue when Continue button clicked', async () => {
    const onContinue = vi.fn();
    render(<AdvancedStep {...defaultProps} onContinue={onContinue} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(onContinue).toHaveBeenCalled();
  });
});
```

### Step 4.2: Run test to verify it fails

```bash
cd frontend && npm test -- AdvancedStep
```

Expected: FAIL - module not found

### Step 4.3: Write minimal implementation

```jsx
// frontend/src/components/wizard/steps/AdvancedStep.jsx
// ABOUTME: Step 3 of scenario wizard - advanced configuration options
// ABOUTME: Handles memory overhead, hypothetical app, and TPS curve

import { ArrowRight, Plus, X } from 'lucide-react';
import { DEFAULT_TPS_CURVE } from '../../../config/resourceConfig';

const AdvancedStep = ({
  overheadPct,
  setOverheadPct,
  useAdditionalApp,
  setUseAdditionalApp,
  additionalApp,
  setAdditionalApp,
  tpsCurve,
  setTPSCurve,
  onContinue,
  onSkip,
}) => {
  const updateTPSPoint = (index, field, value) => {
    setTPSCurve((prev) =>
      prev.map((pt, i) => (i === index ? { ...pt, [field]: parseInt(value) || 0 } : pt))
    );
  };

  const addTPSPoint = () => {
    const lastPt = tpsCurve[tpsCurve.length - 1] || { cells: 0, tps: 0 };
    setTPSCurve([...tpsCurve, { cells: lastPt.cells + 50, tps: Math.max(50, lastPt.tps - 100) }]);
  };

  const removeTPSPoint = (index) => {
    if (tpsCurve.length > 2) {
      setTPSCurve((prev) => prev.filter((_, i) => i !== index));
    }
  };

  const resetTPSCurve = () => {
    setTPSCurve(DEFAULT_TPS_CURVE);
  };

  return (
    <div className="space-y-6">
      {/* Memory Overhead Slider */}
      <div>
        <label
          htmlFor="overhead-slider"
          className="block text-xs uppercase tracking-wider font-medium text-gray-400 mb-2"
        >
          Memory Overhead: {overheadPct}%
        </label>
        <input
          id="overhead-slider"
          type="range"
          value={overheadPct}
          onChange={(e) => setOverheadPct(Number(e.target.value))}
          min={1}
          max={20}
          step={0.5}
          className="w-full h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-cyan-500"
        />
        <div className="flex justify-between text-xs text-gray-500 mt-1">
          <span>1%</span>
          <span>Default: 7%</span>
          <span>20%</span>
        </div>
      </div>

      {/* Hypothetical App Section */}
      <div className="bg-slate-700/30 rounded-lg p-4">
        <h4 className="text-sm font-medium text-gray-300 mb-3">Hypothetical App</h4>
        <div className="flex items-center gap-2 mb-3">
          <input
            type="checkbox"
            id="use-app"
            checked={useAdditionalApp}
            onChange={(e) => setUseAdditionalApp(e.target.checked)}
            className="rounded border-slate-600 bg-slate-700 text-cyan-500 focus:ring-cyan-500"
          />
          <label htmlFor="use-app" className="text-sm text-gray-300">
            Include in analysis
          </label>
        </div>

        {useAdditionalApp && (
          <div className="grid grid-cols-2 gap-3">
            <div className="col-span-2">
              <label htmlFor="app-name" className="block text-xs text-gray-400 mb-1">
                App Name
              </label>
              <input
                id="app-name"
                type="text"
                value={additionalApp.name}
                onChange={(e) => setAdditionalApp({ ...additionalApp, name: e.target.value })}
                className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm focus:border-cyan-500 outline-none"
              />
            </div>
            <div>
              <label htmlFor="app-instances" className="block text-xs text-gray-400 mb-1">
                Instances
              </label>
              <input
                id="app-instances"
                type="number"
                value={additionalApp.instances}
                onChange={(e) =>
                  setAdditionalApp({ ...additionalApp, instances: Number(e.target.value) })
                }
                min={1}
                className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
              />
            </div>
            <div>
              <label htmlFor="app-memory" className="block text-xs text-gray-400 mb-1">
                Memory/Instance (GB)
              </label>
              <input
                id="app-memory"
                type="number"
                value={additionalApp.memoryGB}
                onChange={(e) =>
                  setAdditionalApp({ ...additionalApp, memoryGB: Number(e.target.value) })
                }
                min={1}
                className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
              />
            </div>
            <div>
              <label htmlFor="app-disk" className="block text-xs text-gray-400 mb-1">
                Disk/Instance (GB)
              </label>
              <input
                id="app-disk"
                type="number"
                value={additionalApp.diskGB}
                onChange={(e) =>
                  setAdditionalApp({ ...additionalApp, diskGB: Number(e.target.value) })
                }
                min={1}
                className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
              />
            </div>
          </div>
        )}
      </div>

      {/* TPS Curve Section */}
      <div className="bg-slate-700/30 rounded-lg p-4">
        <h4 className="text-sm font-medium text-gray-300 mb-3">TPS Performance Curve</h4>
        <p className="text-xs text-gray-400 mb-3">
          Customize to match your observed scheduler performance.
        </p>

        <div className="space-y-2 mb-3">
          {tpsCurve.map((pt, i) => (
            <div key={i} className="flex items-center gap-2">
              <input
                type="number"
                value={pt.cells}
                onChange={(e) => updateTPSPoint(i, 'cells', e.target.value)}
                aria-label={`TPS point ${i + 1} cells`}
                className="w-24 bg-slate-700 border border-slate-600 rounded px-2 py-1 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
              />
              <span className="text-gray-500">cells →</span>
              <input
                type="number"
                value={pt.tps}
                onChange={(e) => updateTPSPoint(i, 'tps', e.target.value)}
                aria-label={`TPS point ${i + 1} tps`}
                className="w-24 bg-slate-700 border border-slate-600 rounded px-2 py-1 text-gray-200 text-sm font-mono focus:border-cyan-500 outline-none"
              />
              <span className="text-gray-500">TPS</span>
              {tpsCurve.length > 2 && (
                <button
                  type="button"
                  onClick={() => removeTPSPoint(i)}
                  className="text-red-400 hover:text-red-300 p-1"
                  aria-label={`Remove TPS point ${i + 1}`}
                >
                  <X size={14} />
                </button>
              )}
            </div>
          ))}
        </div>

        <div className="flex gap-2">
          <button
            type="button"
            onClick={addTPSPoint}
            className="text-xs text-cyan-400 hover:text-cyan-300 flex items-center gap-1"
          >
            <Plus size={12} /> Add Point
          </button>
          <button
            type="button"
            onClick={resetTPSCurve}
            className="text-xs text-gray-400 hover:text-gray-300"
          >
            Reset to Default
          </button>
        </div>
      </div>

      <div className="flex justify-end gap-3 pt-4">
        <button
          type="button"
          onClick={onSkip}
          className="px-6 py-2.5 text-gray-400 hover:text-gray-300 transition-colors font-medium"
        >
          Skip
        </button>
        <button
          type="button"
          onClick={onContinue}
          className="flex items-center gap-2 px-6 py-2.5 bg-cyan-600 text-white rounded-lg hover:bg-cyan-500 transition-colors font-medium"
        >
          Continue
          <ArrowRight size={16} />
        </button>
      </div>
    </div>
  );
};

export default AdvancedStep;
```

### Step 4.4: Run test to verify it passes

```bash
cd frontend && npm test -- AdvancedStep
```

Expected: PASS (7 tests)

### Step 4.5: Commit

```bash
git add frontend/src/components/wizard/steps/
git commit -m "feat: add AdvancedStep component for overhead and TPS config"
```

---

## Task 5: ScenarioWizard Component

**Files:**
- Create: `frontend/src/components/wizard/ScenarioWizard.jsx`
- Create: `frontend/src/components/wizard/ScenarioWizard.test.jsx`

### Step 5.1: Write the failing test

```jsx
// frontend/src/components/wizard/ScenarioWizard.test.jsx
// ABOUTME: Tests for scenario wizard container component
// ABOUTME: Covers step navigation, state management, and step rendering

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ScenarioWizard from './ScenarioWizard';

describe('ScenarioWizard', () => {
  const defaultProps = {
    // Cell config props
    selectedPreset: 0,
    setSelectedPreset: vi.fn(),
    customCPU: 4,
    setCustomCPU: vi.fn(),
    customMemory: 32,
    setCustomMemory: vi.fn(),
    cellCount: 100,
    setCellCount: vi.fn(),
    equivalentCellSuggestion: null,
    // Resource props
    selectedResources: ['memory'],
    toggleResource: vi.fn(),
    customDisk: 128,
    setCustomDisk: vi.fn(),
    // Advanced props
    overheadPct: 7,
    setOverheadPct: vi.fn(),
    useAdditionalApp: false,
    setUseAdditionalApp: vi.fn(),
    additionalApp: { name: 'test', instances: 1, memoryGB: 1, diskGB: 1 },
    setAdditionalApp: vi.fn(),
    tpsCurve: [{ cells: 50, tps: 500 }],
    setTPSCurve: vi.fn(),
    onStepComplete: vi.fn(),
  };

  it('renders step indicator', () => {
    render(<ScenarioWizard {...defaultProps} />);
    expect(screen.getByText('Cell Config')).toBeInTheDocument();
    expect(screen.getByText('Resources')).toBeInTheDocument();
    expect(screen.getByText('Advanced')).toBeInTheDocument();
  });

  it('shows CellConfigStep initially', () => {
    render(<ScenarioWizard {...defaultProps} />);
    expect(screen.getByLabelText(/vm size/i)).toBeInTheDocument();
  });

  it('advances to ResourceTypesStep after continuing from Step 1', async () => {
    render(<ScenarioWizard {...defaultProps} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(screen.getByText(/which resources to analyze/i)).toBeInTheDocument();
  });

  it('advances to AdvancedStep after continuing from Step 2', async () => {
    render(<ScenarioWizard {...defaultProps} />);
    // Step 1 -> Step 2
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    // Step 2 -> Step 3
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(screen.getByLabelText(/memory overhead/i)).toBeInTheDocument();
  });

  it('calls onStepComplete after Step 1', async () => {
    const onStepComplete = vi.fn();
    render(<ScenarioWizard {...defaultProps} onStepComplete={onStepComplete} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    expect(onStepComplete).toHaveBeenCalledWith(0);
  });

  it('allows skipping optional steps', async () => {
    render(<ScenarioWizard {...defaultProps} />);
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    await userEvent.click(screen.getByRole('button', { name: /skip/i }));
    expect(screen.getByLabelText(/memory overhead/i)).toBeInTheDocument();
  });

  it('allows clicking on completed steps to navigate back', async () => {
    render(<ScenarioWizard {...defaultProps} />);
    // Go to step 2
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));
    // Click on step 1 in indicator
    await userEvent.click(screen.getByText('Cell Config'));
    // Should show step 1 content
    expect(screen.getByLabelText(/vm size/i)).toBeInTheDocument();
  });
});
```

### Step 5.2: Run test to verify it fails

```bash
cd frontend && npm test -- ScenarioWizard
```

Expected: FAIL - module not found

### Step 5.3: Write minimal implementation

```jsx
// frontend/src/components/wizard/ScenarioWizard.jsx
// ABOUTME: Container component for scenario configuration wizard
// ABOUTME: Manages step navigation and renders appropriate step content

import { useState, useCallback } from 'react';
import StepIndicator from './StepIndicator';
import CellConfigStep from './steps/CellConfigStep';
import ResourceTypesStep from './steps/ResourceTypesStep';
import AdvancedStep from './steps/AdvancedStep';

const STEPS = [
  { id: 'cell-config', label: 'Cell Config', required: true },
  { id: 'resources', label: 'Resources', required: false },
  { id: 'advanced', label: 'Advanced', required: false },
];

const ScenarioWizard = ({
  // Cell config props
  selectedPreset,
  setSelectedPreset,
  customCPU,
  setCustomCPU,
  customMemory,
  setCustomMemory,
  cellCount,
  setCellCount,
  equivalentCellSuggestion,
  // Resource props
  selectedResources,
  toggleResource,
  customDisk,
  setCustomDisk,
  // Advanced props
  overheadPct,
  setOverheadPct,
  useAdditionalApp,
  setUseAdditionalApp,
  additionalApp,
  setAdditionalApp,
  tpsCurve,
  setTPSCurve,
  // Callbacks
  onStepComplete,
}) => {
  const [currentStep, setCurrentStep] = useState(0);
  const [completedSteps, setCompletedSteps] = useState([]);

  const markStepComplete = useCallback(
    (stepIndex) => {
      if (!completedSteps.includes(stepIndex)) {
        setCompletedSteps((prev) => [...prev, stepIndex]);
      }
      onStepComplete?.(stepIndex);
    },
    [completedSteps, onStepComplete]
  );

  const handleContinue = useCallback(() => {
    markStepComplete(currentStep);
    if (currentStep < STEPS.length - 1) {
      setCurrentStep(currentStep + 1);
    }
  }, [currentStep, markStepComplete]);

  const handleSkip = useCallback(() => {
    if (currentStep < STEPS.length - 1) {
      setCurrentStep(currentStep + 1);
    }
  }, [currentStep]);

  const handleStepClick = useCallback((stepIndex) => {
    setCurrentStep(stepIndex);
  }, []);

  const renderStepContent = () => {
    switch (currentStep) {
      case 0:
        return (
          <CellConfigStep
            selectedPreset={selectedPreset}
            setSelectedPreset={setSelectedPreset}
            customCPU={customCPU}
            setCustomCPU={setCustomCPU}
            customMemory={customMemory}
            setCustomMemory={setCustomMemory}
            cellCount={cellCount}
            setCellCount={setCellCount}
            equivalentCellSuggestion={equivalentCellSuggestion}
            onContinue={handleContinue}
          />
        );
      case 1:
        return (
          <ResourceTypesStep
            selectedResources={selectedResources}
            toggleResource={toggleResource}
            customDisk={customDisk}
            setCustomDisk={setCustomDisk}
            onContinue={handleContinue}
            onSkip={handleSkip}
          />
        );
      case 2:
        return (
          <AdvancedStep
            overheadPct={overheadPct}
            setOverheadPct={setOverheadPct}
            useAdditionalApp={useAdditionalApp}
            setUseAdditionalApp={setUseAdditionalApp}
            additionalApp={additionalApp}
            setAdditionalApp={setAdditionalApp}
            tpsCurve={tpsCurve}
            setTPSCurve={setTPSCurve}
            onContinue={handleContinue}
            onSkip={handleSkip}
          />
        );
      default:
        return null;
    }
  };

  return (
    <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
      <StepIndicator
        steps={STEPS}
        currentStep={currentStep}
        completedSteps={completedSteps}
        onStepClick={handleStepClick}
      />
      {renderStepContent()}
    </div>
  );
};

export default ScenarioWizard;
```

### Step 5.4: Run test to verify it passes

```bash
cd frontend && npm test -- ScenarioWizard
```

Expected: PASS (7 tests)

### Step 5.5: Commit

```bash
git add frontend/src/components/wizard/
git commit -m "feat: add ScenarioWizard container with step navigation"
```

---

## Task 6: Integrate Wizard into ScenarioAnalyzer

**Files:**
- Modify: `frontend/src/components/ScenarioAnalyzer.jsx`
- Create: `frontend/src/components/ScenarioAnalyzer.test.jsx` (if not exists)

### Step 6.1: Write the failing test

```jsx
// frontend/src/components/ScenarioAnalyzer.test.jsx
// ABOUTME: Integration tests for ScenarioAnalyzer with wizard
// ABOUTME: Covers data loading, wizard display, and run analysis

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ScenarioAnalyzer from './ScenarioAnalyzer';
import { scenarioApi } from '../services/scenarioApi';

// Mock the API
vi.mock('../services/scenarioApi', () => ({
  scenarioApi: {
    setManualInfrastructure: vi.fn(),
    compareScenario: vi.fn(),
  },
}));

// Mock localStorage
const mockLocalStorage = {
  getItem: vi.fn(),
  setItem: vi.fn(),
};
Object.defineProperty(window, 'localStorage', { value: mockLocalStorage });

describe('ScenarioAnalyzer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockLocalStorage.getItem.mockReturnValue(null);
  });

  it('shows wizard after infrastructure is loaded', async () => {
    const mockData = {
      name: 'Test Infra',
      clusters: [{ diego_cell_count: 10, diego_cell_memory_gb: 64, diego_cell_cpu: 8 }],
    };
    mockLocalStorage.getItem.mockReturnValue(JSON.stringify(mockData));
    scenarioApi.setManualInfrastructure.mockResolvedValue({ ready: true });

    render(<ScenarioAnalyzer />);

    await waitFor(() => {
      expect(screen.getByText('Cell Config')).toBeInTheDocument();
    });
  });

  it('shows Run Analysis section after Step 1 completed', async () => {
    const mockData = {
      name: 'Test Infra',
      clusters: [{ diego_cell_count: 10, diego_cell_memory_gb: 64, diego_cell_cpu: 8 }],
    };
    mockLocalStorage.getItem.mockReturnValue(JSON.stringify(mockData));
    scenarioApi.setManualInfrastructure.mockResolvedValue({ ready: true });

    render(<ScenarioAnalyzer />);

    await waitFor(() => {
      expect(screen.getByText('Cell Config')).toBeInTheDocument();
    });

    // Complete Step 1
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /run analysis/i })).toBeInTheDocument();
    });
  });

  it('displays config summary in Run Analysis section', async () => {
    const mockData = {
      name: 'Test Infra',
      clusters: [{ diego_cell_count: 10, diego_cell_memory_gb: 64, diego_cell_cpu: 8 }],
    };
    mockLocalStorage.getItem.mockReturnValue(JSON.stringify(mockData));
    scenarioApi.setManualInfrastructure.mockResolvedValue({ ready: true });

    render(<ScenarioAnalyzer />);

    await waitFor(() => {
      expect(screen.getByText('Cell Config')).toBeInTheDocument();
    });

    // Complete Step 1
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));

    await waitFor(() => {
      // Summary should show cell count
      expect(screen.getByText(/10 cells/i)).toBeInTheDocument();
    });
  });
});
```

### Step 6.2: Run test to verify it fails

```bash
cd frontend && npm test -- ScenarioAnalyzer
```

Expected: FAIL - wizard components not rendered in current implementation

### Step 6.3: Refactor ScenarioAnalyzer to use wizard

This is a significant refactor. Replace the inline form with ScenarioWizard:

**Key changes:**
1. Remove inline form JSX from "Proposed Configuration" section
2. Import and render ScenarioWizard
3. Add `step1Completed` state to control Run Analysis visibility
4. Add summary display in Run Analysis section

```jsx
// In ScenarioAnalyzer.jsx, add these changes:

// Add import at top:
import ScenarioWizard from './wizard/ScenarioWizard';

// Add state after existing state declarations:
const [step1Completed, setStep1Completed] = useState(false);

// Add handler:
const handleStepComplete = (stepIndex) => {
  if (stepIndex === 0) {
    setStep1Completed(true);
  }
};

// Replace the "Proposed Configuration" section with:
{infrastructureState && (
  <ScenarioWizard
    selectedPreset={selectedPreset}
    setSelectedPreset={setSelectedPreset}
    customCPU={customCPU}
    setCustomCPU={setCustomCPU}
    customMemory={customMemory}
    setCustomMemory={setCustomMemory}
    cellCount={cellCount}
    setCellCount={setCellCount}
    equivalentCellSuggestion={equivalentCellSuggestion}
    selectedResources={selectedResources}
    toggleResource={toggleResource}
    customDisk={customDisk}
    setCustomDisk={setCustomDisk}
    overheadPct={overheadPct}
    setOverheadPct={setOverheadPct}
    useAdditionalApp={useAdditionalApp}
    setUseAdditionalApp={setUseAdditionalApp}
    additionalApp={additionalApp}
    setAdditionalApp={setAdditionalApp}
    tpsCurve={tpsCurve}
    setTPSCurve={setTPSCurve}
    onStepComplete={handleStepComplete}
  />
)}

// Add Run Analysis section after wizard:
{infrastructureState && step1Completed && (
  <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
    <div className="flex items-center justify-between">
      <div>
        <h3 className="text-lg font-semibold text-gray-200 flex items-center gap-2">
          <Sparkles size={18} className="text-cyan-400" />
          Ready to Analyze
        </h3>
        <p className="text-sm text-gray-400 mt-1">
          {preset.label}, {cellCount} cells | {selectedResources.join(', ')}
          {overheadPct !== 7 && ` | ${overheadPct}% overhead`}
        </p>
      </div>
      <div className="flex items-center gap-3">
        {comparison && (
          <button
            onClick={handleExportMarkdown}
            className="flex items-center gap-2 px-4 py-2 bg-slate-700 text-gray-200 rounded-lg hover:bg-slate-600 transition-colors border border-slate-600"
          >
            <FileDown size={16} />
            Export
          </button>
        )}
        <button
          onClick={handleCompare}
          disabled={loading}
          className="flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-cyan-600 to-blue-600 text-white rounded-lg hover:from-cyan-500 hover:to-blue-500 disabled:opacity-50 transition-all font-medium shadow-lg shadow-cyan-500/20"
        >
          {loading ? (
            <RefreshCw className="animate-spin" size={18} />
          ) : (
            <Sparkles size={18} />
          )}
          Run Analysis
        </button>
      </div>
    </div>
  </div>
)}
```

### Step 6.4: Run test to verify it passes

```bash
cd frontend && npm test -- ScenarioAnalyzer
```

Expected: PASS (3 tests)

### Step 6.5: Commit

```bash
git add frontend/src/components/
git commit -m "refactor: integrate ScenarioWizard into ScenarioAnalyzer"
```

---

## Task 7: Run Full Test Suite and Manual Verification

**Files:**
- None (verification only)

### Step 7.1: Run all tests

```bash
cd frontend && npm test
```

Expected: All tests pass

### Step 7.2: Run lint

```bash
cd frontend && npm run lint
```

Expected: No errors

### Step 7.3: Build for production

```bash
cd frontend && npm run build
```

Expected: Build succeeds

### Step 7.4: Manual testing

```bash
make frontend-preview
```

1. Load infrastructure data
2. Verify wizard shows with Step 1 active
3. Complete Step 1, verify Step 2 appears
4. Click back to Step 1, verify it works
5. Skip Step 2, verify Step 3 appears
6. Verify Run Analysis button appears after Step 1
7. Run analysis, verify results display

### Step 7.5: Final commit

```bash
git add -A && git status
git commit -m "chore: complete stepper wizard implementation for ScenarioAnalyzer"
```

---

## Summary

| Task | Description | Est. Time |
|------|-------------|-----------|
| 1 | StepIndicator component | 15 min |
| 2 | CellConfigStep component | 15 min |
| 3 | ResourceTypesStep component | 15 min |
| 4 | AdvancedStep component | 20 min |
| 5 | ScenarioWizard container | 20 min |
| 6 | Integrate into ScenarioAnalyzer | 30 min |
| 7 | Full test suite and verification | 15 min |

**Total estimated time:** ~2-2.5 hours

**Files created:**
- `frontend/src/components/wizard/StepIndicator.jsx`
- `frontend/src/components/wizard/StepIndicator.test.jsx`
- `frontend/src/components/wizard/ScenarioWizard.jsx`
- `frontend/src/components/wizard/ScenarioWizard.test.jsx`
- `frontend/src/components/wizard/steps/CellConfigStep.jsx`
- `frontend/src/components/wizard/steps/CellConfigStep.test.jsx`
- `frontend/src/components/wizard/steps/ResourceTypesStep.jsx`
- `frontend/src/components/wizard/steps/ResourceTypesStep.test.jsx`
- `frontend/src/components/wizard/steps/AdvancedStep.jsx`
- `frontend/src/components/wizard/steps/AdvancedStep.test.jsx`
- `frontend/src/components/ScenarioAnalyzer.test.jsx`

**Files modified:**
- `frontend/src/components/ScenarioAnalyzer.jsx`
