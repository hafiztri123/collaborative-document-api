package config

const (
	// Environment
	ENVIRONMENT = "environment"
	ENV_DEV = "development"
	ENV_PROD = "production"

	// Server Configuration Keys
	SERVER_PORT           = "server.port"
	SERVER_TIMEOUT        = "server.timeout"
	SERVER_READ_TIMEOUT   = "server.read_timeout"
	SERVER_WRITE_TIMEOUT  = "server.write_timeout"
	SERVER_MAX_HEADER_MB  = "server.max_header_bytes"

	// Database Configuration Keys
	DB_DRIVER                 = "database.driver"
	DB_HOST                   = "database.host"
	DB_PORT                   = "database.port"
	DB_USERNAME               = "database.username"
	DB_PASSWORD               = "database.password"
	DB_NAME                   = "database.name"
	DB_SSL_MODE               = "database.ssl_mode"
	DB_MAX_IDLE_CONNECTIONS   = "database.max_idle_connections"
	DB_MAX_OPEN_CONNECTIONS   = "database.max_open_connections"
	DB_CONNECTION_MAX_LIFETIME = "database.connection_max_lifetime"

	// Redis Configuration Keys
	REDIS_HOST     = "redis.host"
	REDIS_PORT     = "redis.port"
	REDIS_PASSWORD = "redis.password"
	REDIS_DB       = "redis.db"

	// JWT Configuration Keys
	JWT_SECRET                 = "jwt.secret"
	JWT_ACCESS_TOKEN_EXPIRY     = "jwt.access_token_expiry"
	JWT_REFRESH_TOKEN_EXPIRY    = "jwt.refresh_token_expiry"

	// Logging Configuration Keys
	LOG_LEVEL  = "logging.level"
	LOG_FORMAT = "logging.format"

	// Rate Limit Configuration Keys
	RATE_LIMIT_REQUESTS = "rate_limit.requests"
	RATE_LIMIT_DURATION = "rate_limit.duration"
)
