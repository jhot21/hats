package infisical

type LoginResponse struct {
	AccessToken       string `json:"accessToken"`
	ExpiresIn         int    `json:"expiresIn"`
	AccessTokenMaxTTL int    `json:"accessTokenMaxTTL"`
	TokenType         string `json:"tokenType"`
}

type RetrieveSecretOptions struct {
	WorkspaceID    string
	Environment    string
	SecretPath     string
	IncludeImports bool
}

func DefaultRetrieveSecretOptions() *RetrieveSecretOptions {
	return &RetrieveSecretOptions{
		WorkspaceID:    "",
		Environment:    "",
		SecretPath:     "/",
		IncludeImports: false,
	}
}

type RetrieveSecretOptionFunc func(*RetrieveSecretOptions)

func WithWorkspaceID(workspaceId string) RetrieveSecretOptionFunc {
	return func(rso *RetrieveSecretOptions) {
		rso.WorkspaceID = workspaceId
	}
}

func WithEnvironment(environment string) RetrieveSecretOptionFunc {
	return func(rso *RetrieveSecretOptions) {
		rso.Environment = environment
	}
}

func WithSecretPath(secretPath string) RetrieveSecretOptionFunc {
	return func(rso *RetrieveSecretOptions) {
		rso.SecretPath = secretPath
	}
}

func WithImports(rso *RetrieveSecretOptions) {
	rso.IncludeImports = true
}

type Secret struct {
	ID            string `json:"_id"`
	Environment   string `json:"environment,omitempty"`
	SecretComment string `json:"secretComment,omitempty"`
	SecretKey     string `json:"secretKey,omitempty"`
	SecretValue   string `json:"secretValue,omitempty"`
	Version       int    `json:"version,omitempty"`
	Workspace     string `json:"workspace,omitempty"`
}

type SecretsResponse struct {
	Secrets []Secret `json:"secrets"`
}

type SecretResponse struct {
	Secret Secret `json:"secret"`
}
