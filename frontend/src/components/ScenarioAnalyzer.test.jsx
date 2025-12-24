// ABOUTME: Integration tests for ScenarioAnalyzer with wizard
// ABOUTME: Covers data loading, wizard display, and run analysis

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ScenarioAnalyzer from './ScenarioAnalyzer';
import { ToastProvider } from '../contexts/ToastContext';
import { scenarioApi } from '../services/scenarioApi';

// Helper to render with providers
const renderWithProviders = (ui) => {
  return render(<ToastProvider>{ui}</ToastProvider>);
};

// Mock the API
vi.mock('../services/scenarioApi', () => ({
  scenarioApi: {
    setManualInfrastructure: vi.fn(),
    compareScenario: vi.fn(),
    getInfrastructureStatus: vi.fn().mockResolvedValue({ vsphere_configured: false }),
  },
}));

// Mock localStorage
const mockLocalStorage = (() => {
  let store = {};
  return {
    getItem: vi.fn((key) => store[key] || null),
    setItem: vi.fn((key, value) => { store[key] = value; }),
    clear: vi.fn(() => { store = {}; }),
  };
})();
Object.defineProperty(window, 'localStorage', { value: mockLocalStorage });

describe('ScenarioAnalyzer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockLocalStorage.clear();
  });

  it('shows wizard after infrastructure is loaded', async () => {
    const mockData = {
      name: 'Test Infra',
      clusters: [{ diego_cell_count: 10, diego_cell_memory_gb: 64, diego_cell_cpu: 8 }],
    };
    mockLocalStorage.getItem.mockReturnValue(JSON.stringify(mockData));
    scenarioApi.setManualInfrastructure.mockResolvedValue({ ready: true });

    renderWithProviders(<ScenarioAnalyzer />);

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

    renderWithProviders(<ScenarioAnalyzer />);

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

    renderWithProviders(<ScenarioAnalyzer />);

    await waitFor(() => {
      expect(screen.getByText('Cell Config')).toBeInTheDocument();
    });

    // Complete Step 1
    await userEvent.click(screen.getByRole('button', { name: /continue/i }));

    await waitFor(() => {
      // Summary should show cell count in the "Ready to Analyze" section
      expect(screen.getByText('Ready to Analyze')).toBeInTheDocument();
      // Use getAllByText since there may be multiple instances
      const cellTexts = screen.getAllByText(/10 cells/i);
      expect(cellTexts.length).toBeGreaterThan(0);
    });
  });
});
