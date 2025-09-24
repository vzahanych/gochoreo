package vault

import "errors"

// Configuration validation errors
var (
	ErrInvalidAddress        = errors.New("invalid address: address cannot be empty")
	ErrInvalidTimeout        = errors.New("invalid timeout: timeout must be greater than 0")
	ErrInvalidMaxRetries     = errors.New("invalid max retries: must be greater than or equal to 0")
	ErrInvalidTokenAuth      = errors.New("invalid token auth: token or token_file must be provided")
	ErrInvalidAppRoleAuth    = errors.New("invalid approle auth: role_id must be provided")
	ErrInvalidKubernetesAuth = errors.New("invalid kubernetes auth: role must be provided")
	ErrInvalidJWTAuth        = errors.New("invalid jwt auth: role and jwt must be provided")
	ErrInvalidAWSAuth        = errors.New("invalid aws auth: role must be provided")
	ErrInvalidGCPAuth        = errors.New("invalid gcp auth: role must be provided")
	ErrInvalidAzureAuth      = errors.New("invalid azure auth: role must be provided")
	ErrInvalidUserpassAuth   = errors.New("invalid userpass auth: username and password must be provided")
	ErrInvalidLDAPAuth       = errors.New("invalid ldap auth: username and password must be provided")
)

// Client operation errors
var (
	ErrClientNotInitialized    = errors.New("vault client not initialized")
	ErrTokenNotSet             = errors.New("vault token not set")
	ErrTokenExpired            = errors.New("vault token expired")
	ErrTokenRenewalFailed      = errors.New("token renewal failed")
	ErrAuthenticationFailed    = errors.New("vault authentication failed")
	ErrPermissionDenied        = errors.New("permission denied")
	ErrSecretNotFound          = errors.New("secret not found")
	ErrSecretsEngineNotEnabled = errors.New("secrets engine not enabled at path")
	ErrInvalidSecretPath       = errors.New("invalid secret path")
	ErrInvalidSecretData       = errors.New("invalid secret data")
	ErrSealedVault             = errors.New("vault is sealed")
	ErrUninitializedVault      = errors.New("vault is not initialized")
	ErrInvalidPolicy           = errors.New("invalid vault policy")
	ErrPolicyNotFound          = errors.New("vault policy not found")
)

// HTTP and network errors
var (
	ErrConnectionFailed   = errors.New("failed to connect to vault server")
	ErrConnectionTimeout  = errors.New("connection timeout")
	ErrRequestTimeout     = errors.New("request timeout")
	ErrTooManyRequests    = errors.New("too many requests (rate limited)")
	ErrServerError        = errors.New("vault server error")
	ErrBadGateway         = errors.New("bad gateway")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrGatewayTimeout     = errors.New("gateway timeout")
)

// Secrets engine specific errors
var (
	ErrKVEngineError       = errors.New("kv secrets engine error")
	ErrDatabaseEngineError = errors.New("database secrets engine error")
	ErrPKIEngineError      = errors.New("pki secrets engine error")
	ErrTransitEngineError  = errors.New("transit secrets engine error")
	ErrAWSEngineError      = errors.New("aws secrets engine error")
	ErrGCPEngineError      = errors.New("gcp secrets engine error")
	ErrAzureEngineError    = errors.New("azure secrets engine error")
)

// Authentication specific errors
var (
	ErrAppRoleAuthFailed    = errors.New("approle authentication failed")
	ErrAWSAuthFailed        = errors.New("aws authentication failed")
	ErrKubernetesAuthFailed = errors.New("kubernetes authentication failed")
	ErrJWTAuthFailed        = errors.New("jwt authentication failed")
	ErrOIDCAuthFailed       = errors.New("oidc authentication failed")
	ErrUserpassAuthFailed   = errors.New("userpass authentication failed")
	ErrLDAPAuthFailed       = errors.New("ldap authentication failed")
	ErrGCPAuthFailed        = errors.New("gcp authentication failed")
	ErrAzureAuthFailed      = errors.New("azure authentication failed")
	ErrGitHubAuthFailed     = errors.New("github authentication failed")
	ErrTLSAuthFailed        = errors.New("tls authentication failed")
)

// Health check errors
var (
	ErrHealthCheckFailed = errors.New("vault health check failed")
	ErrVaultUnavailable  = errors.New("vault unavailable")
	ErrLeaderNotFound    = errors.New("vault leader not found")
	ErrClusterNotReady   = errors.New("vault cluster not ready")
)

// Token management errors
var (
	ErrTokenLookupFailed     = errors.New("token lookup failed")
	ErrTokenCreationFailed   = errors.New("token creation failed")
	ErrTokenRevocationFailed = errors.New("token revocation failed")
	ErrWrappingTokenFailed   = errors.New("wrapping token creation failed")
	ErrUnwrappingFailed      = errors.New("unwrapping failed")
)

// Policy errors
var (
	ErrPolicyCreationFailed = errors.New("policy creation failed")
	ErrPolicyUpdateFailed   = errors.New("policy update failed")
	ErrPolicyDeletionFailed = errors.New("policy deletion failed")
	ErrPolicyListFailed     = errors.New("policy list failed")
)

// Audit errors
var (
	ErrAuditDeviceNotFound      = errors.New("audit device not found")
	ErrAuditDeviceEnableFailed  = errors.New("audit device enable failed")
	ErrAuditDeviceDisableFailed = errors.New("audit device disable failed")
	ErrAuditLogFailed           = errors.New("audit log failed")
)

// Mount errors
var (
	ErrMountNotFound = errors.New("mount not found")
	ErrMountFailed   = errors.New("mount operation failed")
	ErrUnmountFailed = errors.New("unmount operation failed")
	ErrRemountFailed = errors.New("remount operation failed")
	ErrTuneFailed    = errors.New("tune operation failed")
)

// Lease errors
var (
	ErrLeaseNotFound         = errors.New("lease not found")
	ErrLeaseRenewalFailed    = errors.New("lease renewal failed")
	ErrLeaseRevocationFailed = errors.New("lease revocation failed")
	ErrLeaseExpired          = errors.New("lease expired")
)

// Encryption/Decryption errors
var (
	ErrEncryptionFailed    = errors.New("encryption failed")
	ErrDecryptionFailed    = errors.New("decryption failed")
	ErrKeyNotFound         = errors.New("encryption key not found")
	ErrKeyGenerationFailed = errors.New("key generation failed")
	ErrKeyRotationFailed   = errors.New("key rotation failed")
)

// Cache errors
var (
	ErrCacheNotEnabled          = errors.New("cache not enabled")
	ErrCacheFull                = errors.New("cache is full")
	ErrCacheKeyNotFound         = errors.New("cache key not found")
	ErrCacheSerializationFailed = errors.New("cache serialization failed")
)
