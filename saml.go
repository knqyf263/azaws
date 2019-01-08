package main

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	saml "github.com/RobotsAndPencils/go-saml"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

func createSAMLRequest(appID string) (request string, err error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "Failed to generate uuid")
	}
	request = `<samlp:AuthnRequest xmlns="urn:oasis:names:tc:SAML:2.0:metadata" ID="id%s" Version="2.0" IssueInstant="%s" IsPassive="false" AssertionConsumerServiceURL="https://signin.aws.amazon.com/saml" xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol">
	<Issuer xmlns="urn:oasis:names:tc:SAML:2.0:assertion">%s</Issuer>
	<samlp:NameIDPolicy Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"></samlp:NameIDPolicy>
	</samlp:AuthnRequest>`
	request = fmt.Sprintf(request, uuid.String(), time.Now().Format(time.RFC3339), appID)

	// Deflate
	var b bytes.Buffer
	w, err := flate.NewWriter(&b, 9)
	if err != nil {
		return "", errors.Wrap(err, "Failed to get a new Writer compressing data")
	}
	if _, err := w.Write([]byte(request)); err != nil {
		return "", errors.Wrap(err, "Failed to deflate SAML request")
	}
	w.Close()

	// Base64 Encode
	encodedSAMLRequest := base64.StdEncoding.EncodeToString(b.Bytes())
	return encodedSAMLRequest, nil
}

func parseArn(assertion string) (roleArn, principalArn string, err error) {
	response, err := saml.ParseEncodedResponse(assertion)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to parse encoded SAML response")
	}

	for _, attribute := range response.Assertion.AttributeStatement.Attributes {
		if attribute.Name != "http://schemas.microsoft.com/ws/2008/06/identity/claims/role" && attribute.Name != "https://aws.amazon.com/SAML/Attributes/Role" {
			continue
		}
		for _, v := range attribute.AttributeValues {
			s := strings.Split(v.Value, ",")
			roleArn = s[0]
			principalArn = s[1]
			if strings.HasSuffix(roleArn, roleName) {
				break
			}
		}
		break
	}
	if roleArn == "" || principalArn == "" {
		return "", "", errors.Wrapf(err, "The specified role could not be found in SAML response: %s", roleName)
	}
	return roleArn, principalArn, nil
}
