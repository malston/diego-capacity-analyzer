# Implementation Plan: Add CPU and Host-Level Capacity Analysis

**Track ID:** cpu-host-analysis_20251224
**GitHub Issue:** https://github.com/malston/diego-capacity-analyzer/issues/10

---

## Phase 1: Backend - CPU Analysis Models and Calculations

### Task 1.1: Extend models with CPU fields
- [ ] Write tests for CPU-related model fields (cores per host, vCPU:pCPU ratio)
- [ ] Add CPU fields to infrastructure and scenario models
- [ ] Verify model serialization/deserialization

### Task 1.2: Implement CPU utilization calculations
- [ ] Write tests for CPU utilization percentage calculation
- [ ] Write tests for vCPU:pCPU ratio calculation
- [ ] Write tests for CPU risk level thresholds (low/medium/high)
- [ ] Implement CPU calculation functions in planning service

### Task 1.3: Extend API responses with CPU metrics
- [ ] Write tests for API response including CPU metrics
- [ ] Update infrastructure and scenario API handlers
- [ ] Verify backward compatibility with existing clients

- [ ] Task: Conductor - User Manual Verification 'Phase 1: Backend - CPU Analysis Models and Calculations' (Protocol in workflow.md)

---

## Phase 2: Backend - Host-Level Analysis

### Task 2.1: Add host-level model fields
- [ ] Write tests for host-level fields (host count, cores per host, memory per host)
- [ ] Add HA admission control percentage field
- [ ] Extend infrastructure models with host configuration

### Task 2.2: Implement host utilization calculations
- [ ] Write tests for host memory utilization calculation
- [ ] Write tests for host CPU utilization calculation
- [ ] Write tests for VMs per host calculation
- [ ] Implement host utilization functions

### Task 2.3: Implement HA capacity analysis
- [ ] Write tests for HA headroom calculation (survive N host failures)
- [ ] Implement HA capacity analysis
- [ ] Add HA status to API responses

### Task 2.4: Integrate vSphere host data
- [ ] Write tests for host data extraction from vSphere
- [ ] Update vSphere service to fetch host-level metrics
- [ ] Map vSphere host data to host analysis models

- [ ] Task: Conductor - User Manual Verification 'Phase 2: Backend - Host-Level Analysis' (Protocol in workflow.md)

---

## Phase 3: Backend - Multi-Resource Bottleneck and Recommendations

### Task 3.1: Implement resource exhaustion ordering
- [ ] Write tests for ranking resources by utilization percentage
- [ ] Write tests for identifying the constraining resource
- [ ] Implement resource exhaustion ordering logic

### Task 3.2: Implement upgrade path recommendations
- [ ] Write tests for "add cells" recommendation logic
- [ ] Write tests for "resize cells" recommendation logic
- [ ] Write tests for "add hosts" recommendation logic
- [ ] Implement recommendation engine with priority ordering

### Task 3.3: Add recommendations to API
- [ ] Write tests for recommendations in API response
- [ ] Update scenario comparison API to include recommendations
- [ ] Add bottleneck summary to infrastructure status endpoint

- [ ] Task: Conductor - User Manual Verification 'Phase 3: Backend - Multi-Resource Bottleneck and Recommendations' (Protocol in workflow.md)

---

## Phase 4: Frontend - CPU and Host UI Components

### Task 4.1: Add CPU inputs to scenario wizard
- [ ] Write tests for CPU input fields in wizard
- [ ] Add physical cores per host input
- [ ] Add number of hosts input
- [ ] Add target vCPU:pCPU ratio input

### Task 4.2: Create CPU utilization gauge component
- [ ] Write tests for CPU gauge rendering
- [ ] Write tests for gauge color based on risk level
- [ ] Implement CPU utilization gauge (similar to memory gauge)
- [ ] Add vCPU:pCPU ratio indicator with color coding

### Task 4.3: Add host-level inputs (optional section)
- [ ] Write tests for collapsible host configuration section
- [ ] Add host count, cores per host, memory per host inputs
- [ ] Add HA admission control percentage input
- [ ] Make section collapsible with sensible defaults

### Task 4.4: Create host analysis display component
- [ ] Write tests for host metrics display
- [ ] Implement host utilization cards
- [ ] Display VMs per host and HA capacity status

### Task 4.5: Create multi-resource bottleneck component
- [ ] Write tests for resource exhaustion ordering display
- [ ] Write tests for bottleneck highlighting
- [ ] Implement ranked resource list with visual indicators
- [ ] Highlight the constraining resource

### Task 4.6: Create upgrade recommendations component
- [ ] Write tests for recommendations display
- [ ] Implement actionable recommendation cards
- [ ] Add recommendation priority ordering

- [ ] Task: Conductor - User Manual Verification 'Phase 4: Frontend - CPU and Host UI Components' (Protocol in workflow.md)

---

## Phase 5: Integration and Polish

### Task 5.1: End-to-end integration testing
- [ ] Write E2E tests for CPU analysis flow
- [ ] Write E2E tests for host-level analysis flow
- [ ] Write E2E tests for bottleneck and recommendations flow
- [ ] Verify all data flows correctly from vSphere to UI

### Task 5.2: Update sample scenarios
- [ ] Add CPU and host data to existing sample files
- [ ] Create new sample demonstrating CPU-constrained scenario
- [ ] Create new sample demonstrating host-constrained scenario

### Task 5.3: Documentation updates
- [ ] Update UI Guide with new metrics and components
- [ ] Update README with new environment variables
- [ ] Add inline help text for new wizard inputs

### Task 5.4: Final polish and edge cases
- [ ] Handle missing host data gracefully (show N/A, not errors)
- [ ] Ensure backward compatibility with existing infrastructure JSON
- [ ] Verify tooltips and hover states on new components

- [ ] Task: Conductor - User Manual Verification 'Phase 5: Integration and Polish' (Protocol in workflow.md)

---

## Notes

- All tasks follow TDD workflow: write failing tests first, then implement
- Each phase ends with manual verification per workflow.md protocol
- vSphere integration for hosts builds on existing govmomi code
- UI components should match existing gauge and card styling
