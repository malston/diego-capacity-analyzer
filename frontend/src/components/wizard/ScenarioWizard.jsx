// ABOUTME: Container component for scenario configuration wizard
// ABOUTME: Manages step navigation and renders appropriate step content

import { useState, useCallback, useMemo } from 'react';
import StepIndicator from './StepIndicator';
import CellConfigStep from './steps/CellConfigStep';
import ResourceTypesStep from './steps/ResourceTypesStep';
import CPUConfigStep from './steps/CPUConfigStep';
import AdvancedStep from './steps/AdvancedStep';

const BASE_STEPS = [
  { id: 'resources', label: 'Resources', required: true },
  { id: 'cell-config', label: 'Cell Config', required: true },
  { id: 'advanced', label: 'Advanced', required: false },
];

const CPU_STEP = { id: 'cpu-config', label: 'CPU Config', required: false };

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
  // CPU config props
  physicalCoresPerHost,
  setPhysicalCoresPerHost,
  hostCount,
  setHostCount,
  targetVCPURatio,
  setTargetVCPURatio,
  // Host config props (for Advanced step)
  memoryPerHost,
  setMemoryPerHost,
  haAdmissionPct,
  setHaAdmissionPct,
  // Infrastructure data for current ratio calculation
  totalVCPUs,
  // Advanced props
  overheadPct,
  setOverheadPct,
  useAdditionalApp,
  setUseAdditionalApp,
  additionalApp,
  setAdditionalApp,
  tpsCurve,
  setTPSCurve,
  enableTPS,
  setEnableTPS,
  // Callbacks
  onStepComplete,
}) => {
  const [currentStep, setCurrentStep] = useState(0);
  const [completedSteps, setCompletedSteps] = useState([]);

  // Dynamically build steps based on selected resources
  const steps = useMemo(() => {
    const result = [BASE_STEPS[0], BASE_STEPS[1]]; // Resources, Cell Config
    if (selectedResources.includes('cpu')) {
      result.push(CPU_STEP);
    }
    result.push(BASE_STEPS[2]); // Advanced
    return result;
  }, [selectedResources]);

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
    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1);
    }
  }, [currentStep, markStepComplete, steps.length]);

  const handleStepClick = useCallback((stepIndex) => {
    setCurrentStep(stepIndex);
  }, []);

  const renderStepContent = () => {
    const currentStepId = steps[currentStep]?.id;
    const isLastStep = currentStep === steps.length - 1;

    switch (currentStepId) {
      case 'resources':
        return (
          <ResourceTypesStep
            selectedResources={selectedResources}
            toggleResource={toggleResource}
            customDisk={customDisk}
            setCustomDisk={setCustomDisk}
            onContinue={handleContinue}
          />
        );
      case 'cell-config':
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
      case 'cpu-config':
        return (
          <CPUConfigStep
            physicalCoresPerHost={physicalCoresPerHost}
            setPhysicalCoresPerHost={setPhysicalCoresPerHost}
            hostCount={hostCount}
            setHostCount={setHostCount}
            targetVCPURatio={targetVCPURatio}
            setTargetVCPURatio={setTargetVCPURatio}
            totalVCPUs={totalVCPUs}
            onContinue={handleContinue}
            onSkip={handleContinue}
          />
        );
      case 'advanced':
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
            enableTPS={enableTPS}
            setEnableTPS={setEnableTPS}
            hostCount={hostCount}
            setHostCount={setHostCount}
            coresPerHost={physicalCoresPerHost}
            setCoresPerHost={setPhysicalCoresPerHost}
            memoryPerHost={memoryPerHost}
            setMemoryPerHost={setMemoryPerHost}
            haAdmissionPct={haAdmissionPct}
            setHaAdmissionPct={setHaAdmissionPct}
            isLastStep={isLastStep}
          />
        );
      default:
        return null;
    }
  };

  return (
    <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
      <StepIndicator
        steps={steps}
        currentStep={currentStep}
        completedSteps={completedSteps}
        onStepClick={handleStepClick}
      />
      {renderStepContent()}
    </div>
  );
};

export default ScenarioWizard;
