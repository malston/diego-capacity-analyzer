// ABOUTME: Integration tests for ScenarioAnalyzer with wizard
// ABOUTME: Covers data loading, wizard display, and run analysis

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ScenarioAnalyzer from "./ScenarioAnalyzer";
import { ToastProvider } from "../contexts/ToastContext";
import { scenarioApi } from "../services/scenarioApi";

// Helper to render with providers
const renderWithProviders = (ui) => {
  return render(<ToastProvider>{ui}</ToastProvider>);
};

// Mock the API
vi.mock("../services/scenarioApi", () => ({
  scenarioApi: {
    setManualInfrastructure: vi.fn(),
    compareScenario: vi.fn(),
    getInfrastructureStatus: vi
      .fn()
      .mockResolvedValue({ vsphere_configured: false }),
  },
}));

// Mock localStorage
const mockLocalStorage = (() => {
  let store = {};
  return {
    getItem: vi.fn((key) => store[key] || null),
    setItem: vi.fn((key, value) => {
      store[key] = value;
    }),
    clear: vi.fn(() => {
      store = {};
    }),
  };
})();
Object.defineProperty(window, "localStorage", { value: mockLocalStorage });

describe("ScenarioAnalyzer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockLocalStorage.clear();
  });

  it("shows wizard after infrastructure is loaded", async () => {
    const mockData = {
      name: "Test Infra",
      clusters: [
        { diego_cell_count: 10, diego_cell_memory_gb: 64, diego_cell_cpu: 8 },
      ],
    };
    mockLocalStorage.getItem.mockReturnValue(JSON.stringify(mockData));
    scenarioApi.setManualInfrastructure.mockResolvedValue({ ready: true });

    renderWithProviders(<ScenarioAnalyzer />);

    await waitFor(() => {
      expect(screen.getByText("Cell Config")).toBeInTheDocument();
    });
  });

  it("shows Run Analysis section when infrastructure is loaded", async () => {
    const mockData = {
      name: "Test Infra",
      clusters: [
        { diego_cell_count: 10, diego_cell_memory_gb: 64, diego_cell_cpu: 8 },
      ],
    };
    mockLocalStorage.getItem.mockReturnValue(JSON.stringify(mockData));
    scenarioApi.setManualInfrastructure.mockResolvedValue({ ready: true });

    renderWithProviders(<ScenarioAnalyzer />);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /run analysis/i }),
      ).toBeInTheDocument();
    });
  });

  it("displays config summary in Run Analysis section", async () => {
    const mockData = {
      name: "Test Infra",
      clusters: [
        { diego_cell_count: 10, diego_cell_memory_gb: 64, diego_cell_cpu: 8 },
      ],
    };
    mockLocalStorage.getItem.mockReturnValue(JSON.stringify(mockData));
    scenarioApi.setManualInfrastructure.mockResolvedValue({ ready: true });

    renderWithProviders(<ScenarioAnalyzer />);

    await waitFor(() => {
      // Summary should show cell count in the "Ready to Analyze" section
      expect(screen.getByText("Ready to Analyze")).toBeInTheDocument();
      // Use getAllByText since there may be multiple instances
      const cellTexts = screen.getAllByText(/10 cells/i);
      expect(cellTexts.length).toBeGreaterThan(0);
    });
  });

  describe("Cell Config Auto-Population (Issue #35)", () => {
    it("auto-populates cell count from live vSphere data with matching preset", async () => {
      // Mock infrastructure data with 5 cells × 16GB (Small Footprint TAS style)
      const mockData = {
        name: "Test vSphere",
        clusters: [
          {
            diego_cell_count: 5,
            diego_cell_cpu: 4,
            diego_cell_memory_gb: 16,
            host_count: 3,
            memory_gb: 1536,
          },
        ],
      };
      mockLocalStorage.getItem.mockReturnValue(JSON.stringify(mockData));
      scenarioApi.setManualInfrastructure.mockResolvedValue({
        ready: true,
        total_cell_count: 5,
      });

      renderWithProviders(<ScenarioAnalyzer />);

      // Wait for wizard to load (starts at Resources step)
      await waitFor(() => {
        expect(screen.getByText("Cell Config")).toBeInTheDocument();
      });

      // Click Continue on Resources step to advance to Cell Config step
      await userEvent.click(screen.getByRole("button", { name: /continue/i }));

      // Wait for Cell Config step to be visible with its inputs
      const cellCountInput = await screen.findByRole("spinbutton", {
        name: /cell count/i,
      });
      expect(cellCountInput).toHaveValue(5);

      // The VM size preset should match 4×16
      const vmSizeSelect = screen.getByRole("combobox", { name: /vm size/i });
      expect(vmSizeSelect).toHaveValue("0"); // Index 0 = 4×16 preset
    });

    it("uses Custom preset for non-standard cell sizes", async () => {
      // Mock infrastructure data with non-standard 6×24GB cells
      const mockData = {
        name: "Custom vSphere",
        clusters: [
          {
            diego_cell_count: 10,
            diego_cell_cpu: 6,
            diego_cell_memory_gb: 24,
            host_count: 4,
            memory_gb: 2048,
          },
        ],
      };
      mockLocalStorage.getItem.mockReturnValue(JSON.stringify(mockData));
      scenarioApi.setManualInfrastructure.mockResolvedValue({
        ready: true,
        total_cell_count: 10,
      });

      renderWithProviders(<ScenarioAnalyzer />);

      // Wait for wizard to load (starts at Resources step)
      await waitFor(() => {
        expect(screen.getByText("Cell Config")).toBeInTheDocument();
      });

      // Click Continue on Resources step to advance to Cell Config step
      await userEvent.click(screen.getByRole("button", { name: /continue/i }));

      // Wait for Cell Config step to be visible with its inputs
      const cellCountInput = await screen.findByRole("spinbutton", {
        name: /cell count/i,
      });
      expect(cellCountInput).toHaveValue(10);

      // The VM size preset should be Custom (last index = 5)
      const vmSizeSelect = screen.getByRole("combobox", { name: /vm size/i });
      expect(vmSizeSelect).toHaveValue("5"); // Index 5 = Custom preset
    });

    it("cell count matches between Current Config and Ready to Analyze", async () => {
      // This test verifies the bug is fixed: cell count should be consistent
      const mockData = {
        name: "Consistency Test",
        clusters: [
          {
            diego_cell_count: 5,
            diego_cell_cpu: 4,
            diego_cell_memory_gb: 16,
            host_count: 3,
            memory_gb: 1536,
          },
        ],
      };
      mockLocalStorage.getItem.mockReturnValue(JSON.stringify(mockData));
      scenarioApi.setManualInfrastructure.mockResolvedValue({
        ready: true,
        total_cell_count: 5,
      });

      renderWithProviders(<ScenarioAnalyzer />);

      await waitFor(() => {
        expect(screen.getByText("Cell Config")).toBeInTheDocument();
      });

      // Complete Step 1
      await userEvent.click(screen.getByRole("button", { name: /continue/i }));

      await waitFor(() => {
        expect(screen.getByText("Ready to Analyze")).toBeInTheDocument();
        // The summary should show 5 cells, not 3
        const cellTexts = screen.getAllByText(/5 cells/i);
        expect(cellTexts.length).toBeGreaterThan(0);
      });
    });
  });
});
