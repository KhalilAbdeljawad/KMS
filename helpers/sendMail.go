package helpers

import (
	"log"
	"net/smtp"
)

func main() {
	SendEmail("hello there, this is a test of sending mail from golang")
}

func SendEmail(body string) {
	from := "email"
	pass := "pass"
	to := "email"

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: Subject\n\n" +
		body

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
		from, []string{to}, []byte(msg))

	if err != nil {
		log.Printf("smtp error: %s", err)
		return
	}

	log.Print("email sent")
}
