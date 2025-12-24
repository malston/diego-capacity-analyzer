// ABOUTME: Container component for scenario configuration wizard
// ABOUTME: Manages step navigation and renders appropriate step content

import { useState, useCallback } from 'react';
import StepIndicator from './StepIndicator';
import CellConfigStep from './steps/CellConfigStep';
import ResourceTypesStep from './steps/ResourceTypesStep';
import AdvancedStep from './steps/AdvancedStep';

const STEPS = [
  { id: 'resources', label: 'Resources', required: true },
  { id: 'cell-config', label: 'Cell Config', required: true },
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

  const handleStepClick = useCallback((stepIndex) => {
    setCurrentStep(stepIndex);
  }, []);

  const renderStepContent = () => {
    switch (currentStep) {
      case 0:
        return (
          <ResourceTypesStep
            selectedResources={selectedResources}
            toggleResource={toggleResource}
            customDisk={customDisk}
            setCustomDisk={setCustomDisk}
            onContinue={handleContinue}
          />
        );
      case 1:
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
            isLastStep={true}
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
