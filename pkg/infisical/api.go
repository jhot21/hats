package infisical

import (
	"fmt"
	"time"

	"code.jhot.me/jhot/hats/internal/util"
	"github.com/go-resty/resty/v2"
)

type InfisicalClient struct {
	restClient      *resty.Client
	host            string
	clientId        string
	clientSecret    string
	accessToken     string
	tokenExpiration time.Time
}

func New(host string, clientId string, clientSecret string) *InfisicalClient {
	return &InfisicalClient{
		restClient:      resty.New().SetBaseURL(fmt.Sprintf("%s/api", host)),
		host:            host,
		clientId:        clientId,
		clientSecret:    clientSecret,
		accessToken:     "",
		tokenExpiration: time.Now(),
	}
}

func (c *InfisicalClient) Login() error {
	var result LoginResponse
	_, err := util.CheckSuccess(c.restClient.R().SetFormData(map[string]string{
		"clientId":     c.clientId,
		"clientSecret": c.clientSecret,
	}).SetResult(&result).Post("v1/auth/universal-auth/login"))

	if err != nil {
		return err
	}

	c.restClient.SetAuthScheme(result.TokenType).SetAuthToken(result.AccessToken)
	c.accessToken = result.AccessToken
	c.tokenExpiration = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return nil
}

func (c *InfisicalClient) CheckToken() error {
	if c.accessToken == "" {
		return fmt.Errorf("access token empty, please login first")
	}

	if time.Now().Before(c.tokenExpiration) {
		return nil
	}

	var result LoginResponse
	_, err := util.CheckSuccess(c.restClient.R().SetResult(&result).SetBody(map[string]interface{}{
		"accessToken": c.accessToken,
	}).Post("v1/auth/token/renew"))

	if err != nil {
		return fmt.Errorf("error renewing token: %w", err)
	}

	c.restClient.SetAuthScheme(result.TokenType).SetAuthToken(result.AccessToken)
	c.accessToken = result.AccessToken
	c.tokenExpiration = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)

	return nil
}

func (c *InfisicalClient) ListSecrets(workspaceId string, environment string, optFuncs ...RetrieveSecretOptionFunc) ([]Secret, error) {
	opts := DefaultRetrieveSecretOptions()
	optFuncs = append(optFuncs, WithWorkspaceID(workspaceId), WithEnvironment(environment))
	for _, f := range optFuncs {
		f(opts)
	}

	err := c.CheckToken()
	if err != nil {
		return []Secret{}, err
	}

	var result SecretsResponse
	_, err = util.CheckSuccess(c.restClient.R().SetResult(&result).SetQueryParams(map[string]string{
		"workspaceId":     opts.WorkspaceID,
		"environment":     opts.Environment,
		"secretPath":      opts.SecretPath,
		"include_imports": fmt.Sprintf("%t", opts.IncludeImports),
	}).Get("v3/secrets/raw"))

	if err != nil {
		return []Secret{}, err
	}

	return result.Secrets, nil
}

func (c *InfisicalClient) GetSecret(name string, workspaceId string, environment string, optFuncs ...RetrieveSecretOptionFunc) (Secret, error) {
	opts := DefaultRetrieveSecretOptions()
	optFuncs = append(optFuncs, WithWorkspaceID(workspaceId), WithEnvironment(environment))
	for _, f := range optFuncs {
		f(opts)
	}

	err := c.CheckToken()
	if err != nil {
		return Secret{}, err
	}

	var result SecretResponse
	_, err = util.CheckSuccess(c.restClient.R().SetResult(&result).SetQueryParams(map[string]string{
		"workspaceId":     opts.WorkspaceID,
		"environment":     opts.Environment,
		"secretPath":      opts.SecretPath,
		"include_imports": fmt.Sprintf("%t", opts.IncludeImports),
		"type":            "shared",
	}).Get("v3/secrets/raw/" + name))

	if err != nil {
		return Secret{}, err
	}

	return result.Secret, nil
}
