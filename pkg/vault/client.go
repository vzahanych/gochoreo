package vault

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
	"golang.org/x/time/rate"
)

// SecretData represents secret data structure
type SecretData struct {
	Data     map[string]interface{} `json:"data"`
	Metadata *SecretMetadata        `json:"metadata,omitempty"`
}

// SecretMetadata represents secret metadata
type SecretMetadata struct {
	CreatedTime    time.Time              `json:"created_time"`
	DeletionTime   string                 `json:"deletion_time"`
	Destroyed      bool                   `json:"destroyed"`
	Version        int                    `json:"version"`
	CustomMetadata map[string]interface{} `json:"custom_metadata"`
}

// AuthInfo holds authentication information
type AuthInfo struct {
	ClientToken   string            `json:"client_token"`
	Accessor      string            `json:"accessor"`
	Policies      []string          `json:"policies"`
	TokenPolicies []string          `json:"token_policies"`
	Metadata      map[string]string `json:"metadata"`
	LeaseDuration int               `json:"lease_duration"`
	Renewable     bool              `json:"renewable"`
}

// Client represents the main Vault client
type Client struct {
	config      *Config
	client      *api.Client
	rateLimiter *rate.Limiter

	// Authentication state
	authInfo  *AuthInfo
	authMutex sync.RWMutex

	// Token management
	tokenRenewer *api.Renewer
	renewStop    chan struct{}

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	closed bool

	// Health monitoring
	healthTicker *time.Ticker
	healthStop   chan struct{}

	// Cache
	cache      map[string]cacheEntry
	cacheMutex sync.RWMutex
}

// cacheEntry represents a cached secret
type cacheEntry struct {
	data      *SecretData
	expiresAt time.Time
}

// Metrics holds client metrics
type Metrics struct {
	RequestsTotal       int64
	RequestErrors       int64
	AuthAttempts        int64
	AuthFailures        int64
	TokenRenewals       int64
	CacheHits           int64
	CacheMisses         int64
	AverageResponseTime time.Duration
	mu                  sync.RWMutex
}

// New creates a new Vault client with the given configuration
func New(ctx context.Context, config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create Vault API config
	apiConfig, err := config.ToAPIConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create API config: %w", err)
	}

	// Create Vault API client
	apiClient, err := api.NewClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	// Set namespace if configured (Vault Enterprise)
	if config.Namespace != "" {
		apiClient.SetNamespace(config.Namespace)
	}

	clientCtx, cancel := context.WithCancel(ctx)

	client := &Client{
		config:     config,
		client:     apiClient,
		ctx:        clientCtx,
		cancel:     cancel,
		renewStop:  make(chan struct{}),
		healthStop: make(chan struct{}),
		cache:      make(map[string]cacheEntry),
	}

	// Set up rate limiting if configured
	if config.RateLimit != nil {
		client.rateLimiter = rate.NewLimiter(rate.Limit(config.RateLimit.Rate), config.RateLimit.Burst)
	}

	// Authenticate
	if err := client.authenticate(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Start token renewal if enabled
	if config.TokenRenew && client.authInfo != nil && client.authInfo.Renewable {
		if err := client.startTokenRenewal(); err != nil {
			return nil, fmt.Errorf("failed to start token renewal: %w", err)
		}
	}

	// Start health monitoring if enabled
	if config.EnableHealthCheck {
		client.startHealthMonitoring()
	}

	return client, nil
}

// authenticate performs authentication based on the configured method
func (c *Client) authenticate() error {
	switch c.config.AuthMethod {
	case AuthMethodToken:
		return c.authenticateWithToken()
	case AuthMethodAppRole:
		return c.authenticateWithAppRole()
	case AuthMethodAWS:
		return c.authenticateWithAWS()
	case AuthMethodKubernetes:
		return c.authenticateWithKubernetes()
	case AuthMethodJWT:
		return c.authenticateWithJWT()
	case AuthMethodOIDC:
		return c.authenticateWithOIDC()
	case AuthMethodUserpass:
		return c.authenticateWithUserpass()
	case AuthMethodLDAP:
		return c.authenticateWithLDAP()
	case AuthMethodGCP:
		return c.authenticateWithGCP()
	case AuthMethodAzure:
		return c.authenticateWithAzure()
	case AuthMethodGitHub:
		return c.authenticateWithGitHub()
	case AuthMethodTLS:
		return c.authenticateWithTLS()
	default:
		return fmt.Errorf("unsupported authentication method: %s", c.config.AuthMethod)
	}
}

// authenticateWithToken authenticates using a token
func (c *Client) authenticateWithToken() error {
	var token string

	if c.config.Token != "" {
		token = c.config.Token
	} else if c.config.TokenFile != "" {
		tokenBytes, err := ioutil.ReadFile(c.config.TokenFile)
		if err != nil {
			return fmt.Errorf("failed to read token file: %w", err)
		}
		token = string(tokenBytes)
	} else {
		return ErrTokenNotSet
	}

	c.client.SetToken(token)

	// Lookup token info
	auth, err := c.client.Auth().Token().LookupSelf()
	if err != nil {
		return fmt.Errorf("token lookup failed: %w", err)
	}

	c.authMutex.Lock()
	c.authInfo = &AuthInfo{
		ClientToken:   token,
		Policies:      auth.Data["policies"].([]string),
		LeaseDuration: auth.LeaseDuration,
		Renewable:     auth.Renewable,
	}
	c.authMutex.Unlock()

	return nil
}

// authenticateWithAppRole authenticates using AppRole
func (c *Client) authenticateWithAppRole() error {
	if c.config.AppRole == nil {
		return ErrInvalidAppRoleAuth
	}

	secretID := c.config.AppRole.SecretID
	if secretID == "" && c.config.AppRole.SecretIDFile != "" {
		secretIDBytes, err := ioutil.ReadFile(c.config.AppRole.SecretIDFile)
		if err != nil {
			return fmt.Errorf("failed to read secret ID file: %w", err)
		}
		secretID = string(secretIDBytes)
	}

	data := map[string]interface{}{
		"role_id": c.config.AppRole.RoleID,
	}
	if secretID != "" {
		data["secret_id"] = secretID
	}

	path := c.config.AppRole.MountPath + "/login"
	auth, err := c.client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("approle auth failed: %w", err)
	}

	if auth == nil || auth.Auth == nil {
		return ErrAppRoleAuthFailed
	}

	c.client.SetToken(auth.Auth.ClientToken)

	c.authMutex.Lock()
	c.authInfo = &AuthInfo{
		ClientToken:   auth.Auth.ClientToken,
		Accessor:      auth.Auth.Accessor,
		Policies:      auth.Auth.Policies,
		TokenPolicies: auth.Auth.TokenPolicies,
		Metadata:      auth.Auth.Metadata,
		LeaseDuration: auth.Auth.LeaseDuration,
		Renewable:     auth.Auth.Renewable,
	}
	c.authMutex.Unlock()

	return nil
}

// authenticateWithKubernetes authenticates using Kubernetes service account
func (c *Client) authenticateWithKubernetes() error {
	if c.config.Kubernetes == nil {
		return ErrInvalidKubernetesAuth
	}

	jwt := c.config.Kubernetes.JWT
	if jwt == "" {
		jwtPath := c.config.Kubernetes.JWTPath
		if jwtPath == "" {
			jwtPath = c.config.Kubernetes.ServiceAccountTokenPath
		}

		jwtBytes, err := ioutil.ReadFile(jwtPath)
		if err != nil {
			return fmt.Errorf("failed to read JWT file: %w", err)
		}
		jwt = string(jwtBytes)
	}

	data := map[string]interface{}{
		"role": c.config.Kubernetes.Role,
		"jwt":  jwt,
	}

	path := c.config.Kubernetes.MountPath + "/login"
	auth, err := c.client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("kubernetes auth failed: %w", err)
	}

	if auth == nil || auth.Auth == nil {
		return ErrKubernetesAuthFailed
	}

	c.client.SetToken(auth.Auth.ClientToken)

	c.authMutex.Lock()
	c.authInfo = &AuthInfo{
		ClientToken:   auth.Auth.ClientToken,
		Accessor:      auth.Auth.Accessor,
		Policies:      auth.Auth.Policies,
		TokenPolicies: auth.Auth.TokenPolicies,
		Metadata:      auth.Auth.Metadata,
		LeaseDuration: auth.Auth.LeaseDuration,
		Renewable:     auth.Auth.Renewable,
	}
	c.authMutex.Unlock()

	return nil
}

// authenticateWithAWS authenticates using AWS credentials
func (c *Client) authenticateWithAWS() error {
	if c.config.AWS == nil {
		return ErrInvalidAWSAuth
	}

	data := map[string]interface{}{
		"role": c.config.AWS.Role,
	}

	// Add AWS specific parameters based on type
	if c.config.AWS.Type == "iam" {
		// For IAM authentication, additional AWS signature data is required
		// This is typically handled by AWS SDK
		if c.config.AWS.AccessKeyID != "" {
			data["aws_access_key_id"] = c.config.AWS.AccessKeyID
		}
		if c.config.AWS.SecretAccessKey != "" {
			data["aws_secret_access_key"] = c.config.AWS.SecretAccessKey
		}
		if c.config.AWS.SessionToken != "" {
			data["aws_session_token"] = c.config.AWS.SessionToken
		}
	}

	path := c.config.AWS.MountPath + "/login"
	auth, err := c.client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("aws auth failed: %w", err)
	}

	if auth == nil || auth.Auth == nil {
		return ErrAWSAuthFailed
	}

	c.client.SetToken(auth.Auth.ClientToken)

	c.authMutex.Lock()
	c.authInfo = &AuthInfo{
		ClientToken:   auth.Auth.ClientToken,
		Accessor:      auth.Auth.Accessor,
		Policies:      auth.Auth.Policies,
		TokenPolicies: auth.Auth.TokenPolicies,
		Metadata:      auth.Auth.Metadata,
		LeaseDuration: auth.Auth.LeaseDuration,
		Renewable:     auth.Auth.Renewable,
	}
	c.authMutex.Unlock()

	return nil
}

// Placeholder implementations for other auth methods
func (c *Client) authenticateWithJWT() error {
	// JWT implementation would go here
	return fmt.Errorf("JWT authentication not implemented")
}

func (c *Client) authenticateWithOIDC() error {
	// OIDC implementation would go here
	return fmt.Errorf("OIDC authentication not implemented")
}

func (c *Client) authenticateWithUserpass() error {
	// Userpass implementation would go here
	return fmt.Errorf("Userpass authentication not implemented")
}

func (c *Client) authenticateWithLDAP() error {
	// LDAP implementation would go here
	return fmt.Errorf("LDAP authentication not implemented")
}

func (c *Client) authenticateWithGCP() error {
	// GCP implementation would go here
	return fmt.Errorf("GCP authentication not implemented")
}

func (c *Client) authenticateWithAzure() error {
	// Azure implementation would go here
	return fmt.Errorf("Azure authentication not implemented")
}

func (c *Client) authenticateWithGitHub() error {
	// GitHub implementation would go here
	return fmt.Errorf("GitHub authentication not implemented")
}

func (c *Client) authenticateWithTLS() error {
	// TLS implementation would go here
	return fmt.Errorf("TLS authentication not implemented")
}

// GetSecret retrieves a secret from the specified path
func (c *Client) GetSecret(ctx context.Context, path string) (*SecretData, error) {
	if err := c.checkRateLimit(ctx); err != nil {
		return nil, err
	}

	// Check cache first
	if c.config.EnableCache {
		if secret := c.getCachedSecret(path); secret != nil {
			return secret, nil
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientNotInitialized
	}

	secret, err := c.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret at %s: %w", path, err)
	}

	if secret == nil {
		return nil, ErrSecretNotFound
	}

	secretData := &SecretData{
		Data: secret.Data,
	}

	// Handle KV v2 format
	if data, ok := secret.Data["data"]; ok {
		if dataMap, ok := data.(map[string]interface{}); ok {
			secretData.Data = dataMap
		}
	}

	if metadata, ok := secret.Data["metadata"]; ok {
		if metadataMap, ok := metadata.(map[string]interface{}); ok {
			secretData.Metadata = &SecretMetadata{}
			if created, ok := metadataMap["created_time"].(time.Time); ok {
				secretData.Metadata.CreatedTime = created
			}
			if version, ok := metadataMap["version"].(int); ok {
				secretData.Metadata.Version = version
			}
		}
	}

	// Cache the result
	if c.config.EnableCache {
		c.setCachedSecret(path, secretData)
	}

	return secretData, nil
}

// PutSecret stores a secret at the specified path
func (c *Client) PutSecret(ctx context.Context, path string, data map[string]interface{}) error {
	if err := c.checkRateLimit(ctx); err != nil {
		return err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientNotInitialized
	}

	// For KV v2, wrap data in "data" field
	secretData := map[string]interface{}{
		"data": data,
	}

	_, err := c.client.Logical().WriteWithContext(ctx, path, secretData)
	if err != nil {
		return fmt.Errorf("failed to write secret at %s: %w", path, err)
	}

	// Invalidate cache
	if c.config.EnableCache {
		c.invalidateCachedSecret(path)
	}

	return nil
}

// DeleteSecret deletes a secret at the specified path
func (c *Client) DeleteSecret(ctx context.Context, path string) error {
	if err := c.checkRateLimit(ctx); err != nil {
		return err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientNotInitialized
	}

	_, err := c.client.Logical().DeleteWithContext(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to delete secret at %s: %w", path, err)
	}

	// Invalidate cache
	if c.config.EnableCache {
		c.invalidateCachedSecret(path)
	}

	return nil
}

// ListSecrets lists secrets at the specified path
func (c *Client) ListSecrets(ctx context.Context, path string) ([]string, error) {
	if err := c.checkRateLimit(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientNotInitialized
	}

	secret, err := c.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets at %s: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	result := make([]string, len(keys))
	for i, key := range keys {
		result[i] = key.(string)
	}

	return result, nil
}

// Health performs a health check
func (c *Client) Health(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientNotInitialized
	}

	health, err := c.client.Sys().HealthWithContext(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if health.Sealed {
		return ErrSealedVault
	}

	if !health.Initialized {
		return ErrUninitializedVault
	}

	return nil
}

// IsSealed checks if Vault is sealed
func (c *Client) IsSealed(ctx context.Context) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return true, ErrClientNotInitialized
	}

	status, err := c.client.Sys().SealStatusWithContext(ctx)
	if err != nil {
		return true, fmt.Errorf("failed to get seal status: %w", err)
	}

	return status.Sealed, nil
}

// GetToken returns the current authentication token
func (c *Client) GetToken() string {
	c.authMutex.RLock()
	defer c.authMutex.RUnlock()

	if c.authInfo == nil {
		return ""
	}
	return c.authInfo.ClientToken
}

// GetAuthInfo returns current authentication information
func (c *Client) GetAuthInfo() *AuthInfo {
	c.authMutex.RLock()
	defer c.authMutex.RUnlock()

	if c.authInfo == nil {
		return nil
	}

	// Return a copy to avoid race conditions
	return &AuthInfo{
		ClientToken:   c.authInfo.ClientToken,
		Accessor:      c.authInfo.Accessor,
		Policies:      append([]string{}, c.authInfo.Policies...),
		TokenPolicies: append([]string{}, c.authInfo.TokenPolicies...),
		Metadata:      c.copyStringMap(c.authInfo.Metadata),
		LeaseDuration: c.authInfo.LeaseDuration,
		Renewable:     c.authInfo.Renewable,
	}
}

// Cache management methods
func (c *Client) getCachedSecret(path string) *SecretData {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	entry, exists := c.cache[path]
	if !exists || time.Now().After(entry.expiresAt) {
		return nil
	}

	return entry.data
}

func (c *Client) setCachedSecret(path string, data *SecretData) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.cache[path] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.config.CacheTTL),
	}
}

func (c *Client) invalidateCachedSecret(path string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	delete(c.cache, path)
}

// Token renewal
func (c *Client) startTokenRenewal() error {
	c.authMutex.RLock()
	token := c.authInfo.ClientToken
	renewable := c.authInfo.Renewable
	c.authMutex.RUnlock()

	if !renewable {
		return nil // Token is not renewable
	}

	renewer, err := c.client.NewRenewer(&api.RenewerInput{
		Secret: &api.Secret{
			Auth: &api.SecretAuth{
				ClientToken:   token,
				Renewable:     renewable,
				LeaseDuration: c.authInfo.LeaseDuration,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create token renewer: %w", err)
	}

	c.tokenRenewer = renewer

	go func() {
		defer c.tokenRenewer.Stop()

		for {
			select {
			case renewal := <-c.tokenRenewer.RenewCh():
				// Update auth info with new lease duration
				c.authMutex.Lock()
				if renewal.Secret != nil && renewal.Secret.Auth != nil {
					c.authInfo.LeaseDuration = renewal.Secret.Auth.LeaseDuration
				}
				c.authMutex.Unlock()

			case <-c.tokenRenewer.DoneCh():
				// Token renewal completed or error occurred
				return

			case <-c.renewStop:
				return
			}
		}
	}()

	go c.tokenRenewer.Renew()

	return nil
}

// Health monitoring
func (c *Client) startHealthMonitoring() {
	c.healthTicker = time.NewTicker(c.config.HealthCheckInterval)

	go func() {
		for {
			select {
			case <-c.healthTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				if err := c.Health(ctx); err != nil {
					// Log health check failure - integrate with your logger
					// log.Warn("Vault health check failed", "error", err)
				}
				cancel()
			case <-c.healthStop:
				return
			case <-c.ctx.Done():
				return
			}
		}
	}()
}

// Rate limiting
func (c *Client) checkRateLimit(ctx context.Context) error {
	if c.rateLimiter == nil {
		return nil
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit exceeded: %w", err)
	}

	return nil
}

// Helper methods
func (c *Client) copyStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}

	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// Close gracefully shuts down the client
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	c.cancel()

	// Stop token renewal
	if c.tokenRenewer != nil {
		close(c.renewStop)
		c.tokenRenewer.Stop()
	}

	// Stop health monitoring
	if c.healthTicker != nil {
		c.healthTicker.Stop()
		close(c.healthStop)
	}

	// Clear cache
	c.cacheMutex.Lock()
	c.cache = make(map[string]cacheEntry)
	c.cacheMutex.Unlock()
}
