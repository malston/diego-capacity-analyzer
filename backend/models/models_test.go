package models

import (
	"encoding/json"
	"testing"
)

func TestDiegoCell_JSON(t *testing.T) {
	cell := DiegoCell{
		ID:               "cell-01",
		Name:             "diego_cell/0",
		MemoryMB:         16384,
		AllocatedMB:      12288,
		UsedMB:           9830,
		CPUPercent:       45,
		IsolationSegment: "default",
	}

	data, err := json.Marshal(cell)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded DiegoCell
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != cell.ID {
		t.Errorf("Expected ID %s, got %s", cell.ID, decoded.ID)
	}
}

func TestApp_JSON(t *testing.T) {
	app := App{
		Name:             "test-app",
		Instances:        2,
		RequestedMB:      1024,
		ActualMB:         780,
		IsolationSegment: "production",
	}

	data, err := json.Marshal(app)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded App
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Name != app.Name {
		t.Errorf("Expected Name %s, got %s", app.Name, decoded.Name)
	}
}
