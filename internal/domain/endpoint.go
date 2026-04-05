package domain

import "time"

// Environment represents the deployment environment of an application.
type Environment string

const (
	EnvProduction  Environment = "production"
	EnvStaging     Environment = "staging"
	EnvDevelopment Environment = "development"
)

// AuthType represents the authentication method for a protected endpoint.
type AuthType string

const (
	AuthNone        AuthType = "none"
	AuthBasic       AuthType = "basic"
	AuthBearerToken AuthType = "bearer"
)

// Credentials holds authentication details for a protected pprof endpoint.
type Credentials struct {
	AuthType AuthType `json:"auth_type"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	Token    string   `json:"token,omitempty"`
}

// Endpoint represents a registered Go application with an exposed pprof server.
type Endpoint struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	BaseURL         string        `json:"base_url"`
	Environment     Environment   `json:"environment"`
	CollectInterval time.Duration `json:"collect_interval_s"`
	Credentials     Credentials   `json:"credentials"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}
