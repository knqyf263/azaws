package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/go-ini/ini"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

func assumeRoleWithSAML(ctx context.Context, roleArn, principalArn, assertion string, durationHours int) (*sts.Credentials, error) {
	sess := session.Must(session.NewSession())
	svc := sts.New(sess)

	input := &sts.AssumeRoleWithSAMLInput{
		DurationSeconds: aws.Int64(int64(durationHours) * 60 * 60),
		RoleArn:         aws.String(roleArn),
		PrincipalArn:    aws.String(principalArn),
		SAMLAssertion:   aws.String(assertion),
	}
	res, err := svc.AssumeRoleWithSAMLWithContext(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to AssumeRoleWithSAMLWithContext")
	}
	return res.Credentials, nil
}

func getProfileConfig(profileName string) (tenantID, appID string, durationHours int, err error) {
	configFile, err := getAWSConfigFilePath("config")
	if err != nil {
		return "", "", -1, errors.Wrapf(err, "Failed to get AWS config file path")
	}
	cfg, err := ini.Load(configFile)
	if err != nil {
		return "", "", -1, errors.Wrapf(err, "Failed to read file: %s", configFile)
	}
	if profileName != "default" {
		profileName = fmt.Sprintf("profile %s", profileName)
	}
	tenantID = cfg.Section(profileName).Key("azure_tenant_id").MustString("")
	appID = cfg.Section(profileName).Key("azure_app_id").MustString("")
	durationHours = cfg.Section(profileName).Key("azure_duration_hours").MustInt(1)
	return tenantID, appID, durationHours, nil
}

func setProfileConfig(profileName string, tenantID, appID string, durationHours int) error {
	configFile, err := getAWSConfigFilePath("config")
	if err != nil {
		return errors.Wrapf(err, "Failed to get AWS config file path")
	}
	cfg, err := ini.Load(configFile)
	if err != nil {
		return errors.Wrapf(err, "Failed to read file: %s", configFile)
	}
	if profileName != "default" {
		profileName = fmt.Sprintf("profile %s", profileName)
	}
	cfg.Section(profileName).Key("azure_tenant_id").SetValue(tenantID)
	cfg.Section(profileName).Key("azure_app_id").SetValue(appID)
	cfg.Section(profileName).Key("azure_duration_hours").SetValue(fmt.Sprint(durationHours))
	return cfg.SaveTo(configFile)
}

func setProfileCredentials(profileName string, credentials *sts.Credentials) error {
	credentialFile, err := getAWSConfigFilePath("credentials")
	if err != nil {
		return errors.Wrapf(err, "Failed to get AWS config file path")
	}
	cfg, err := ini.Load(credentialFile)
	if err != nil {
		return errors.Wrapf(err, "Failed to read file: %s", credentialFile)
	}
	cfg.Section(profileName).Key("aws_access_key_id").SetValue(*credentials.AccessKeyId)
	cfg.Section(profileName).Key("aws_secret_access_key").SetValue(*credentials.SecretAccessKey)
	cfg.Section(profileName).Key("aws_session_token").SetValue(*credentials.SessionToken)
	cfg.Section(profileName).Key("aws_session_expiration").SetValue(credentials.Expiration.String())
	return cfg.SaveTo(credentialFile)
}

func getAWSConfigFilePath(fileName string) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get home directory")
	}
	filePath := filepath.Join(homeDir, ".aws", fileName)
	return filePath, nil
}
