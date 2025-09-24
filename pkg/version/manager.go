package version

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Manager manages versioned components and their registration
type Manager struct {
	components       map[string]VersionedComponent
	componentMeta    map[string]*VersionedComponentMeta
	constraints      map[string][]VersionConstraint
	metricsCollector MetricsCollector
	detector         *Detector
	mu               sync.RWMutex
}

// ManagerOption allows customization of the version manager
type ManagerOption func(*Manager)

// WithDetector sets the version detector
func WithDetector(detector *Detector) ManagerOption {
	return func(m *Manager) {
		m.detector = detector
	}
}

// WithMetricsCollector sets the metrics collector
func WithMetricsCollector(collector MetricsCollector) ManagerOption {
	return func(m *Manager) {
		m.metricsCollector = collector
	}
}

// NewManager creates a new version manager
func NewManager(options ...ManagerOption) *Manager {
	m := &Manager{
		components:    make(map[string]VersionedComponent),
		componentMeta: make(map[string]*VersionedComponentMeta),
		constraints:   make(map[string][]VersionConstraint),
		detector:      NewDetector(),
	}

	for _, option := range options {
		option(m)
	}

	return m
}

// Register registers a versioned component
func (m *Manager) Register(component VersionedComponent) error {
	if component == nil {
		return fmt.Errorf("component cannot be nil")
	}

	name := component.Name()
	if name == "" {
		return fmt.Errorf("component name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for name conflicts
	if _, exists := m.components[name]; exists {
		return fmt.Errorf("component with name '%s' already registered", name)
	}

	// Register the component
	m.components[name] = component

	// Create metadata
	now := time.Now().UTC().Format(time.RFC3339)
	meta := &VersionedComponentMeta{
		Name:              name,
		Type:              component.Type(),
		SupportedVersions: component.SupportedVersions(),
		DefaultVersion:    component.GetDefaultVersion(),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Add deprecation info if component supports it
	if deprecatable, ok := component.(DeprecatableComponent); ok {
		for _, version := range meta.SupportedVersions {
			if deprecatable.IsVersionDeprecated(version) {
				if depInfo := deprecatable.GetDeprecationInfo(version); depInfo != nil {
					meta.DeprecatedVersions = append(meta.DeprecatedVersions, *depInfo)
				}
			}
		}
	}

	m.componentMeta[name] = meta
	return nil
}

// RegisterWithConstraints registers a component with version constraints
func (m *Manager) RegisterWithConstraints(component VersionedComponent, constraints []VersionConstraint) error {
	if err := m.Register(component); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	name := component.Name()
	m.constraints[name] = constraints
	m.componentMeta[name].Constraints = constraints

	return nil
}

// Unregister removes a component from the manager
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.components[name]; !exists {
		return fmt.Errorf("component '%s' not found", name)
	}

	delete(m.components, name)
	delete(m.componentMeta, name)
	delete(m.constraints, name)

	return nil
}

// Get retrieves a component by name and validates version support
func (m *Manager) Get(name string, version Version) (VersionedComponent, error) {
	m.mu.RLock()
	component, exists := m.components[name]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("component '%s' not found", name)
	}

	if !component.IsVersionSupported(version) {
		supported := component.SupportedVersions()
		supportedStrs := make([]string, len(supported))
		for i, v := range supported {
			supportedStrs[i] = v.String()
		}
		return nil, NewVersionError(name, version, supported,
			fmt.Sprintf("version not supported. Supported versions: %v", supportedStrs))
	}

	// Record metrics if collector is available
	if m.metricsCollector != nil {
		m.metricsCollector.RecordRequest(name, version)
	}

	return component, nil
}

// GetComponent retrieves a component by name without version validation
func (m *Manager) GetComponent(name string) (VersionedComponent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	component, exists := m.components[name]
	return component, exists
}

// List returns all registered component names
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.components))
	for name := range m.components {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// ListByType returns component names filtered by type
func (m *Manager) ListByType(componentType string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name, component := range m.components {
		if component.Type() == componentType {
			names = append(names, name)
		}
	}

	sort.Strings(names)
	return names
}

// GetMeta returns metadata for a component
func (m *Manager) GetMeta(name string) (*VersionedComponentMeta, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	meta, exists := m.componentMeta[name]
	return meta, exists
}

// GetAllMeta returns metadata for all components
func (m *Manager) GetAllMeta() map[string]*VersionedComponentMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*VersionedComponentMeta, len(m.componentMeta))
	for name, meta := range m.componentMeta {
		// Create a copy to avoid concurrent access issues
		metaCopy := *meta
		result[name] = &metaCopy
	}

	return result
}

// GetSupportedVersions returns all supported versions for a component
func (m *Manager) GetSupportedVersions(name string) ([]Version, error) {
	component, exists := m.GetComponent(name)
	if !exists {
		return nil, fmt.Errorf("component '%s' not found", name)
	}

	return component.SupportedVersions(), nil
}

// GetAllSupportedVersions returns supported versions for all components
func (m *Manager) GetAllSupportedVersions() map[string][]Version {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]Version, len(m.components))
	for name, component := range m.components {
		result[name] = component.SupportedVersions()
	}

	return result
}

// ProcessVersioned processes a request using the appropriate component version
func (m *Manager) ProcessVersioned(req *VersionedRequest, input interface{}) (*VersionedResponse, error) {
	component, err := m.Get(req.Component, req.Version)
	if err != nil {
		if m.metricsCollector != nil {
			m.metricsCollector.RecordError(req.Component, req.Version, err)
		}
		return nil, err
	}

	response, err := component.ProcessVersioned(req, input)
	if err != nil && m.metricsCollector != nil {
		m.metricsCollector.RecordError(req.Component, req.Version, err)
	}

	return response, err
}

// ProcessWithAutoVersion processes a request with automatic version detection
func (m *Manager) ProcessWithAutoVersion(ctx context.Context, componentName string, input interface{}) (*VersionedResponse, error) {
	version := m.detector.DetectFromContext(ctx).Version

	req := NewVersionedRequest(ctx, version, componentName)
	return m.ProcessVersioned(req, input)
}

// CheckCompatibility checks if the given component versions are compatible
func (m *Manager) CheckCompatibility(componentVersions map[string]Version) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check each component's constraints
	for componentName, version := range componentVersions {
		constraints, exists := m.constraints[componentName]
		if !exists {
			continue
		}

		for _, constraint := range constraints {
			// Check if required component version is satisfied
			requiredComponent := constraint.Component
			if requiredVersion, ok := componentVersions[requiredComponent]; ok {
				if !constraint.Requires.Contains(requiredVersion) {
					return fmt.Errorf("component '%s' version %s requires '%s' version in range %s, but got %s",
						componentName, version.String(), requiredComponent,
						constraint.Requires.String(), requiredVersion.String())
				}

				// Check for conflicts
				for _, conflictVersion := range constraint.Conflicts {
					if requiredVersion.Major == conflictVersion.Major &&
						requiredVersion.Minor == conflictVersion.Minor &&
						requiredVersion.Patch == conflictVersion.Patch {
						return fmt.Errorf("component '%s' version %s conflicts with '%s' version %s",
							componentName, version.String(), requiredComponent, conflictVersion.String())
					}
				}
			}
		}
	}

	return nil
}

// GetDetector returns the version detector
func (m *Manager) GetDetector() *Detector {
	return m.detector
}

// GetMetricsCollector returns the metrics collector
func (m *Manager) GetMetricsCollector() MetricsCollector {
	return m.metricsCollector
}

// UpdateComponentMeta updates metadata for a component
func (m *Manager) UpdateComponentMeta(name string, updateFn func(*VersionedComponentMeta)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	meta, exists := m.componentMeta[name]
	if !exists {
		return fmt.Errorf("component '%s' not found", name)
	}

	updateFn(meta)
	meta.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	return nil
}

// GetCompatibleVersions returns versions of a component that are compatible with the given version
func (m *Manager) GetCompatibleVersions(componentName string, targetVersion Version) ([]Version, error) {
	component, exists := m.GetComponent(componentName)
	if !exists {
		return nil, fmt.Errorf("component '%s' not found", componentName)
	}

	supported := component.SupportedVersions()
	var compatible []Version

	for _, version := range supported {
		if version.IsCompatible(targetVersion) {
			compatible = append(compatible, version)
		}
	}

	return compatible, nil
}

// GetLatestVersion returns the latest version for a component
func (m *Manager) GetLatestVersion(componentName string) (Version, error) {
	component, exists := m.GetComponent(componentName)
	if !exists {
		return Version{}, fmt.Errorf("component '%s' not found", componentName)
	}

	versions := component.SupportedVersions()
	if len(versions) == 0 {
		return Version{}, fmt.Errorf("component '%s' has no supported versions", componentName)
	}

	// Sort versions and return the latest
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Compare(versions[j]) > 0
	})

	return versions[0], nil
}

// GlobalManager is the default global version manager
var GlobalManager = NewManager()

// RegisterGlobal registers a component in the global manager
func RegisterGlobal(component VersionedComponent) error {
	return GlobalManager.Register(component)
}

// GetGlobal retrieves a component from the global manager
func GetGlobal(name string, version Version) (VersionedComponent, error) {
	return GlobalManager.Get(name, version)
}
