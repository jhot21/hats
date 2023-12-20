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

func (o *RetrieveSecretOptions) WithWorkspaceID(workspaceId string) *RetrieveSecretOptions {
	o.WorkspaceID = workspaceId
	return o
}

func (o *RetrieveSecretOptions) WithEnvironment(environment string) *RetrieveSecretOptions {
	o.Environment = environment
	return o
}

func (o *RetrieveSecretOptions) WithSecretPath(secretPath string) *RetrieveSecretOptions {
	o.SecretPath = secretPath
	return o
}

func (o *RetrieveSecretOptions) WithImports(includeImports bool) *RetrieveSecretOptions {
	o.IncludeImports = includeImports
	return o
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
