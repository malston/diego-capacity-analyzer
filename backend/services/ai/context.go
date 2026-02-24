// ABOUTME: Serializes in-memory capacity data into annotated markdown for the LLM
// ABOUTME: Pure function accepting only model types -- no config, clients, or services

package ai

import (
	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

// ContextInput bundles all data sources the context builder can serialize.
// Nil pointers indicate absent/unconfigured data sources.
type ContextInput struct {
	Dashboard *models.DashboardResponse
	Infra     *models.InfrastructureState
	Scenario  *models.ScenarioComparison

	// Data source availability flags (distinct from nil data --
	// a source can be configured but have no data yet)
	BOSHConfigured    bool
	VSphereConfigured bool
	LogCacheAvailable bool
}

// BuildContext serializes capacity data into annotated markdown for the LLM.
func BuildContext(input ContextInput) string {
	return ""
}
