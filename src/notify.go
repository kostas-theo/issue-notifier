package main

import (
	"fmt"

	gomail "gopkg.in/gomail.v2"
	hermes "github.com/matcornic/hermes/v2"
)

func notify(issues []Issue, state *map[string]IssueState) error {
	var err error
	tempState := *state
	for _, issue := range issues {
		s, ok := tempState[fmt.Sprintf("%v",issue.ID)]
		if !ok { continue }
		if s.Notified && issue.numUpvotes() < notifyUpvotes {
			s.Notified = false
			tempState[fmt.Sprintf("%v",issue.ID)] = s
		}
		if !s.Notified && issue.numUpvotes() >= notifyUpvotes {
			err = sendNotification(issue)
			if err != nil {
				return err
			}
			s.Notified = true
			tempState[fmt.Sprintf("%v",issue.ID)] = s
		}
	}
	state = &tempState
	return nil
}

func sendNotification(issue Issue) error {
	fmt.Printf("Sending notification for issue \"%v\"...\n",issue.Title)
	d := gomail.NewPlainDialer(smtpEndpoint, smtpPort, smtpUser, smtpPass)
	m := gomail.NewMessage()

	m.SetHeader("From", smtpUser)
	m.SetHeader("To", notifyAddress)
	m.SetHeader("Subject", "New suggestion for Platform Engineering")

	html, text := genEmailBody(issue)
	m.SetBody("text/plain", text)
	m.SetBody("text/html", html)

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	fmt.Printf("notification sent successfully\n\n")
	return nil
}

func genEmailBody(issue Issue) (string, string) {
	repoLink:= fmt.Sprintf("https://github.com/%v/%v", org, repoName)

	h := hermes.Hermes{
		Product: hermes.Product{
			Name:      repoName,
			Link:      repoLink,
			Logo:      notifyEmailLogo,
			Copyright: "",
		},
	}

	email := hermes.Email{
		Body: hermes.Body{
			Title: repoName,
			Intros: []string{
				fmt.Sprintf("A new %v has reached the necessary %d upvotes:", notifyItemName, notifyUpvotes),
			},
			Actions: []hermes.Action{
				{
						Instructions: "",
						Button: hermes.Button{
								Color: "#22BC66",
								Text:  fmt.Sprintf("Check new %v", notifyItemName),
								Link:  fmt.Sprintf("https://github.com/%v/%v/issues/%d", org, repoName, issue.Number),
						},
				},
		},
			Outros: []string{
				"Raise a new ticket and add the reference to that ticket in the \"Current Issues\" section here:",
				repoLink,
			},
		},
	}

	html, _ := h.GenerateHTML(email)
	text, _ := h.GeneratePlainText(email)

	return html, text
}