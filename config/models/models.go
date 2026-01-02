package models

// APIConfig represents a single API configuration
type APIConfig struct {
	Alias     string   `json:"alias"`
	Provider  string   `json:"provider"` // API provider type
	APIKey    string   `json:"api_key"`
	AuthToken string   `json:"auth_token"`
	BaseURL   string   `json:"base_url"`
	Model     string   `json:"model"`            // Currently active model
	Models    []string `json:"models,omitempty"` // Supported models list
}

// File represents the structure of the config file
type File struct {
	Active  string     `json:"active"`
	Configs []APIConfig `json:"configs"`
}
