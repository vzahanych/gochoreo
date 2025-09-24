package version

import (
	"fmt"
	"strings"
)

// VersionError represents version-related errors
type VersionError struct {
	Component         string    `json:"component"`
	RequestedVersion  Version   `json:"requested_version"`
	SupportedVersions []Version `json:"supported_versions"`
	Message           string    `json:"message"`
	Code              string    `json:"code"`
}

// Error implements the error interface
func (e *VersionError) Error() string {
	supportedStrs := make([]string, len(e.SupportedVersions))
	for i, v := range e.SupportedVersions {
		supportedStrs[i] = v.String()
	}

	return fmt.Sprintf("version error in component '%s': %s (requested: %s, supported: [%s])",
		e.Component, e.Message, e.RequestedVersion.String(), strings.Join(supportedStrs, ", "))
}

// NewVersionError creates a new version error
func NewVersionError(component string, requested Version, supported []Version, message string) *VersionError {
	return &VersionError{
		Component:         component,
		RequestedVersion:  requested,
		SupportedVersions: supported,
		Message:           message,
		Code:              "VERSION_NOT_SUPPORTED",
	}
}

// NewVersionErrorWithCode creates a new version error with a specific error code
func NewVersionErrorWithCode(component string, requested Version, supported []Version, message, code string) *VersionError {
	return &VersionError{
		Component:         component,
		RequestedVersion:  requested,
		SupportedVersions: supported,
		Message:           message,
		Code:              code,
	}
}

// ComponentNotFoundError represents errors when a component is not found
type ComponentNotFoundError struct {
	Component string `json:"component"`
	Message   string `json:"message"`
}

// Error implements the error interface
func (e *ComponentNotFoundError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("component not found: %s - %s", e.Component, e.Message)
	}
	return fmt.Sprintf("component not found: %s", e.Component)
}

// NewComponentNotFoundError creates a new component not found error
func NewComponentNotFoundError(component string, message string) *ComponentNotFoundError {
	return &ComponentNotFoundError{
		Component: component,
		Message:   message,
	}
}

// MigrationError represents errors during version migration
type MigrationError struct {
	Component   string  `json:"component"`
	FromVersion Version `json:"from_version"`
	ToVersion   Version `json:"to_version"`
	Message     string  `json:"message"`
	Cause       error   `json:"-"`
}

// Error implements the error interface
func (e *MigrationError) Error() string {
	baseMsg := fmt.Sprintf("migration error in component '%s' from %s to %s: %s",
		e.Component, e.FromVersion.String(), e.ToVersion.String(), e.Message)

	if e.Cause != nil {
		return fmt.Sprintf("%s (cause: %s)", baseMsg, e.Cause.Error())
	}

	return baseMsg
}

// Unwrap returns the underlying cause
func (e *MigrationError) Unwrap() error {
	return e.Cause
}

// NewMigrationError creates a new migration error
func NewMigrationError(component string, from, to Version, message string, cause error) *MigrationError {
	return &MigrationError{
		Component:   component,
		FromVersion: from,
		ToVersion:   to,
		Message:     message,
		Cause:       cause,
	}
}

// CompatibilityError represents errors in version compatibility
type CompatibilityError struct {
	Component     string            `json:"component"`
	Version       Version           `json:"version"`
	Constraints   map[string]string `json:"constraints"`
	ConflictsWith map[string]string `json:"conflicts_with"`
	Message       string            `json:"message"`
}

// Error implements the error interface
func (e *CompatibilityError) Error() string {
	var details []string

	if len(e.Constraints) > 0 {
		constraintStrs := make([]string, 0, len(e.Constraints))
		for comp, constraint := range e.Constraints {
			constraintStrs = append(constraintStrs, fmt.Sprintf("%s: %s", comp, constraint))
		}
		details = append(details, fmt.Sprintf("requires: %s", strings.Join(constraintStrs, ", ")))
	}

	if len(e.ConflictsWith) > 0 {
		conflictStrs := make([]string, 0, len(e.ConflictsWith))
		for comp, version := range e.ConflictsWith {
			conflictStrs = append(conflictStrs, fmt.Sprintf("%s: %s", comp, version))
		}
		details = append(details, fmt.Sprintf("conflicts with: %s", strings.Join(conflictStrs, ", ")))
	}

	baseMsg := fmt.Sprintf("compatibility error for component '%s' version %s",
		e.Component, e.Version.String())

	if e.Message != "" {
		baseMsg = fmt.Sprintf("%s: %s", baseMsg, e.Message)
	}

	if len(details) > 0 {
		return fmt.Sprintf("%s (%s)", baseMsg, strings.Join(details, "; "))
	}

	return baseMsg
}

// NewCompatibilityError creates a new compatibility error
func NewCompatibilityError(component string, version Version, message string) *CompatibilityError {
	return &CompatibilityError{
		Component:     component,
		Version:       version,
		Message:       message,
		Constraints:   make(map[string]string),
		ConflictsWith: make(map[string]string),
	}
}

// WithConstraint adds a constraint requirement to the error
func (e *CompatibilityError) WithConstraint(component, constraint string) *CompatibilityError {
	if e.Constraints == nil {
		e.Constraints = make(map[string]string)
	}
	e.Constraints[component] = constraint
	return e
}

// WithConflict adds a conflict to the error
func (e *CompatibilityError) WithConflict(component, version string) *CompatibilityError {
	if e.ConflictsWith == nil {
		e.ConflictsWith = make(map[string]string)
	}
	e.ConflictsWith[component] = version
	return e
}

// DeprecationError represents errors related to deprecated versions
type DeprecationError struct {
	Component   string           `json:"component"`
	Version     Version          `json:"version"`
	Information *DeprecationInfo `json:"deprecation_info,omitempty"`
	Message     string           `json:"message"`
}

// Error implements the error interface
func (e *DeprecationError) Error() string {
	baseMsg := fmt.Sprintf("deprecated version warning for component '%s' version %s",
		e.Component, e.Version.String())

	if e.Message != "" {
		baseMsg = fmt.Sprintf("%s: %s", baseMsg, e.Message)
	}

	if e.Information != nil {
		details := []string{}
		if e.Information.Reason != "" {
			details = append(details, fmt.Sprintf("reason: %s", e.Information.Reason))
		}
		if e.Information.SunsetAt != "" {
			details = append(details, fmt.Sprintf("sunset: %s", e.Information.SunsetAt))
		}
		if !e.Information.Replacement.IsZero() {
			details = append(details, fmt.Sprintf("replacement: %s", e.Information.Replacement.String()))
		}

		if len(details) > 0 {
			return fmt.Sprintf("%s (%s)", baseMsg, strings.Join(details, "; "))
		}
	}

	return baseMsg
}

// NewDeprecationError creates a new deprecation error
func NewDeprecationError(component string, version Version, info *DeprecationInfo, message string) *DeprecationError {
	return &DeprecationError{
		Component:   component,
		Version:     version,
		Information: info,
		Message:     message,
	}
}

// ConfigurationError represents configuration-related version errors
type ConfigurationError struct {
	Component string  `json:"component"`
	Version   Version `json:"version"`
	Field     string  `json:"field,omitempty"`
	Message   string  `json:"message"`
	Cause     error   `json:"-"`
}

// Error implements the error interface
func (e *ConfigurationError) Error() string {
	baseMsg := fmt.Sprintf("configuration error for component '%s' version %s",
		e.Component, e.Version.String())

	if e.Field != "" {
		baseMsg = fmt.Sprintf("%s field '%s'", baseMsg, e.Field)
	}

	if e.Message != "" {
		baseMsg = fmt.Sprintf("%s: %s", baseMsg, e.Message)
	}

	if e.Cause != nil {
		return fmt.Sprintf("%s (cause: %s)", baseMsg, e.Cause.Error())
	}

	return baseMsg
}

// Unwrap returns the underlying cause
func (e *ConfigurationError) Unwrap() error {
	return e.Cause
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(component string, version Version, field, message string, cause error) *ConfigurationError {
	return &ConfigurationError{
		Component: component,
		Version:   version,
		Field:     field,
		Message:   message,
		Cause:     cause,
	}
}

// ValidationError represents validation errors for versioned inputs/outputs
type ValidationError struct {
	Component string      `json:"component"`
	Version   Version     `json:"version"`
	Field     string      `json:"field,omitempty"`
	Value     interface{} `json:"value,omitempty"`
	Message   string      `json:"message"`
	Errors    []string    `json:"errors,omitempty"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	baseMsg := fmt.Sprintf("validation error for component '%s' version %s",
		e.Component, e.Version.String())

	if e.Field != "" {
		baseMsg = fmt.Sprintf("%s field '%s'", baseMsg, e.Field)
	}

	if e.Message != "" {
		baseMsg = fmt.Sprintf("%s: %s", baseMsg, e.Message)
	}

	if len(e.Errors) > 0 {
		return fmt.Sprintf("%s (%s)", baseMsg, strings.Join(e.Errors, "; "))
	}

	return baseMsg
}

// NewValidationError creates a new validation error
func NewValidationError(component string, version Version, field, message string) *ValidationError {
	return &ValidationError{
		Component: component,
		Version:   version,
		Field:     field,
		Message:   message,
	}
}

// WithErrors adds validation errors to the error
func (e *ValidationError) WithErrors(errors []string) *ValidationError {
	e.Errors = errors
	return e
}

// WithValue sets the invalid value
func (e *ValidationError) WithValue(value interface{}) *ValidationError {
	e.Value = value
	return e
}

// IsVersionError checks if an error is a version-related error
func IsVersionError(err error) bool {
	switch err.(type) {
	case *VersionError, *ComponentNotFoundError, *MigrationError,
		*CompatibilityError, *DeprecationError, *ConfigurationError, *ValidationError:
		return true
	default:
		return false
	}
}

// GetVersionErrorCode returns the error code from a version error
func GetVersionErrorCode(err error) string {
	switch e := err.(type) {
	case *VersionError:
		return e.Code
	case *ComponentNotFoundError:
		return "COMPONENT_NOT_FOUND"
	case *MigrationError:
		return "MIGRATION_ERROR"
	case *CompatibilityError:
		return "COMPATIBILITY_ERROR"
	case *DeprecationError:
		return "DEPRECATED_VERSION"
	case *ConfigurationError:
		return "CONFIGURATION_ERROR"
	case *ValidationError:
		return "VALIDATION_ERROR"
	default:
		return "UNKNOWN_ERROR"
	}
}
