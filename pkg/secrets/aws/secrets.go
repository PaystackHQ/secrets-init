package aws

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"secrets-init/pkg/secrets"
)

const (
	paramNameTokens            = 6
	paramNameTokensWithVersion = 7
)

// SecretsProvider AWS secrets provider.
type SecretsProvider struct {
	sm  SecretsManagerAPI
	ssm SSMAPI
}

// SecretsManagerAPI contains the Secrets Manager operation used by the provider.
type SecretsManagerAPI interface {
	GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// SSMAPI contains the Systems Manager operation used by the provider.
type SSMAPI interface {
	GetParameter(context.Context, *ssm.GetParameterInput, ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

// NewAwsSecretsProvider initializes the AWS secrets provider.
func NewAwsSecretsProvider(ctx context.Context) (secrets.Provider, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load AWS configuration")
	}

	return &SecretsProvider{
		sm:  secretsmanager.NewFromConfig(cfg),
		ssm: ssm.NewFromConfig(cfg),
	}, nil
}

// ResolveSecrets replaces AWS Secrets Manager and Parameter Store references with their values.
func (sp *SecretsProvider) ResolveSecrets(ctx context.Context, vars []string) ([]string, error) {
	envs := make([]string, 0, len(vars))

	for _, env := range vars {
		resolved, err := sp.resolveEnvironmentVariable(ctx, env)
		if err != nil {
			return vars, err
		}
		envs = append(envs, resolved...)
	}

	sort.Strings(envs)
	return envs, nil
}

func (sp *SecretsProvider) resolveEnvironmentVariable(ctx context.Context, env string) ([]string, error) {
	key, value, found := strings.Cut(env, "=")
	if !found {
		return []string{env}, nil
	}

	switch {
	case isSecretsManagerARN(value):
		return sp.resolveSecretsManagerValue(ctx, env, key, value)
	case isParameterStoreARN(value):
		return sp.resolveParameterStoreValue(ctx, env, key, value)
	default:
		return []string{env}, nil
	}
}

func (sp *SecretsProvider) resolveSecretsManagerValue(ctx context.Context, env, key, value string) ([]string, error) {
	secretKey, nestedKey, _ := strings.Cut(value, "$")
	secret, err := sp.sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: awssdk.String(secretKey)})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret from AWS Secrets Manager")
	}
	if secret.SecretString == nil {
		return nil, errors.New("AWS Secrets Manager secret does not contain a string value")
	}

	if !IsJSON(secret.SecretString) {
		return []string{key + "=" + awssdk.ToString(secret.SecretString)}, nil
	}
	if nestedKey != "" {
		jsonValue := gjson.Get(awssdk.ToString(secret.SecretString), nestedKey)
		if !jsonValue.Exists() {
			return []string{env}, nil
		}
		return []string{key + "=" + jsonValue.String()}, nil
	}

	var keyValueSecret map[string]string
	if err = json.Unmarshal([]byte(awssdk.ToString(secret.SecretString)), &keyValueSecret); err != nil {
		return nil, errors.Wrap(err, "failed to decode key/value secret")
	}

	resolved := make([]string, 0, len(keyValueSecret))
	for secretKey, secretValue := range keyValueSecret {
		resolved = append(resolved, secretKey+"="+secretValue)
	}
	return resolved, nil
}

func (sp *SecretsProvider) resolveParameterStoreValue(ctx context.Context, env, key, value string) ([]string, error) {
	paramName, valid := parameterName(value)
	if !valid {
		return []string{env}, nil
	}

	param, err := sp.ssm.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           awssdk.String(paramName),
		WithDecryption: awssdk.Bool(true),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret from AWS Parameters Store")
	}
	if param.Parameter == nil || param.Parameter.Value == nil {
		return nil, errors.New("AWS Parameter Store parameter does not contain a value")
	}

	return []string{key + "=" + awssdk.ToString(param.Parameter.Value)}, nil
}

func isSecretsManagerARN(value string) bool {
	return strings.HasPrefix(value, "arn:aws:secretsmanager") || strings.HasPrefix(value, "arn:aws-cn:secretsmanager")
}

func isParameterStoreARN(value string) bool {
	return (strings.HasPrefix(value, "arn:aws:ssm") || strings.HasPrefix(value, "arn:aws-cn:ssm")) &&
		strings.Contains(value, ":parameter/")
}

func parameterName(value string) (string, bool) {
	tokens := strings.Split(value, ":")
	if len(tokens) != paramNameTokens && len(tokens) != paramNameTokensWithVersion {
		return "", false
	}

	name := strings.TrimPrefix(tokens[5], "parameter")
	if len(tokens) == paramNameTokensWithVersion {
		name += ":" + tokens[6]
	}
	return name, true
}

// IsJSON reports whether str contains valid JSON.
func IsJSON(str *string) bool {
	if str == nil {
		return false
	}
	return json.Valid([]byte(*str))
}
