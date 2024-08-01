package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
)

var (
	configureMode bool
	profileName   string
	roleName      string
	userDataDir   string

	msgChann = make(chan cdproto.Message)
)

func devToolHandler(s string, is ...interface{}) {
	go func() {
		for _, elem := range is {
			var msg cdproto.Message
			// The CDP messages are sent as strings so we need to convert them back
			json.Unmarshal([]byte(fmt.Sprintf("%s", elem)), &msg)
			msgChann <- msg
		}
	}()
}

func handleSignal(cancel context.CancelFunc) {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		for {
			s := <-signalChannel
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				cancel()
			}
		}
	}()
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	flag.BoolVar(&configureMode, "configure", false, "Configure options")
	flag.StringVar(&profileName, "profile", "azaws", "AWS profile name")
	flag.StringVar(&roleName, "role", "", "AWS role name (required)")
	flag.StringVar(&userDataDir, "user-data-dir", "/tmp/azaws", "Chrome option")
	flag.Parse()

	if configureMode {
		return configure()
	}

	tenantID, appID, durationHours, err := getProfileConfig(profileName)
	if err != nil {
		return errors.Wrap(err, "Failed to get config parameters")
	}
	if tenantID == "" || appID == "" {
		return errors.New("You must configure it first with --configure")
	}

	if roleName == "" {
		os.Stderr.WriteString("the 'role' option is required\n")
		usage()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handleSignal(cancel)

	// create chrome instance
	ctx, cancel = chromedp.NewExecAllocator(ctx, chromedp.Flag("user-data-dir", userDataDir), chromedp.Flag("disable-infobars", true))
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx, chromedp.WithDebugf(devToolHandler))
	defer cancel()

	err = chromedp.Run(ctx, network.Enable())
	if err != nil {
		return err
	}

	samlRequest, err := createSAMLRequest(appID)
	if err != nil {
		return err
	}

	loginURL := `https://login.microsoftonline.com/%s/saml2?SAMLRequest=%s`
	loginURL = fmt.Sprintf(loginURL, tenantID, url.QueryEscape(samlRequest))

	err = chromedp.Run(ctx, chromedp.Navigate(loginURL))
	if err != nil {
		return err
	}

	err = chromedp.Run(ctx, chromedp.ActionFunc(func(_ context.Context) error {
		for {
			var msg cdproto.Message
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg = <-msgChann:
			}
			switch msg.Method.String() {
			case "Network.requestWillBeSent":
				var reqWillSend network.EventRequestWillBeSent
				if err = json.Unmarshal(msg.Params, &reqWillSend); err != nil {
					return err
				}
				if reqWillSend.Request.URL != "https://signin.aws.amazon.com/saml" || !reqWillSend.Request.HasPostData ||
					len(reqWillSend.Request.PostDataEntries) == 0 {
					continue
				}
				dec, err := base64.StdEncoding.DecodeString(reqWillSend.Request.PostDataEntries[0].Bytes)
				if err != nil {
					return errors.Wrap(err, "Failed to decode post data")
				}
				form, err := url.ParseQuery(string(dec))
				if err != nil {
					return errors.Wrap(err, "Failed to parse query")
				}
				samlResponse, ok := form["SAMLResponse"]
				if !ok || len(samlResponse) == 0 {
					return errors.Wrap(err, "No such key: SAMLResponse")
				}
				err = assumeRole(ctx, samlResponse[0], durationHours)
				if err != nil {
					return errors.Wrap(err, "Failed to assume role")
				}
				return nil
			}
		}
	}))
	if err != nil {
		return errors.Wrap(err, "Failed to handle events")
	}
	return nil
}

func assumeRole(ctx context.Context, assertion string, durationHours int) error {
	roleArn, principalArn, err := parseArn(assertion)
	if err != nil {
		return errors.Wrap(err, "Failed to parse arn from SAML response")
	}

	credentials, err := assumeRoleWithSAML(ctx, roleArn, principalArn, assertion, durationHours)
	if err != nil {
		return errors.Wrap(err, "Failed to assume role with SAML")
	}
	return setProfileCredentials(profileName, credentials)
}

func configure() error {
	tenantID, err := prompt("Azure Tenant ID: ")
	if err != nil {
		return errors.Wrap(err, "Failed to get Tenant ID")
	}
	appID, err := prompt("Azure App ID URI: ")
	if err != nil {
		return errors.Wrap(err, "Failed to get Azure App ID URI")
	}
	durationHours, err := promptInt("Session Duration Hours (up to 12): ")
	if err != nil {
		return errors.Wrap(err, "Failed to get Azure App ID URI")
	}
	return setProfileConfig(profileName, tenantID, appID, durationHours)
}

func prompt(q string) (string, error) {
	fmt.Print(q)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	answer := scanner.Text()
	if err := scanner.Err(); err != nil {
		return "", errors.Wrap(err, "Failed to parse user input")
	}
	return answer, nil
}

func promptInt(q string) (num int, err error) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print(q)
	for scanner.Scan() {
		answer := scanner.Text()
		num, err = strconv.Atoi(answer)
		if err != nil {
			fmt.Println("Enter number")
			fmt.Print(q)
			continue
		}

		if err := scanner.Err(); err != nil {
			return -1, errors.Wrap(err, "Failed to parse user input")
		}
		break
	}
	return num, nil
}
