package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterOSArgs(t *testing.T) {

	var whitelist = []string{
		"api_server_github_auth_redirect_url",
		"github_client_id",
		"github_app_private_key_file_path",
		"key_manager_aws_kms_master_key_id",
		"blob_store_aws_s3_bucket_name",
		"database_driver",
		"blob_store_type",
		"github_app_id",
		"key_manager_type",
	}

	var in = []string{
		"/usr/bin/bb-server",
		"--api_server_github_auth_redirect_url",
		"https://app.staging.changeme.com/api/v1/authentication/github/callback",
		"--github_client_id",
		"Iv1.53f6349a14a8c00d",
		"--github_app_private_key_file_path",
		"/tmp/github-private-key.pem",
		"--key_manager_aws_kms_master_key_id",
		"arn:aws:kms:us-west-2:733436759586:alias/buildbeaver-staging-kms-data-key",
		"--blob_store_aws_s3_bucket_name",
		"buildbeaver-staging-data",
		"--github_client_secret",
		"secret",
		"--database_driver",
		"postgres",
		"--blob_store_type",
		"AWS_S3",
		"--github_app_id",
		"234695",
		"--api_server_session_encryption_key",
		"secret",
		"--database_connection_string",
		"secret",
		"--key_manager_type",
		"AWS_KMS",
		"--api_server_session_authentication_key",
		"secret"}

	var out = []string{
		"/usr/bin/bb-server",
		"--api_server_github_auth_redirect_url",
		"https://app.staging.changeme.com/api/v1/authentication/github/callback",
		"--github_client_id",
		"Iv1.53f6349a14a8c00d",
		"--github_app_private_key_file_path",
		"/tmp/github-private-key.pem",
		"--key_manager_aws_kms_master_key_id",
		"arn:aws:kms:us-west-2:733436759586:alias/buildbeaver-staging-kms-data-key",
		"--blob_store_aws_s3_bucket_name",
		"buildbeaver-staging-data",
		"--github_client_secret",
		"******",
		"--database_driver",
		"postgres",
		"--blob_store_type",
		"AWS_S3",
		"--github_app_id",
		"234695",
		"--api_server_session_encryption_key",
		"******",
		"--database_connection_string",
		"******",
		"--key_manager_type",
		"AWS_KMS",
		"--api_server_session_authentication_key",
		"******"}

	filtered := FilterOSArgs(in, whitelist)
	require.Equal(t, out, filtered)
}
