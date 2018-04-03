package main

import (
	"bytes"
	"fmt"
	"github.com/SlyMarbo/gmail"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"net/url"
	"os"
	"text/template"
)

type Contact struct {
	Name    string `json:name`
	Email   string `json:email`
	Phone   string `json:phone`
	Message string `json:message`
}

func mail(contact Contact) error {
	subject := fmt.Sprintf("%s: [[%s]]", os.Getenv("CONTACT_EMAIL_SUBJECT"), contact.Name)
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
	err := t.Execute(message, contact)

	if err != nil {
		return err
	}

	mail := gmail.Compose(subject, message.String())
	mail.From = user
	mail.Password = password
	mail.AddRecipient(recipient)

	return mail.Send()
}

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	message := "Message sent successfully."
	status := 200

	query, err := url.ParseQuery(request.Body)

	if err != nil {
		message = fmt.Sprint("Could not unmarshal the form query: ", err)
		status = 500
	}

	err = mail(Contact{
		Name:    query.Get("name"),
		Email:   query.Get("email"),
		Phone:   query.Get("phone"),
		Message: query.Get("message"),
	})

	if err != nil {
		message = "Failed to send message."
		status = 500
	}

	return events.APIGatewayProxyResponse{
		Body:       message,
		StatusCode: status,
	}, err
}

func main() {
	lambda.Start(Handler)
}
