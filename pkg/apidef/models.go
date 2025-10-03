package apidef

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// AuthenticationType represents the type of authentication required
type AuthenticationType string

const (
	AuthNone   AuthenticationType = "none"
	AuthAPIKey AuthenticationType = "api_key"
	AuthJWT    AuthenticationType = "jwt"
	AuthOAuth2 AuthenticationType = "oauth2"
	AuthBasic  AuthenticationType = "basic"
	AuthHMAC   AuthenticationType = "hmac"
	AuthCustom AuthenticationType = "custom"
	AuthMulti  AuthenticationType = "multi"
)

// ProtocolType represents the protocol used
type ProtocolType string

const (
	ProtocolHTTP  ProtocolType = "http"
	ProtocolHTTPS ProtocolType = "https"
	ProtocolTCP   ProtocolType = "tcp"
	ProtocolGRPC  ProtocolType = "grpc"
)

// LoadBalancingType represents load balancing strategies
type LoadBalancingType string

const (
	LoadBalanceRoundRobin LoadBalancingType = "round_robin"
	LoadBalanceWeighted   LoadBalancingType = "weighted"
	LoadBalanceIPHash     LoadBalancingType = "ip_hash"
	LoadBalanceLeastConn  LoadBalancingType = "least_conn"
	LoadBalanceRandom     LoadBalancingType = "random"
)

// APIStatus represents the current status of an API
type APIStatus string

const (
	StatusActive     APIStatus = "active"
	StatusInactive   APIStatus = "inactive"
	StatusDraft      APIStatus = "draft"
	StatusDeprecated APIStatus = "deprecated"
)

// APIDefinition represents a complete API configuration
type APIDefinition struct {
	// Basic Information
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	APIID       string    `json:"api_id" db:"api_id"` // Unique identifier
	OrgID       string    `json:"org_id" db:"org_id"`
	Description string    `json:"description" db:"description"`
	Status      APIStatus `json:"status" db:"status"`
	Version     string    `json:"version" db:"version"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	CreatedBy string    `json:"created_by" db:"created_by"`
	UpdatedBy string    `json:"updated_by" db:"updated_by"`

	// Network Configuration
	Proxy           ProxyConfig  `json:"proxy"`
	Protocol        ProtocolType `json:"protocol" db:"protocol"`
	Domain          string       `json:"domain" db:"domain"`
	ListenPath      string       `json:"listen_path" db:"listen_path"`
	StripListenPath bool         `json:"strip_listen_path" db:"strip_listen_path"`

	// Authentication & Security
	AuthConfig       AuthConfig       `json:"auth_config"`
	UseKeylessAccess bool             `json:"use_keyless_access" db:"use_keyless_access"`
	EnableCORS       bool             `json:"enable_cors" db:"enable_cors"`
	CORSConfig       *CORSConfig      `json:"cors_config,omitempty"`
	SecurityPolicies []SecurityPolicy `json:"security_policies,omitempty"`

	// Rate Limiting & Quotas
	GlobalRateLimit *RateLimit `json:"global_rate_limit,omitempty"`
	PerKeyRateLimit *RateLimit `json:"per_key_rate_limit,omitempty"`
	GlobalQuota     *Quota     `json:"global_quota,omitempty"`

	// Middleware & Transformations
	Middleware         MiddlewareConfig `json:"middleware"`
	RequestTransforms  []Transform      `json:"request_transforms,omitempty"`
	ResponseTransforms []Transform      `json:"response_transforms,omitempty"`

	// Caching
	CacheConfig *CacheConfig `json:"cache_config,omitempty"`

	// Monitoring & Analytics
	EnableAnalytics bool             `json:"enable_analytics" db:"enable_analytics"`
	AnalyticsConfig *AnalyticsConfig `json:"analytics_config,omitempty"`

	// Advanced Features
	Tags       []string   `json:"tags" db:"tags"`
	Categories []string   `json:"categories" db:"categories"`
	Internal   bool       `json:"internal" db:"internal"`
	ExpireDate *time.Time `json:"expire_date,omitempty" db:"expire_date"`

	// Versioning
	VersioningConfig *VersioningConfig `json:"versioning_config,omitempty"`

	// Custom Fields
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// ProxyConfig defines upstream proxy configuration
type ProxyConfig struct {
	TargetURL     string            `json:"target_url"`
	Targets       []UpstreamTarget  `json:"targets,omitempty"`
	LoadBalancing LoadBalancingType `json:"load_balancing"`
	HealthCheck   *HealthCheck      `json:"health_check,omitempty"`

	// Timeout Configuration
	ConnectTimeout  time.Duration `json:"connect_timeout"`
	ResponseTimeout time.Duration `json:"response_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`

	// Retry Configuration
	RetryAttempts int           `json:"retry_attempts"`
	RetryBackoff  time.Duration `json:"retry_backoff"`

	// TLS Configuration
	TLSConfig *TLSConfig `json:"tls_config,omitempty"`

	// Headers
	PreserveHostHeader bool              `json:"preserve_host_header"`
	Headers            map[string]string `json:"headers,omitempty"`
}

// UpstreamTarget represents a backend target
type UpstreamTarget struct {
	ID       string            `json:"id"`
	URL      string            `json:"url"`
	Weight   int               `json:"weight"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// HealthCheck configuration for upstream targets
type HealthCheck struct {
	Enabled            bool          `json:"enabled"`
	Path               string        `json:"path"`
	Method             string        `json:"method"`
	Interval           time.Duration `json:"interval"`
	Timeout            time.Duration `json:"timeout"`
	HealthyThreshold   int           `json:"healthy_threshold"`
	UnhealthyThreshold int           `json:"unhealthy_threshold"`
	ExpectedStatus     []int         `json:"expected_status,omitempty"`
	ExpectedBody       string        `json:"expected_body,omitempty"`
}

// TLSConfig for upstream connections
type TLSConfig struct {
	Enabled            bool   `json:"enabled"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	ServerName         string `json:"server_name,omitempty"`
	ClientCert         string `json:"client_cert,omitempty"`
	ClientKey          string `json:"client_key,omitempty"`
	CACert             string `json:"ca_cert,omitempty"`
	MinVersion         string `json:"min_version,omitempty"`
	MaxVersion         string `json:"max_version,omitempty"`
}

// AuthConfig defines authentication configuration
type AuthConfig struct {
	Type         AuthenticationType     `json:"type"`
	APIKeyConfig *APIKeyConfig          `json:"api_key_config,omitempty"`
	JWTConfig    *JWTConfig             `json:"jwt_config,omitempty"`
	OAuth2Config *OAuth2Config          `json:"oauth2_config,omitempty"`
	BasicConfig  *BasicAuthConfig       `json:"basic_config,omitempty"`
	HMACConfig   *HMACConfig            `json:"hmac_config,omitempty"`
	CustomConfig map[string]interface{} `json:"custom_config,omitempty"`
	MultiConfig  []AuthConfig           `json:"multi_config,omitempty"` // For multi-auth
}

// APIKeyConfig for API key authentication
type APIKeyConfig struct {
	Location      string        `json:"location"` // header, query, form
	ParamName     string        `json:"param_name"`
	HashKeys      bool          `json:"hash_keys"`
	EnableCaching bool          `json:"enable_caching"`
	CacheTTL      time.Duration `json:"cache_ttl"`
}

// JWTConfig for JWT authentication
type JWTConfig struct {
	SigningMethod   string            `json:"signing_method"`
	SigningKey      string            `json:"signing_key"`
	IdentityKey     string            `json:"identity_key"`
	ClaimsToHeaders map[string]string `json:"claims_to_headers,omitempty"`
	RequiredClaims  []string          `json:"required_claims,omitempty"`
	Issuer          string            `json:"issuer,omitempty"`
	Audience        []string          `json:"audience,omitempty"`
	SkipValidation  []string          `json:"skip_validation,omitempty"`
}

// OAuth2Config for OAuth2 authentication
type OAuth2Config struct {
	TokenURL       string        `json:"token_url"`
	IntrospectURL  string        `json:"introspect_url"`
	ClientID       string        `json:"client_id"`
	ClientSecret   string        `json:"client_secret"`
	Scopes         []string      `json:"scopes,omitempty"`
	RequiredScopes []string      `json:"required_scopes,omitempty"`
	TokenLocation  string        `json:"token_location"` // header, query, form
	CacheTTL       time.Duration `json:"cache_ttl"`
}

// BasicAuthConfig for basic authentication
type BasicAuthConfig struct {
	Realm           string `json:"realm"`
	HideCredentials bool   `json:"hide_credentials"`
}

// HMACConfig for HMAC authentication
type HMACConfig struct {
	Algorithm        string        `json:"algorithm"`
	Header           string        `json:"header"`
	SecretKey        string        `json:"secret_key"`
	AllowedClockSkew time.Duration `json:"allowed_clock_skew"`
	Headers          []string      `json:"headers,omitempty"`
}

// CORSConfig defines CORS settings
type CORSConfig struct {
	AllowedOrigins     []string `json:"allowed_origins"`
	AllowedMethods     []string `json:"allowed_methods"`
	AllowedHeaders     []string `json:"allowed_headers"`
	ExposedHeaders     []string `json:"exposed_headers,omitempty"`
	AllowCredentials   bool     `json:"allow_credentials"`
	MaxAge             int      `json:"max_age"`
	OptionsPassthrough bool     `json:"options_passthrough"`
}

// SecurityPolicy represents security policies applied to the API
type SecurityPolicy struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type"` // ip_whitelist, ip_blacklist, geo_restriction, etc.
	Config   map[string]interface{} `json:"config"`
	Enabled  bool                   `json:"enabled"`
	Priority int                    `json:"priority"`
}

// RateLimit defines rate limiting configuration
type RateLimit struct {
	Rate     int           `json:"rate"`              // requests per period
	Period   time.Duration `json:"period"`            // time period
	Burst    int           `json:"burst"`             // burst size
	Strategy string        `json:"strategy"`          // local, distributed
	Headers  []string      `json:"headers,omitempty"` // rate limit headers to return
	Per      string        `json:"per"`               // ip, key, user, etc.
}

// Quota defines quota configuration
type Quota struct {
	Max                int           `json:"max"`    // maximum requests
	Period             time.Duration `json:"period"` // quota period
	RenewOnPeriodStart bool          `json:"renew_on_period_start"`
	Per                string        `json:"per"` // ip, key, user, etc.
}

// MiddlewareConfig defines middleware chain configuration
type MiddlewareConfig struct {
	Pre      []MiddlewareSpec `json:"pre,omitempty"`       // Pre-authentication
	Auth     []MiddlewareSpec `json:"auth,omitempty"`      // Authentication
	PostAuth []MiddlewareSpec `json:"post_auth,omitempty"` // Post-authentication
	Post     []MiddlewareSpec `json:"post,omitempty"`      // Post-processing
	Response []MiddlewareSpec `json:"response,omitempty"`  // Response processing
}

// MiddlewareSpec defines a single middleware configuration
type MiddlewareSpec struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"` // built-in, plugin, script
	Enabled  bool                   `json:"enabled"`
	Priority int                    `json:"priority"`
	Config   map[string]interface{} `json:"config,omitempty"`

	// For plugin middleware
	PluginPath string `json:"plugin_path,omitempty"`
	FuncName   string `json:"func_name,omitempty"`

	// For script middleware
	Script     string `json:"script,omitempty"`
	ScriptType string `json:"script_type,omitempty"` // js, lua, python
}

// Transform defines request/response transformations
type Transform struct {
	Type      string                 `json:"type"`   // header, body, url, method
	Action    string                 `json:"action"` // set, add, remove, replace
	Target    string                 `json:"target"` // field/header name, path
	Value     string                 `json:"value,omitempty"`
	Template  string                 `json:"template,omitempty"`
	Condition *TransformCondition    `json:"condition,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
}

// TransformCondition defines when a transform should be applied
type TransformCondition struct {
	Field    string         `json:"field"`    // header, query, body
	Operator string         `json:"operator"` // equals, contains, regex, exists
	Value    string         `json:"value"`
	Regex    *regexp.Regexp `json:"-"` // Compiled regex, not serialized
}

// CacheConfig defines caching behavior
type CacheConfig struct {
	Enabled         bool          `json:"enabled"`
	TTL             time.Duration `json:"ttl"`
	VaryHeaders     []string      `json:"vary_headers,omitempty"`
	CacheKeys       []string      `json:"cache_keys,omitempty"`
	SkipMethods     []string      `json:"skip_methods,omitempty"`
	SkipPaths       []string      `json:"skip_paths,omitempty"`
	OnlyStatusCodes []int         `json:"only_status_codes,omitempty"`
	Storage         string        `json:"storage"` // redis, memory, distributed
}

// AnalyticsConfig defines analytics and monitoring settings
type AnalyticsConfig struct {
	SampleRate     float64           `json:"sample_rate"`
	EnableDetailed bool              `json:"enable_detailed"`
	ExcludeHeaders []string          `json:"exclude_headers,omitempty"`
	ExcludePaths   []string          `json:"exclude_paths,omitempty"`
	CustomFields   map[string]string `json:"custom_fields,omitempty"`
	ExportConfig   *ExportConfig     `json:"export_config,omitempty"`
}

// ExportConfig defines where analytics data should be exported
type ExportConfig struct {
	Type          string                 `json:"type"` // elasticsearch, influxdb, webhook
	Config        map[string]interface{} `json:"config"`
	Enabled       bool                   `json:"enabled"`
	BatchSize     int                    `json:"batch_size"`
	FlushInterval time.Duration          `json:"flush_interval"`
}

// VersioningConfig defines API versioning strategy
type VersioningConfig struct {
	Strategy       string                 `json:"strategy"` // header, query, path, subdomain
	Key            string                 `json:"key"`      // header/query parameter name
	DefaultVersion string                 `json:"default_version"`
	Versions       map[string]VersionSpec `json:"versions"`
}

// VersionSpec defines configuration for a specific version
type VersionSpec struct {
	Name            string                 `json:"name"`
	Deprecated      bool                   `json:"deprecated"`
	DeprecationDate *time.Time             `json:"deprecation_date,omitempty"`
	SunsetDate      *time.Time             `json:"sunset_date,omitempty"`
	Overrides       map[string]interface{} `json:"overrides,omitempty"`
}

// Validation methods

// Validate validates the API definition
func (a *APIDefinition) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("API name is required")
	}

	if a.APIID == "" {
		return fmt.Errorf("API ID is required")
	}

	if a.ListenPath == "" {
		return fmt.Errorf("listen path is required")
	}

	if a.Proxy.TargetURL == "" && len(a.Proxy.Targets) == 0 {
		return fmt.Errorf("either target_url or targets must be specified")
	}

	// Validate authentication config
	if err := a.AuthConfig.Validate(); err != nil {
		return fmt.Errorf("auth config validation failed: %w", err)
	}

	// Validate transforms
	for i, transform := range a.RequestTransforms {
		if err := transform.Validate(); err != nil {
			return fmt.Errorf("request transform %d validation failed: %w", i, err)
		}
	}

	for i, transform := range a.ResponseTransforms {
		if err := transform.Validate(); err != nil {
			return fmt.Errorf("response transform %d validation failed: %w", i, err)
		}
	}

	return nil
}

// Validate validates authentication configuration
func (a *AuthConfig) Validate() error {
	switch a.Type {
	case AuthNone:
		return nil
	case AuthAPIKey:
		if a.APIKeyConfig == nil {
			return fmt.Errorf("api_key_config is required for API key authentication")
		}
		return a.APIKeyConfig.Validate()
	case AuthJWT:
		if a.JWTConfig == nil {
			return fmt.Errorf("jwt_config is required for JWT authentication")
		}
		return a.JWTConfig.Validate()
	case AuthOAuth2:
		if a.OAuth2Config == nil {
			return fmt.Errorf("oauth2_config is required for OAuth2 authentication")
		}
		return a.OAuth2Config.Validate()
	case AuthBasic:
		if a.BasicConfig == nil {
			return fmt.Errorf("basic_config is required for Basic authentication")
		}
		return a.BasicConfig.Validate()
	case AuthHMAC:
		if a.HMACConfig == nil {
			return fmt.Errorf("hmac_config is required for HMAC authentication")
		}
		return a.HMACConfig.Validate()
	case AuthMulti:
		if len(a.MultiConfig) == 0 {
			return fmt.Errorf("multi_config is required for multi authentication")
		}
		for i, config := range a.MultiConfig {
			if err := config.Validate(); err != nil {
				return fmt.Errorf("multi auth config %d validation failed: %w", i, err)
			}
		}
	default:
		return fmt.Errorf("unsupported authentication type: %s", a.Type)
	}

	return nil
}

// Validate validates API key configuration
func (a *APIKeyConfig) Validate() error {
	if a.Location == "" {
		return fmt.Errorf("location is required")
	}
	if a.ParamName == "" {
		return fmt.Errorf("param_name is required")
	}
	return nil
}

// Validate validates JWT configuration
func (j *JWTConfig) Validate() error {
	if j.SigningMethod == "" {
		return fmt.Errorf("signing_method is required")
	}
	if j.SigningKey == "" {
		return fmt.Errorf("signing_key is required")
	}
	return nil
}

// Validate validates OAuth2 configuration
func (o *OAuth2Config) Validate() error {
	if o.IntrospectURL == "" && o.TokenURL == "" {
		return fmt.Errorf("either introspect_url or token_url must be specified")
	}
	return nil
}

// Validate validates Basic auth configuration
func (b *BasicAuthConfig) Validate() error {
	// Basic auth config is always valid
	return nil
}

// Validate validates HMAC configuration
func (h *HMACConfig) Validate() error {
	if h.Algorithm == "" {
		return fmt.Errorf("algorithm is required")
	}
	if h.SecretKey == "" {
		return fmt.Errorf("secret_key is required")
	}
	return nil
}

// Validate validates transform configuration
func (t *Transform) Validate() error {
	if t.Type == "" {
		return fmt.Errorf("transform type is required")
	}
	if t.Action == "" {
		return fmt.Errorf("transform action is required")
	}
	if t.Target == "" {
		return fmt.Errorf("transform target is required")
	}

	// Compile regex if condition uses regex
	if t.Condition != nil && t.Condition.Operator == "regex" {
		regex, err := regexp.Compile(t.Condition.Value)
		if err != nil {
			return fmt.Errorf("invalid regex in transform condition: %w", err)
		}
		t.Condition.Regex = regex
	}

	return nil
}

// Helper methods

// IsActive returns true if the API is active
func (a *APIDefinition) IsActive() bool {
	return a.Status == StatusActive
}

// IsExpired returns true if the API is expired
func (a *APIDefinition) IsExpired() bool {
	if a.ExpireDate == nil {
		return false
	}
	return time.Now().After(*a.ExpireDate)
}

// GetAuthType returns the primary authentication type
func (a *APIDefinition) GetAuthType() AuthenticationType {
	return a.AuthConfig.Type
}

// HasRateLimit returns true if rate limiting is enabled
func (a *APIDefinition) HasRateLimit() bool {
	return a.GlobalRateLimit != nil || a.PerKeyRateLimit != nil
}

// HasQuota returns true if quota is enabled
func (a *APIDefinition) HasQuota() bool {
	return a.GlobalQuota != nil
}

// GetListenPath returns the cleaned listen path
func (a *APIDefinition) GetListenPath() string {
	path := a.ListenPath
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

// ToJSON converts the API definition to JSON
func (a *APIDefinition) ToJSON() ([]byte, error) {
	return json.Marshal(a)
}

// FromJSON loads API definition from JSON
func (a *APIDefinition) FromJSON(data []byte) error {
	return json.Unmarshal(data, a)
}

// Clone creates a deep copy of the API definition
func (a *APIDefinition) Clone() *APIDefinition {
	data, _ := json.Marshal(a)
	var clone APIDefinition
	json.Unmarshal(data, &clone)
	return &clone
}

// MatchesRequest checks if this API definition matches the given request
func (a *APIDefinition) MatchesRequest(r *http.Request) bool {
	// Check if the request path matches the listen path
	listenPath := a.GetListenPath()
	requestPath := r.URL.Path

	// Exact match or prefix match
	if listenPath == "/" {
		return true // Catch-all
	}

	if requestPath == listenPath {
		return true
	}

	if strings.HasPrefix(requestPath, listenPath+"/") {
		return true
	}

	// Check domain if specified
	if a.Domain != "" && r.Host != a.Domain {
		return false
	}

	return false
}

// GetEffectiveTargetURL returns the target URL to use for this request
func (a *APIDefinition) GetEffectiveTargetURL() string {
	if a.Proxy.TargetURL != "" {
		return a.Proxy.TargetURL
	}

	// For multiple targets, this would implement load balancing logic
	if len(a.Proxy.Targets) > 0 {
		// Simple round-robin for now
		return a.Proxy.Targets[0].URL
	}

	return ""
}
