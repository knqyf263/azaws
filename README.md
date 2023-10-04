[![Build Status](https://travis-ci.org/knqyf263/azaws.svg?branch=master)](https://travis-ci.org/knqyf263/azaws)
[![Go Report Card](https://goreportcard.com/badge/github.com/knqyf263/azaws)](https://goreportcard.com/report/github.com/knqyf263/azaws)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](https://github.com/knqyf263/azaws/blob/master/LICENSE)

# azaws
If your organization uses [Azure Active Directory](https://azure.microsoft.com) to provide SSO login to the AWS console, then there is no easy way to log in on the command line or to use the [AWS CLI](https://aws.amazon.com/cli/). This tool fixes that. It lets you use the normal Azure AD login (including MFA) from a command line to create a federated AWS session and places the temporary credentials in the proper place for the AWS CLI and SDKs.

Inspired by [aws-azure-login](https://github.com/dtjohnson/aws-azure-login)

## Install

### From source
```
$ go install github.com/knqyf263/azaws@latest
```

### RedHat, CentOS
```
$ sudo rpm -ivh https://github.com/knqyf263/azaws/releases/download/v0.0.1/azaws_0.0.1_Tux_64-bit.rpm
```

### Debian, Ubuntu
```
$ wget https://github.com/knqyf263/azaws/releases/download/v0.0.1/azaws_0.0.1_Tux_64-bit.deb
$ dpkg -i azaws_0.0.1_linux_amd64.deb
```

### Other 
Download binary from https://github.com/knqyf263/azaws/releases

## Usage
### Configuration

```
$ azaws --configure
Azure Tenant ID: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
Azure App ID URI: https://signin.aws.amazon.com/saml?XXXXXXXXXXXX
```

### Log in
The following command will open Google Chrome.

```
$ azaws --role [YOUR ROLE NAME]
```

Enter your credentials and log in to Azure.

After that, you can use aws-cli.
```
$ aws sts get-caller-identity --profile [YOUR_PROFILE_NAME (default: azaws)]
```

### Option

```
Usage of azaws:
  -configure
        Configure options
  -profile string
        AWS profile name (default "azaws")
  -role string
        AWS role name (required)
  -user-data-dir string
        Chrome option (default "/tmp/azaws")
```

## Getting Your Tenant ID and App ID URI

Your Azure AD system admin should be able to provide you with your Tenant ID and App ID URI. If you can't get it from them, you can scrape it from a login page from the myapps.microsoft.com page.

1. Load the myapps.microsoft.com page.
2. Click the chicklet for the login you want.
3. In the window the pops open quickly copy the login.microsoftonline.com URL. (If you miss it just try again. You can also open the developer console with nagivation preservation to capture the URL.)
4. The GUID right after login.microsoftonline.com/ is the tenant ID.
5. Copy the SAMLRequest URL param.
6. Paste it into a URL decoder ([like this one](https://www.samltool.com/url.php)) and decode.
7. Paste the decoded output into the a SAML deflated and encoded XML decoder ([like this one](https://www.samltool.com/decode.php)).
8. In the decoded XML output the value of the Issuer tag is the App ID URI.

## Author

Teppei Fukuda
