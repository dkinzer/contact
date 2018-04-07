package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/SlyMarbo/gmail"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dpapathanasiou/go-recaptcha"
	"log"
	"net/url"
	"os"
	"strings"
	"text/template"
)

var (
	ErrorMailConfiguration         = errors.New("Missing required configuration for mailing.")
	ErrorContactInfo               = errors.New("Missing required contact form fields.")
	ErrorFailedCaptchaConfirmation = errors.New("Failed reCAPTCHA verification")
)

type Contact struct {
	Name    string `json:name`
	Email   string `json:email`
	Phone   string `json:phone`
	Message string `json:message`
}

type MailConfiguration struct {
	Subject    string
	Recipients []string
	User       string
	Password   string
}

type Captcha struct {
	Secret   string
	Response string
	ClientIp string
}

func GetDefaultMailConfiguration() (MailConfiguration, error) {
	subject := os.Getenv("CONTACT_EMAIL_SUBJECT")
	if subject == "" {
		return MailConfiguration{}, ErrorMailConfiguration
	}

	recipients := os.Getenv("CONTACT_EMAIL_RECIPIENTS")
	if recipients == "" {
		return MailConfiguration{}, ErrorMailConfiguration
	}

	user := os.Getenv("CONTACT_EMAIL_USER")
	if user == "" {
		return MailConfiguration{}, ErrorMailConfiguration
	}

	password := os.Getenv("CONTACT_EMAIL_PASSWORD")
	if password == "" {
		return MailConfiguration{}, ErrorMailConfiguration
	}

	return MailConfiguration{
		Subject:    subject,
		Recipients: strings.Split(recipients, ","),
		User:       user,
		Password:   password,
	}, nil
}

func mail(contact Contact) error {
	const email = `
	Name: {{.Name}}
	Email: {{.Email}}
	Phone: {{.Phone}}

	Message:
	{{.Message}}
	`
	var (
		err     error
		config  MailConfiguration
		subject string
	)

	// Create a new template and parse the email into it.
	t := template.Must(template.New("email").Parse(email))
	message := &bytes.Buffer{}
	err = t.Execute(message, contact)

	if err != nil {
		log.Println(err)
		return err
	}

	config, err = GetDefaultMailConfiguration()
	if err != nil {
		return err
	}

	subject = fmt.Sprintf("%s: [[%s]]", config.Subject, contact.Name)
	mail := gmail.Compose(subject, message.String())
	mail.From = config.User
	mail.Password = config.Password

	for i := 0; i < len(config.Recipients); i++ {
		mail.AddRecipient(config.Recipients[i])
	}

	return mail.Send()
}

func GetContact(request events.APIGatewayProxyRequest) (Contact, error) {
	var (
		query   url.Values
		contact Contact
		err     error
	)

	query, err = url.ParseQuery(request.Body)

	if err != nil {
		return contact, err
	}

	contact.Name = query.Get("name")
	if contact.Name == "" {
		return contact, ErrorContactInfo
	}

	contact.Email = query.Get("email")
	if contact.Email == "" {
		return contact, ErrorContactInfo
	}

	contact.Phone = query.Get("phone")
	if contact.Phone == "" {
		return contact, ErrorContactInfo
	}

	contact.Message = query.Get("message")
	if contact.Message == "" {
		return contact, ErrorContactInfo
	}

	if !hasValidCaptchaResponse(request) {
		return contact, ErrorFailedCaptchaConfirmation
	}

	return contact, err
}

func GetCaptcha(request events.APIGatewayProxyRequest) Captcha {
	var (
		captcha Captcha
		query   url.Values
		err     error
	)

	captcha.Secret = os.Getenv("CAPTCHA_SECRET")

	if captcha.Secret == "" {
		return captcha
	}

	query, err = url.ParseQuery(request.Body)
	if err == nil {
		captcha.Response = query.Get("g-recaptcha-response")
	}

	captcha.ClientIp = request.RequestContext.Identity.SourceIP

	return captcha
}

func hasValidCaptchaResponse(request events.APIGatewayProxyRequest) bool {
	var (
		captcha Captcha
		valid   bool = true
		err     error
	)

	captcha = GetCaptcha(request)

	if captcha.Secret == "" {
		return true
	}

	recaptcha.Init(captcha.Secret)
	// This will fail on multiple attempts (good thing).
	valid, err = recaptcha.Confirm(captcha.ClientIp, captcha.Response)

	if err != nil {
		log.Println(err)
		return false
	}

	return valid
}

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var (
		contact  Contact
		response events.APIGatewayProxyResponse
		message  string = `{ "message": "%s" }`
		err      error
	)

	response = events.APIGatewayProxyResponse{
		Body:       `{ "message": "Message sent successfully" }`,
		StatusCode: 200,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
	}

	contact, err = GetContact(request)

	if err != nil {
		log.Println(err)
		response.Body = fmt.Sprintf(message, err)
		response.StatusCode = 400
		return response, nil
	}

	err = mail(contact)

	if err != nil {
		log.Println(err)
		response.Body = fmt.Sprintf(message, err)
		response.StatusCode = 500
		return response, nil
	}

	return response, nil
}

func main() {
	lambda.Start(Handler)
}
