package main

import (
	"bytes"
	"fmt"
	"github.com/SlyMarbo/gmail"
	"github.com/aws/aws-lambda-go/lambda"
	"os"
	"text/template"
)

type Request struct {
	Name    string `json:name`
	Email   string `json:email`
	Phone   string `json:phone`
	Message string `json:message`
}

type Response struct {
	Message string `json:message`
	Ok      bool   `json:ok`
}

func mail(request Request) error {
	subject := fmt.Sprintf("%s: [[%s]]", os.Getenv("CONTACT_EMAIL_SUBJECT"), request.Name)
	recipient := os.Getenv("CONTACT_EMAIL_RECIPIENT")
	user := os.Getenv("CONTACT_EMAIL_USER")
	password := os.Getenv("CONTACT_EMAIL_PASSWORD")

	const email = `
	Name: {{.Name}}
	Email: {{.Email}}
	Phone: {{.Phone}}

	Message:
	{{.Message}}
	`

	// Create a new template and parse the email into it.
	t := template.Must(template.New("email").Parse(email))
	message := &bytes.Buffer{}
	err := t.Execute(message, request)

	if err != nil {
		return err
	}

	mail := gmail.Compose(subject, message.String())
	mail.From = user
	mail.Password = password

	// Normally you'll only need one of these, but I thought I'd show both.
	mail.AddRecipient(recipient)

	return mail.Send()
}

func Handler(request Request) (Response, error) {
	err := mail(request)
	ok := true
	message := "Message sent successfully."

	if err != nil {
		ok = false
		message = "Failed to send message."
	}

	return Response{
		Message: message,
		Ok:      ok,
	}, err
}

func main() {
	lambda.Start(Handler)
}
