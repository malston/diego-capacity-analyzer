# Product Guide: TAS Capacity Analyzer

## Overview

TAS Capacity Analyzer is a full-stack dashboard for analyzing Tanzu Application Service (TAS) Diego cell capacity and density optimization. It provides platform teams with real-time visibility into cell utilization and actionable insights for infrastructure cost reduction.

## Target Users

**Platform Engineers and Operators** responsible for managing TAS/Cloud Foundry infrastructure. These teams need to understand current capacity utilization, plan for workload changes, and optimize infrastructure costs without impacting application performance.

## Problem Statement

Managing Diego cell capacity in production TAS environments requires gathering data from multiple sources (BOSH Director, CF API, vSphere, Log Cache) and performing manual calculations to understand utilization patterns. This fragmented approach makes it difficult to:

- Identify underutilized cells and optimization opportunities
- Predict the impact of configuration changes before implementing them
- Catch capacity issues before they affect running applications

## Core Value Proposition

### Visual Dashboard with Real-Time Metrics
Aggregates capacity data from BOSH, Cloud Foundry, vSphere, and Log Cache into a unified dashboard. Platform engineers see cell memory, CPU, and application density at a glance rather than querying multiple APIs.

### What-If Scenario Modeling
Simulate configuration changes—such as memory overcommit ratios or cell sizing—before applying them to production. The scenario wizard lets operators model proposed changes and understand their impact on capacity and application placement.

## Primary Goal

**Optimize Diego cell density** to maximize application placement per cell, reducing the number of cells (and underlying vSphere resources) required to run production workloads.

## Success Criteria

1. **Reduced Infrastructure Costs** - Lower vSphere/IaaS spend through improved cell utilization and right-sized infrastructure
2. **Proactive Capacity Management** - Identify utilization trends and capacity constraints before they impact running applications

## Target Environment

Production TAS foundations running real application workloads. The tool integrates with live BOSH Directors and vCenter instances to provide accurate, current capacity data.

## Key Features

- Real-time Diego cell capacity monitoring
- Isolation segment filtering and comparison
- Memory overcommit modeling
- Right-sizing recommendations for over-provisioned applications
- Scenario analysis wizard with step-based configuration
- vSphere infrastructure discovery
- Multiple data source support (live vSphere, JSON upload, manual entry, samples)
