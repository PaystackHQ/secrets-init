package google

import (
	"context"

	secretspb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	gax "github.com/googleapis/gax-go/v2"
)

// SecretsManagerAPI is the interface for the Google Secrets Manager API.
type SecretsManagerAPI interface {
	AccessSecretVersion(ctx context.Context, req *secretspb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretspb.AccessSecretVersionResponse, error) //nolint:lll
}
