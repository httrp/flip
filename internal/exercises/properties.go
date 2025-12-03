package exercises

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// PropertyDefinition represents a tracking property with its unit
type PropertyDefinition struct {
	Name string `json:"name"`
	Unit string `json:"unit"`
}

// PropertyManager manages known tracking properties
type PropertyManager struct {
	configPath string
	properties []PropertyDefinition
}

// NewPropertyManager creates a new property manager
func NewPropertyManager(brainPath string) (*PropertyManager, error) {
	configPath := filepath.Join(brainPath, ".flip", "exercise-properties.json")
	
	pm := &PropertyManager{
		configPath: configPath,
		properties: getDefaultProperties(),
	}
	
	// Load custom properties if they exist
	if err := pm.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	
	return pm, nil
}

// getDefaultProperties returns standard tracking properties
func getDefaultProperties() []PropertyDefinition {
	return []PropertyDefinition{
		{Name: "duration", Unit: "min"},
		{Name: "repetitions", Unit: "count"},
		{Name: "sets", Unit: "count"},
		{Name: "tempo", Unit: "bpm"},
		{Name: "weight", Unit: "kg"},
		{Name: "distance", Unit: "m"},
		{Name: "speed", Unit: "km/h"},
		{Name: "level", Unit: "1-10"},
		{Name: "intensity", Unit: "1-10"},
		{Name: "focus", Unit: "text"},
		{Name: "notes", Unit: "text"},
	}
}

// Load reads custom properties from file
func (pm *PropertyManager) Load() error {
	data, err := os.ReadFile(pm.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No custom properties yet
		}
		return fmt.Errorf("failed to read properties file: %w", err)
	}
	
	var customProps []PropertyDefinition
	if err := json.Unmarshal(data, &customProps); err != nil {
		return fmt.Errorf("failed to parse properties file: %w", err)
	}
	
	// Merge with defaults (custom props override defaults with same name)
	merged := make(map[string]PropertyDefinition)
	for _, prop := range pm.properties {
		merged[prop.Name] = prop
	}
	for _, prop := range customProps {
		merged[prop.Name] = prop
	}
	
	pm.properties = make([]PropertyDefinition, 0, len(merged))
	for _, prop := range merged {
		pm.properties = append(pm.properties, prop)
	}
	
	// Sort by name
	sort.Slice(pm.properties, func(i, j int) bool {
		return pm.properties[i].Name < pm.properties[j].Name
	})
	
	return nil
}

// Save writes custom properties to file
func (pm *PropertyManager) Save() error {
	// Only save non-default properties
	defaultMap := make(map[string]bool)
	for _, prop := range getDefaultProperties() {
		defaultMap[prop.Name] = true
	}
	
	var customProps []PropertyDefinition
	for _, prop := range pm.properties {
		if !defaultMap[prop.Name] {
			customProps = append(customProps, prop)
		}
	}
	
	// Ensure .flip directory exists
	flipDir := filepath.Dir(pm.configPath)
	if err := os.MkdirAll(flipDir, 0755); err != nil {
		return fmt.Errorf("failed to create .flip directory: %w", err)
	}
	
	data, err := json.MarshalIndent(customProps, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal properties: %w", err)
	}
	
	if err := os.WriteFile(pm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write properties file: %w", err)
	}
	
	return nil
}

// GetAll returns all known properties
func (pm *PropertyManager) GetAll() []PropertyDefinition {
	return pm.properties
}

// Add adds a new property
func (pm *PropertyManager) Add(name, unit string) error {
	// Check if already exists
	for _, prop := range pm.properties {
		if prop.Name == name {
			// Update unit if different
			if prop.Unit != unit {
				prop.Unit = unit
			}
			return pm.Save()
		}
	}
	
	// Add new property
	pm.properties = append(pm.properties, PropertyDefinition{
		Name: name,
		Unit: unit,
	})
	
	// Re-sort
	sort.Slice(pm.properties, func(i, j int) bool {
		return pm.properties[i].Name < pm.properties[j].Name
	})
	
	return pm.Save()
}

// GetPropertyNames returns a list of property names for selection
func (pm *PropertyManager) GetPropertyNames() []string {
	names := make([]string, len(pm.properties))
	for i, prop := range pm.properties {
		names[i] = fmt.Sprintf("%s (%s)", prop.Name, prop.Unit)
	}
	return names
}

// GetByName returns a property by name
func (pm *PropertyManager) GetByName(name string) *PropertyDefinition {
	for _, prop := range pm.properties {
		if prop.Name == name {
			return &prop
		}
	}
	return nil
}
