package tools

import (
	"fmt"
	"net"
	"net/smtp"

	"github.com/jordan-wright/email"
)

var (
	smtpServerURL string
	auth          smtp.Auth
	fromWithName  string
)

// InitEmailConfig init email config
func InitEmailConfig(server string, port int, from, name, password string) {
	smtpServerURL = net.JoinHostPort(server, fmt.Sprintf("%d", port))
	auth = smtp.PlainAuth("", from, password, server)
	if name != "" {
		fromWithName = fmt.Sprintf("%s <%s>", name, from)
	} else {
		fromWithName = from
	}
}

// SendEmail send email
func SendEmail(to, cc []string, subject, content string) error {
	return SendEmailWithAttach(to, cc, subject, content, nil)
}

// SendEmailWithAttach send email with attach
func SendEmailWithAttach(to, cc []string, subject, content string, attachFiles []string) error {
	e := email.NewEmail()
	e.From = fromWithName
	e.To = to
	e.Cc = cc
	e.Subject = subject
	e.Text = []byte(content)
	for _, file := range attachFiles {
		_, err := e.AttachFile(file)
		if err != nil {
			fmt.Printf("attach file '%v' failed. err=%v", file, err)
		}
	}
	return e.Send(smtpServerURL, auth)
}
