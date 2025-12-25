# Implementation Plan: Add CPU and Host-Level Capacity Analysis

**Track ID:** cpu-host-analysis_20251224
**GitHub Issue:** https://github.com/malston/diego-capacity-analyzer/issues/10

---

## Phase 1: Backend - CPU Analysis Models and Calculations [checkpoint: 2df9807]

### Task 1.1: Extend models with CPU fields
- [x] Write tests for CPU-related model fields (cores per host, vCPU:pCPU ratio) [4bd595b]
- [x] Add CPU fields to infrastructure and scenario models [4bd595b]
- [x] Verify model serialization/deserialization [4bd595b]

### Task 1.2: Implement CPU utilization calculations
- [x] Write tests for CPU utilization percentage calculation [d274056]
- [x] Write tests for vCPU:pCPU ratio calculation [4bd595b]
- [x] Write tests for CPU risk level thresholds (low/medium/high) [4bd595b]
- [x] Implement CPU calculation functions in planning service [d274056]

### Task 1.3: Extend API responses with CPU metrics
- [x] Write tests for API response including CPU metrics [04bfc48]
- [x] Update infrastructure and scenario API handlers [04bfc48]
- [x] Verify backward compatibility with existing clients [04bfc48]

- [x] Task: Conductor - User Manual Verification 'Phase 1: Backend - CPU Analysis Models and Calculations' (Protocol in workflow.md) [2df9807]

---

## Phase 2: Backend - Host-Level Analysis [checkpoint: b8aabf4]

### Task 2.1: Add host-level model fields
- [x] Write tests for host-level fields (host count, cores per host, memory per host) [73decc1]
- [x] Add HA admission control percentage field [73decc1]
- [x] Extend infrastructure models with host configuration [73decc1]

### Task 2.2: Implement host utilization calculations
- [x] Write tests for host memory utilization calculation [4eb8d94]
- [x] Write tests for host CPU utilization calculation [4eb8d94]
- [x] Write tests for VMs per host calculation [73decc1]
- [x] Implement host utilization functions [4eb8d94]

### Task 2.3: Implement HA capacity analysis
- [x] Write tests for HA headroom calculation (survive N host failures) [90fe279]
- [x] Implement HA capacity analysis [90fe279]
- [x] Add HA status to API responses [90fe279]

### Task 2.4: Integrate vSphere host data
- [x] Write tests for host data extraction from vSphere [5dba8a6]
- [x] Update vSphere service to fetch host-level metrics [5dba8a6]
- [x] Map vSphere host data to host analysis models [5dba8a6]

- [x] Task: Conductor - User Manual Verification 'Phase 2: Backend - Host-Level Analysis' (Protocol in workflow.md) [b8aabf4]

---

## Phase 3: Backend - Multi-Resource Bottleneck and Recommendations

### Task 3.1: Implement resource exhaustion ordering
- [x] Write tests for ranking resources by utilization percentage [470d9de]
- [x] Write tests for identifying the constraining resource [470d9de]
- [x] Implement resource exhaustion ordering logic [470d9de]

### Task 3.2: Implement upgrade path recommendations
- [x] Write tests for "add cells" recommendation logic [26b393f]
- [x] Write tests for "resize cells" recommendation logic [26b393f]
- [x] Write tests for "add hosts" recommendation logic [26b393f]
- [x] Implement recommendation engine with priority ordering [26b393f]

### Task 3.3: Add recommendations to API
- [x] Write tests for recommendations in API response [a749d78]
- [x] Update scenario comparison API to include recommendations [a749d78]
- [x] Add bottleneck summary to infrastructure status endpoint [a749d78]

- [~] Task: Conductor - User Manual Verification 'Phase 3: Backend - Multi-Resource Bottleneck and Recommendations' (Protocol in workflow.md)

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
