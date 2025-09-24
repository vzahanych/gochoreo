package vault

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/vault/api"
)

// AuthMethod represents the authentication method for Vault
type AuthMethod string

const (
	AuthMethodToken      AuthMethod = "token"
	AuthMethodAppRole    AuthMethod = "approle"
	AuthMethodAWS        AuthMethod = "aws"
	AuthMethodKubernetes AuthMethod = "kubernetes"
	AuthMethodJWT        AuthMethod = "jwt"
	AuthMethodOIDC       AuthMethod = "oidc"
	AuthMethodUserpass   AuthMethod = "userpass"
	AuthMethodLDAP       AuthMethod = "ldap"
	AuthMethodGCP        AuthMethod = "gcp"
	AuthMethodAzure      AuthMethod = "azure"
	AuthMethodRadius     AuthMethod = "radius"
	AuthMethodOkta       AuthMethod = "okta"
	AuthMethodGitHub     AuthMethod = "github"
	AuthMethodTLS        AuthMethod = "cert"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelTrace LogLevel = "trace"
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// RetryPolicy defines the retry behavior
type RetryPolicy string

const (
	RetryPolicyLinear      RetryPolicy = "linear"
	RetryPolicyExponential RetryPolicy = "exponential"
	RetryPolicyFixed       RetryPolicy = "fixed"
)

// Config holds all Vault client configuration options
type Config struct {
	// Connection settings
	Address            string        `json:"address" yaml:"address"`
	Timeout            time.Duration `json:"timeout" yaml:"timeout"`
	MaxIdleConnections int           `json:"max_idle_connections" yaml:"max_idle_connections"`
	MaxRetries         int           `json:"max_retries" yaml:"max_retries"`
	RetryWaitMin       time.Duration `json:"retry_wait_min" yaml:"retry_wait_min"`
	RetryWaitMax       time.Duration `json:"retry_wait_max" yaml:"retry_wait_max"`
	RetryPolicy        RetryPolicy   `json:"retry_policy" yaml:"retry_policy"`

	// TLS Configuration
	TLSConfig *TLSConfig `json:"tls_config" yaml:"tls_config"`

	// Authentication
	AuthMethod AuthMethod      `json:"auth_method" yaml:"auth_method"`
	Token      string          `json:"token" yaml:"token"`
	TokenFile  string          `json:"token_file" yaml:"token_file"`
	AppRole    *AppRoleAuth    `json:"app_role" yaml:"app_role"`
	AWS        *AWSAuth        `json:"aws" yaml:"aws"`
	Kubernetes *KubernetesAuth `json:"kubernetes" yaml:"kubernetes"`
	JWT        *JWTAuth        `json:"jwt" yaml:"jwt"`
	OIDC       *OIDCAuth       `json:"oidc" yaml:"oidc"`
	Userpass   *UserpassAuth   `json:"userpass" yaml:"userpass"`
	LDAP       *LDAPAuth       `json:"ldap" yaml:"ldap"`
	GCP        *GCPAuth        `json:"gcp" yaml:"gcp"`
	Azure      *AzureAuth      `json:"azure" yaml:"azure"`
	GitHub     *GitHubAuth     `json:"github" yaml:"github"`
	TLSAuth    *TLSAuth        `json:"tls_auth" yaml:"tls_auth"`

	// Namespace support (Vault Enterprise)
	Namespace string `json:"namespace" yaml:"namespace"`

	// Agent configuration
	AgentAddress string `json:"agent_address" yaml:"agent_address"`

	// Rate limiting
	RateLimit *RateLimit `json:"rate_limit" yaml:"rate_limit"`

	// Logging and monitoring
	LogLevel            LogLevel      `json:"log_level" yaml:"log_level"`
	EnableDebug         bool          `json:"enable_debug" yaml:"enable_debug"`
	EnableMetrics       bool          `json:"enable_metrics" yaml:"enable_metrics"`
	MetricsPrefix       string        `json:"metrics_prefix" yaml:"metrics_prefix"`
	EnableHealthCheck   bool          `json:"enable_health_check" yaml:"enable_health_check"`
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval"`

	// Token management
	TokenRenew         bool          `json:"token_renew" yaml:"token_renew"`
	TokenRenewInterval time.Duration `json:"token_renew_interval" yaml:"token_renew_interval"`
	TokenRenewBuffer   time.Duration `json:"token_renew_buffer" yaml:"token_renew_buffer"`

	// Request settings
	RequestTimeout  time.Duration `json:"request_timeout" yaml:"request_timeout"`
	DialTimeout     time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	KeepAlive       time.Duration `json:"keep_alive" yaml:"keep_alive"`
	IdleConnTimeout time.Duration `json:"idle_conn_timeout" yaml:"idle_conn_timeout"`

	// Proxy settings
	ProxyAddress string `json:"proxy_address" yaml:"proxy_address"`

	// Custom headers
	Headers map[string]string `json:"headers" yaml:"headers"`

	// Cache settings
	EnableCache bool          `json:"enable_cache" yaml:"enable_cache"`
	CacheTTL    time.Duration `json:"cache_ttl" yaml:"cache_ttl"`
	CacheSize   int           `json:"cache_size" yaml:"cache_size"`

	// Wrapping settings
	WrapTTL string `json:"wrap_ttl" yaml:"wrap_ttl"`

	// Output settings
	OutputFormat string `json:"output_format" yaml:"output_format"`
	OutputField  string `json:"output_field" yaml:"output_field"`

	// Custom CA bundle
	CACert     string `json:"ca_cert" yaml:"ca_cert"`
	CAPath     string `json:"ca_path" yaml:"ca_path"`
	ClientCert string `json:"client_cert" yaml:"client_cert"`
	ClientKey  string `json:"client_key" yaml:"client_key"`

	// SRV DNS discovery
	SRVLookup bool `json:"srv_lookup" yaml:"srv_lookup"`
}

// TLSConfig holds TLS/SSL configuration
type TLSConfig struct {
	Insecure      bool     `json:"insecure" yaml:"insecure"`
	TLSServerName string   `json:"tls_server_name" yaml:"tls_server_name"`
	CACert        string   `json:"ca_cert" yaml:"ca_cert"`
	CAPath        string   `json:"ca_path" yaml:"ca_path"`
	ClientCert    string   `json:"client_cert" yaml:"client_cert"`
	ClientKey     string   `json:"client_key" yaml:"client_key"`
	TLSSkipVerify bool     `json:"tls_skip_verify" yaml:"tls_skip_verify"`
	MinVersion    uint16   `json:"min_version" yaml:"min_version"`
	MaxVersion    uint16   `json:"max_version" yaml:"max_version"`
	CipherSuites  []uint16 `json:"cipher_suites" yaml:"cipher_suites"`
}

// AppRoleAuth holds AppRole authentication configuration
type AppRoleAuth struct {
	RoleID       string `json:"role_id" yaml:"role_id"`
	SecretID     string `json:"secret_id" yaml:"secret_id"`
	SecretIDFile string `json:"secret_id_file" yaml:"secret_id_file"`
	MountPath    string `json:"mount_path" yaml:"mount_path"`
	Unwrap       bool   `json:"unwrap" yaml:"unwrap"`
}

// AWSAuth holds AWS authentication configuration
type AWSAuth struct {
	Type            string `json:"type" yaml:"type"` // "iam" or "ec2"
	Role            string `json:"role" yaml:"role"`
	MountPath       string `json:"mount_path" yaml:"mount_path"`
	Region          string `json:"region" yaml:"region"`
	AccessKeyID     string `json:"access_key_id" yaml:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key" yaml:"secret_access_key"`
	SessionToken    string `json:"session_token" yaml:"session_token"`
	HeaderValue     string `json:"header_value" yaml:"header_value"`
	Nonce           string `json:"nonce" yaml:"nonce"`
}

// KubernetesAuth holds Kubernetes authentication configuration
type KubernetesAuth struct {
	Role                    string `json:"role" yaml:"role"`
	JWT                     string `json:"jwt" yaml:"jwt"`
	JWTPath                 string `json:"jwt_path" yaml:"jwt_path"`
	MountPath               string `json:"mount_path" yaml:"mount_path"`
	ServiceAccountTokenPath string `json:"service_account_token_path" yaml:"service_account_token_path"`
}

// JWTAuth holds JWT authentication configuration
type JWTAuth struct {
	Role      string `json:"role" yaml:"role"`
	JWT       string `json:"jwt" yaml:"jwt"`
	MountPath string `json:"mount_path" yaml:"mount_path"`
}

// OIDCAuth holds OIDC authentication configuration
type OIDCAuth struct {
	Role         string            `json:"role" yaml:"role"`
	MountPath    string            `json:"mount_path" yaml:"mount_path"`
	CallbackAddr string            `json:"callback_addr" yaml:"callback_addr"`
	CallbackPort int               `json:"callback_port" yaml:"callback_port"`
	Scopes       []string          `json:"scopes" yaml:"scopes"`
	Claims       map[string]string `json:"claims" yaml:"claims"`
}

// UserpassAuth holds username/password authentication configuration
type UserpassAuth struct {
	Username  string `json:"username" yaml:"username"`
	Password  string `json:"password" yaml:"password"`
	MountPath string `json:"mount_path" yaml:"mount_path"`
}

// LDAPAuth holds LDAP authentication configuration
type LDAPAuth struct {
	Username  string `json:"username" yaml:"username"`
	Password  string `json:"password" yaml:"password"`
	MountPath string `json:"mount_path" yaml:"mount_path"`
}

// GCPAuth holds GCP authentication configuration
type GCPAuth struct {
	Type           string `json:"type" yaml:"type"` // "gce" or "iam"
	Role           string `json:"role" yaml:"role"`
	MountPath      string `json:"mount_path" yaml:"mount_path"`
	ServiceAccount string `json:"service_account" yaml:"service_account"`
	Project        string `json:"project" yaml:"project"`
	JWT            string `json:"jwt" yaml:"jwt"`
	Credentials    string `json:"credentials" yaml:"credentials"`
}

// AzureAuth holds Azure authentication configuration
type AzureAuth struct {
	Role              string `json:"role" yaml:"role"`
	MountPath         string `json:"mount_path" yaml:"mount_path"`
	Resource          string `json:"resource" yaml:"resource"`
	ObjectID          string `json:"object_id" yaml:"object_id"`
	ClientID          string `json:"client_id" yaml:"client_id"`
	SubscriptionID    string `json:"subscription_id" yaml:"subscription_id"`
	ResourceGroupName string `json:"resource_group_name" yaml:"resource_group_name"`
	VMName            string `json:"vm_name" yaml:"vm_name"`
	VMSSName          string `json:"vmss_name" yaml:"vmss_name"`
}

// GitHubAuth holds GitHub authentication configuration
type GitHubAuth struct {
	Token     string `json:"token" yaml:"token"`
	MountPath string `json:"mount_path" yaml:"mount_path"`
}

// TLSAuth holds TLS certificate authentication configuration
type TLSAuth struct {
	MountPath string `json:"mount_path" yaml:"mount_path"`
	Name      string `json:"name" yaml:"name"`
}

// RateLimit holds rate limiting configuration
type RateLimit struct {
	Rate  float64 `json:"rate" yaml:"rate"`
	Burst int     `json:"burst" yaml:"burst"`
}

// DefaultConfig returns a default Vault configuration
func DefaultConfig() *Config {
	return &Config{
		// Connection settings
		Address:            "https://127.0.0.1:8200",
		Timeout:            60 * time.Second,
		MaxIdleConnections: 10,
		MaxRetries:         3,
		RetryWaitMin:       1 * time.Second,
		RetryWaitMax:       30 * time.Second,
		RetryPolicy:        RetryPolicyExponential,

		// TLS settings
		TLSConfig: &TLSConfig{
			Insecure:      false,
			TLSSkipVerify: false,
			MinVersion:    tls.VersionTLS12,
			MaxVersion:    tls.VersionTLS13,
		},

		// Authentication
		AuthMethod: AuthMethodToken,
		AppRole: &AppRoleAuth{
			MountPath: "approle",
		},
		AWS: &AWSAuth{
			Type:      "iam",
			MountPath: "aws",
		},
		Kubernetes: &KubernetesAuth{
			MountPath:               "kubernetes",
			ServiceAccountTokenPath: "/var/run/secrets/kubernetes.io/serviceaccount/token",
		},
		JWT: &JWTAuth{
			MountPath: "jwt",
		},
		OIDC: &OIDCAuth{
			MountPath:    "oidc",
			CallbackPort: 8250,
		},
		Userpass: &UserpassAuth{
			MountPath: "userpass",
		},
		LDAP: &LDAPAuth{
			MountPath: "ldap",
		},
		GCP: &GCPAuth{
			Type:      "gce",
			MountPath: "gcp",
		},
		Azure: &AzureAuth{
			MountPath: "azure",
			Resource:  "https://management.azure.com/",
		},
		GitHub: &GitHubAuth{
			MountPath: "github",
		},
		TLSAuth: &TLSAuth{
			MountPath: "cert",
		},

		// Rate limiting
		RateLimit: &RateLimit{
			Rate:  100.0,
			Burst: 200,
		},

		// Logging and monitoring
		LogLevel:            LogLevelWarn,
		EnableDebug:         false,
		EnableMetrics:       true,
		MetricsPrefix:       "vault",
		EnableHealthCheck:   true,
		HealthCheckInterval: 30 * time.Second,

		// Token management
		TokenRenew:         true,
		TokenRenewInterval: 15 * time.Minute,
		TokenRenewBuffer:   30 * time.Second,

		// Request settings
		RequestTimeout:  60 * time.Second,
		DialTimeout:     30 * time.Second,
		KeepAlive:       30 * time.Second,
		IdleConnTimeout: 90 * time.Second,

		// Cache settings
		EnableCache: false,
		CacheTTL:    5 * time.Minute,
		CacheSize:   1000,

		// Output settings
		OutputFormat: "json",

		// Headers
		Headers: make(map[string]string),

		// SRV DNS discovery
		SRVLookup: false,
	}
}

// DevelopmentConfig returns a development-friendly configuration
func DevelopmentConfig() *Config {
	config := DefaultConfig()
	config.Address = "http://127.0.0.1:8200"
	config.TLSConfig.Insecure = true
	config.TLSConfig.TLSSkipVerify = true
	config.LogLevel = LogLevelDebug
	config.EnableDebug = true
	config.TokenRenew = false // Don't auto-renew in development
	config.MaxRetries = 1
	config.RetryWaitMin = 500 * time.Millisecond
	config.RetryWaitMax = 2 * time.Second
	return config
}

// ProductionConfig returns a production-ready configuration
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.Address = "https://vault.example.com:8200"
	config.TLSConfig.Insecure = false
	config.TLSConfig.TLSSkipVerify = false
	config.TLSConfig.MinVersion = tls.VersionTLS13 // Enforce TLS 1.3
	config.LogLevel = LogLevelWarn
	config.EnableDebug = false
	config.TokenRenew = true
	config.TokenRenewInterval = 10 * time.Minute
	config.MaxRetries = 5
	config.RetryWaitMax = 2 * time.Minute
	config.EnableCache = true
	config.CacheTTL = 10 * time.Minute
	config.RateLimit.Rate = 50.0 // More conservative rate limiting
	config.RateLimit.Burst = 100
	return config
}

// TestConfig returns a configuration optimized for testing
func TestConfig() *Config {
	config := DefaultConfig()
	config.Address = "http://127.0.0.1:8200"
	config.TLSConfig.Insecure = true
	config.TLSConfig.TLSSkipVerify = true
	config.LogLevel = LogLevelError
	config.EnableDebug = false
	config.EnableHealthCheck = false
	config.TokenRenew = false
	config.MaxRetries = 1
	config.Timeout = 5 * time.Second
	config.RequestTimeout = 5 * time.Second
	return config
}

// ToAPIConfig converts the configuration to a Vault API config
func (c *Config) ToAPIConfig() (*api.Config, error) {
	config := api.DefaultConfig()

	// Set address
	if c.Address != "" {
		config.Address = c.Address
	}

	// Configure HTTP client settings
	if c.Timeout > 0 {
		config.Timeout = c.Timeout
	}

	// Ensure we have an HTTP transport
	if config.HttpClient.Transport == nil {
		config.HttpClient.Transport = &http.Transport{}
	}

	if c.MaxIdleConnections > 0 {
		if transport, ok := config.HttpClient.Transport.(*http.Transport); ok {
			transport.MaxIdleConns = c.MaxIdleConnections
		}
	}

	// Configure retries
	config.MaxRetries = c.MaxRetries
	// Note: Vault API client handles retry wait times internally

	// Configure TLS
	if c.TLSConfig != nil {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: c.TLSConfig.TLSSkipVerify,
			ServerName:         c.TLSConfig.TLSServerName,
			MinVersion:         c.TLSConfig.MinVersion,
			MaxVersion:         c.TLSConfig.MaxVersion,
		}

		if len(c.TLSConfig.CipherSuites) > 0 {
			tlsConfig.CipherSuites = c.TLSConfig.CipherSuites
		}

		// Load certificates if specified
		if c.TLSConfig.ClientCert != "" && c.TLSConfig.ClientKey != "" {
			cert, err := tls.LoadX509KeyPair(c.TLSConfig.ClientCert, c.TLSConfig.ClientKey)
			if err != nil {
				return nil, fmt.Errorf("failed to load client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		// Load CA certificate if specified
		if c.TLSConfig.CACert != "" || c.TLSConfig.CAPath != "" {
			caCertPool := x509.NewCertPool()

			if c.TLSConfig.CACert != "" {
				caCert, err := ioutil.ReadFile(c.TLSConfig.CACert)
				if err != nil {
					return nil, fmt.Errorf("failed to read CA certificate: %w", err)
				}
				caCertPool.AppendCertsFromPEM(caCert)
			}

			if c.TLSConfig.CAPath != "" {
				err := appendCertsFromDir(caCertPool, c.TLSConfig.CAPath)
				if err != nil {
					return nil, fmt.Errorf("failed to load CA certificates from path: %w", err)
				}
			}

			tlsConfig.RootCAs = caCertPool
		}

		if transport, ok := config.HttpClient.Transport.(*http.Transport); ok {
			transport.TLSClientConfig = tlsConfig
		}
	}

	// Set custom headers
	if len(c.Headers) > 0 {
		config.HttpClient.Transport = &headerRoundTripper{
			wrapped: config.HttpClient.Transport,
			headers: c.Headers,
		}
	}

	// Configure proxy
	if c.ProxyAddress != "" {
		proxyURL, err := url.Parse(c.ProxyAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy address: %w", err)
		}
		if transport, ok := config.HttpClient.Transport.(*http.Transport); ok {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// Configure additional timeouts
	if transport, ok := config.HttpClient.Transport.(*http.Transport); ok {
		if c.DialTimeout > 0 {
			transport.DialContext = (&net.Dialer{
				Timeout:   c.DialTimeout,
				KeepAlive: c.KeepAlive,
			}).DialContext
		}
		if c.IdleConnTimeout > 0 {
			transport.IdleConnTimeout = c.IdleConnTimeout
		}
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Address == "" {
		return ErrInvalidAddress
	}

	if c.Timeout <= 0 {
		return ErrInvalidTimeout
	}

	if c.MaxRetries < 0 {
		return ErrInvalidMaxRetries
	}

	// Validate auth method specific configuration
	switch c.AuthMethod {
	case AuthMethodToken:
		if c.Token == "" && c.TokenFile == "" {
			return ErrInvalidTokenAuth
		}
	case AuthMethodAppRole:
		if c.AppRole == nil || c.AppRole.RoleID == "" {
			return ErrInvalidAppRoleAuth
		}
	case AuthMethodKubernetes:
		if c.Kubernetes == nil || c.Kubernetes.Role == "" {
			return ErrInvalidKubernetesAuth
		}
	}

	return nil
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	clone := *c

	// Deep copy nested structs
	if c.TLSConfig != nil {
		tlsClone := *c.TLSConfig
		clone.TLSConfig = &tlsClone
	}

	if c.AppRole != nil {
		appRoleClone := *c.AppRole
		clone.AppRole = &appRoleClone
	}

	if c.AWS != nil {
		awsClone := *c.AWS
		clone.AWS = &awsClone
	}

	if c.Kubernetes != nil {
		k8sClone := *c.Kubernetes
		clone.Kubernetes = &k8sClone
	}

	if c.JWT != nil {
		jwtClone := *c.JWT
		clone.JWT = &jwtClone
	}

	if c.OIDC != nil {
		oidcClone := *c.OIDC
		clone.OIDC = &oidcClone
		if c.OIDC.Scopes != nil {
			clone.OIDC.Scopes = make([]string, len(c.OIDC.Scopes))
			copy(clone.OIDC.Scopes, c.OIDC.Scopes)
		}
		if c.OIDC.Claims != nil {
			clone.OIDC.Claims = make(map[string]string, len(c.OIDC.Claims))
			for k, v := range c.OIDC.Claims {
				clone.OIDC.Claims[k] = v
			}
		}
	}

	if c.RateLimit != nil {
		rateLimitClone := *c.RateLimit
		clone.RateLimit = &rateLimitClone
	}

	// Deep copy headers
	if c.Headers != nil {
		clone.Headers = make(map[string]string, len(c.Headers))
		for k, v := range c.Headers {
			clone.Headers[k] = v
		}
	}

	return &clone
}

// WithAuth configures authentication settings
func (c *Config) WithAuth(method AuthMethod) *Config {
	c.AuthMethod = method
	return c
}

// WithToken configures token authentication
func (c *Config) WithToken(token string) *Config {
	c.AuthMethod = AuthMethodToken
	c.Token = token
	return c
}

// WithAppRole configures AppRole authentication
func (c *Config) WithAppRole(roleID, secretID string) *Config {
	c.AuthMethod = AuthMethodAppRole
	c.AppRole.RoleID = roleID
	c.AppRole.SecretID = secretID
	return c
}

// WithTLS configures TLS settings
func (c *Config) WithTLS(certFile, keyFile, caFile string, skipVerify bool) *Config {
	c.TLSConfig.ClientCert = certFile
	c.TLSConfig.ClientKey = keyFile
	c.TLSConfig.CACert = caFile
	c.TLSConfig.TLSSkipVerify = skipVerify
	return c
}

// WithTimeout configures timeout settings
func (c *Config) WithTimeout(timeout time.Duration) *Config {
	c.Timeout = timeout
	c.RequestTimeout = timeout
	return c
}

// WithRetry configures retry settings
func (c *Config) WithRetry(maxRetries int, waitMin, waitMax time.Duration) *Config {
	c.MaxRetries = maxRetries
	c.RetryWaitMin = waitMin
	c.RetryWaitMax = waitMax
	return c
}

// WithNamespace configures Vault namespace (Enterprise feature)
func (c *Config) WithNamespace(namespace string) *Config {
	c.Namespace = namespace
	return c
}

// Helper types and functions

// headerRoundTripper wraps an http.RoundTripper to add custom headers
type headerRoundTripper struct {
	wrapped http.RoundTripper
	headers map[string]string
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range h.headers {
		req.Header.Set(key, value)
	}
	return h.wrapped.RoundTrip(req)
}
