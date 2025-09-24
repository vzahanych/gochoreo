package version

import (
	"fmt"
	"reflect"
)

// MigrationStrategy defines different strategies for version migration
type MigrationStrategy int

const (
	// MigrationStrategyNone - no migration, fail if versions don't match
	MigrationStrategyNone MigrationStrategy = iota
	// MigrationStrategyAutomatic - attempt automatic migration based on field mapping
	MigrationStrategyAutomatic
	// MigrationStrategyCustom - use custom migration functions
	MigrationStrategyCustom
	// MigrationStrategyFallback - fallback to default values for missing fields
	MigrationStrategyFallback
)

// String returns the string representation of the migration strategy
func (ms MigrationStrategy) String() string {
	switch ms {
	case MigrationStrategyNone:
		return "none"
	case MigrationStrategyAutomatic:
		return "automatic"
	case MigrationStrategyCustom:
		return "custom"
	case MigrationStrategyFallback:
		return "fallback"
	default:
		return "unknown"
	}
}

// MigrationFunc represents a function that can migrate data between versions
type MigrationFunc func(from, to Version, input interface{}) (interface{}, error)

// FieldMapping represents field mappings between versions
type FieldMapping struct {
	FromField    string      `json:"from_field"`
	ToField      string      `json:"to_field"`
	Transform    string      `json:"transform,omitempty"`     // transformation rule
	DefaultValue interface{} `json:"default_value,omitempty"` // default if field doesn't exist
	Required     bool        `json:"required"`                // whether the field is required
}

// VersionMigration contains migration information between two specific versions
type VersionMigration struct {
	FromVersion   Version                `json:"from_version"`
	ToVersion     Version                `json:"to_version"`
	Strategy      MigrationStrategy      `json:"strategy"`
	FieldMappings []FieldMapping         `json:"field_mappings,omitempty"`
	CustomFunc    MigrationFunc          `json:"-"` // custom migration function
	Reversible    bool                   `json:"reversible"`
	Description   string                 `json:"description,omitempty"`
	Examples      map[string]interface{} `json:"examples,omitempty"`
}

// Migrator handles version migrations for components
type Migrator struct {
	component  string
	migrations map[string]*VersionMigration // key: "fromVersion-toVersion"
}

// NewMigrator creates a new migrator for a component
func NewMigrator(component string) *Migrator {
	return &Migrator{
		component:  component,
		migrations: make(map[string]*VersionMigration),
	}
}

// AddMigration adds a migration between two versions
func (m *Migrator) AddMigration(migration *VersionMigration) error {
	if migration == nil {
		return fmt.Errorf("migration cannot be nil")
	}

	key := m.migrationKey(migration.FromVersion, migration.ToVersion)
	m.migrations[key] = migration

	// If reversible, add the reverse migration
	if migration.Reversible {
		reverseKey := m.migrationKey(migration.ToVersion, migration.FromVersion)
		reverseMigration := &VersionMigration{
			FromVersion:   migration.ToVersion,
			ToVersion:     migration.FromVersion,
			Strategy:      migration.Strategy,
			FieldMappings: m.reverseFieldMappings(migration.FieldMappings),
			Reversible:    false, // prevent infinite recursion
			Description:   fmt.Sprintf("Reverse migration: %s", migration.Description),
		}
		m.migrations[reverseKey] = reverseMigration
	}

	return nil
}

// AddCustomMigration adds a custom migration function between versions
func (m *Migrator) AddCustomMigration(from, to Version, migrationFunc MigrationFunc, reversible bool, description string) error {
	migration := &VersionMigration{
		FromVersion: from,
		ToVersion:   to,
		Strategy:    MigrationStrategyCustom,
		CustomFunc:  migrationFunc,
		Reversible:  reversible,
		Description: description,
	}

	return m.AddMigration(migration)
}

// AddAutomaticMigration adds an automatic migration with field mappings
func (m *Migrator) AddAutomaticMigration(from, to Version, fieldMappings []FieldMapping, reversible bool) error {
	migration := &VersionMigration{
		FromVersion:   from,
		ToVersion:     to,
		Strategy:      MigrationStrategyAutomatic,
		FieldMappings: fieldMappings,
		Reversible:    reversible,
		Description:   fmt.Sprintf("Automatic migration from %s to %s", from.String(), to.String()),
	}

	return m.AddMigration(migration)
}

// Migrate performs migration between two versions
func (m *Migrator) Migrate(from, to Version, input interface{}) (interface{}, error) {
	if from.Compare(to) == 0 {
		// Same version, no migration needed
		return input, nil
	}

	key := m.migrationKey(from, to)
	migration, exists := m.migrations[key]
	if !exists {
		return nil, NewMigrationError(m.component, from, to,
			"no migration path available", nil)
	}

	switch migration.Strategy {
	case MigrationStrategyNone:
		return nil, NewMigrationError(m.component, from, to,
			"migration not supported", nil)
	case MigrationStrategyCustom:
		if migration.CustomFunc == nil {
			return nil, NewMigrationError(m.component, from, to,
				"custom migration function not provided", nil)
		}
		return migration.CustomFunc(from, to, input)
	case MigrationStrategyAutomatic:
		return m.performAutomaticMigration(migration, input)
	case MigrationStrategyFallback:
		return m.performFallbackMigration(migration, input)
	default:
		return nil, NewMigrationError(m.component, from, to,
			fmt.Sprintf("unsupported migration strategy: %s", migration.Strategy.String()), nil)
	}
}

// CanMigrate returns true if migration is possible between two versions
func (m *Migrator) CanMigrate(from, to Version) bool {
	if from.Compare(to) == 0 {
		return true // same version
	}

	key := m.migrationKey(from, to)
	_, exists := m.migrations[key]
	return exists
}

// GetMigrationPath returns the migration path between two versions
func (m *Migrator) GetMigrationPath(from, to Version) ([]*VersionMigration, error) {
	if from.Compare(to) == 0 {
		return []*VersionMigration{}, nil
	}

	// For now, we only support direct migrations
	// TODO: Implement multi-step migration paths
	key := m.migrationKey(from, to)
	if migration, exists := m.migrations[key]; exists {
		return []*VersionMigration{migration}, nil
	}

	return nil, NewMigrationError(m.component, from, to,
		"no migration path found", nil)
}

// GetSupportedMigrations returns all supported migrations for this component
func (m *Migrator) GetSupportedMigrations() []*VersionMigration {
	migrations := make([]*VersionMigration, 0, len(m.migrations))
	for _, migration := range m.migrations {
		migrations = append(migrations, migration)
	}
	return migrations
}

// performAutomaticMigration performs automatic migration using field mappings
func (m *Migrator) performAutomaticMigration(migration *VersionMigration, input interface{}) (interface{}, error) {
	if input == nil {
		return nil, nil
	}

	// Use reflection to handle automatic migration
	inputVal := reflect.ValueOf(input)
	if inputVal.Kind() == reflect.Ptr {
		inputVal = inputVal.Elem()
	}

	if inputVal.Kind() != reflect.Struct && inputVal.Kind() != reflect.Map {
		// For non-struct/map types, try to convert directly
		return input, nil
	}

	var result interface{}
	var err error

	if inputVal.Kind() == reflect.Map {
		result, err = m.migrateMapData(migration, input)
	} else {
		result, err = m.migrateStructData(migration, input)
	}

	if err != nil {
		return nil, NewMigrationError(m.component, migration.FromVersion, migration.ToVersion,
			"automatic migration failed", err)
	}

	return result, nil
}

// migrateMapData migrates map-based data
func (m *Migrator) migrateMapData(migration *VersionMigration, input interface{}) (interface{}, error) {
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("input is not a map[string]interface{}")
	}

	resultMap := make(map[string]interface{})

	// Apply field mappings
	for _, mapping := range migration.FieldMappings {
		if value, exists := inputMap[mapping.FromField]; exists {
			// Transform the value if needed
			transformedValue, err := m.transformValue(value, mapping.Transform)
			if err != nil {
				return nil, fmt.Errorf("failed to transform field %s: %w", mapping.FromField, err)
			}
			resultMap[mapping.ToField] = transformedValue
		} else if mapping.DefaultValue != nil {
			// Use default value
			resultMap[mapping.ToField] = mapping.DefaultValue
		} else if mapping.Required {
			return nil, fmt.Errorf("required field %s is missing", mapping.FromField)
		}
	}

	// Copy unmapped fields (for additive migrations)
	for key, value := range inputMap {
		mappingExists := false
		for _, mapping := range migration.FieldMappings {
			if mapping.FromField == key {
				mappingExists = true
				break
			}
		}
		if !mappingExists {
			resultMap[key] = value
		}
	}

	return resultMap, nil
}

// migrateStructData migrates struct-based data
func (m *Migrator) migrateStructData(migration *VersionMigration, input interface{}) (interface{}, error) {
	// Convert struct to map for easier handling
	inputMap, err := m.structToMap(input)
	if err != nil {
		return nil, fmt.Errorf("failed to convert struct to map: %w", err)
	}

	// Migrate as map
	resultMap, err := m.migrateMapData(migration, inputMap)
	if err != nil {
		return nil, err
	}

	return resultMap, nil
}

// performFallbackMigration performs fallback migration with default values
func (m *Migrator) performFallbackMigration(migration *VersionMigration, input interface{}) (interface{}, error) {
	// Similar to automatic migration but more lenient with missing fields
	return m.performAutomaticMigration(migration, input)
}

// Helper methods

func (m *Migrator) migrationKey(from, to Version) string {
	return fmt.Sprintf("%s-%s", from.String(), to.String())
}

func (m *Migrator) reverseFieldMappings(mappings []FieldMapping) []FieldMapping {
	reversed := make([]FieldMapping, len(mappings))
	for i, mapping := range mappings {
		reversed[i] = FieldMapping{
			FromField:    mapping.ToField,
			ToField:      mapping.FromField,
			Transform:    m.reverseTransform(mapping.Transform),
			DefaultValue: nil, // Don't reverse default values
			Required:     mapping.Required,
		}
	}
	return reversed
}

func (m *Migrator) reverseTransform(transform string) string {
	// TODO: Implement reverse transformations
	// For now, return empty string (no transformation)
	return ""
}

func (m *Migrator) transformValue(value interface{}, transform string) (interface{}, error) {
	if transform == "" {
		return value, nil
	}

	// TODO: Implement various transformations
	// For now, just return the original value
	switch transform {
	case "string":
		return fmt.Sprintf("%v", value), nil
	case "int":
		if str, ok := value.(string); ok {
			// Try to parse string to int
			return str, nil // Placeholder
		}
		return value, nil
	default:
		return value, nil
	}
}

func (m *Migrator) structToMap(input interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	val := reflect.ValueOf(input)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input is not a struct")
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		if !fieldValue.CanInterface() {
			continue
		}

		// Use json tag if available, otherwise use field name
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			fieldName = jsonTag
		}

		result[fieldName] = fieldValue.Interface()
	}

	return result, nil
}

// GlobalMigrators stores migrators for different components
var GlobalMigrators = make(map[string]*Migrator)

// GetGlobalMigrator returns a migrator for the given component
func GetGlobalMigrator(component string) *Migrator {
	if migrator, exists := GlobalMigrators[component]; exists {
		return migrator
	}

	migrator := NewMigrator(component)
	GlobalMigrators[component] = migrator
	return migrator
}

// RegisterGlobalMigration registers a migration in the global migrators
func RegisterGlobalMigration(component string, migration *VersionMigration) error {
	migrator := GetGlobalMigrator(component)
	return migrator.AddMigration(migration)
}
