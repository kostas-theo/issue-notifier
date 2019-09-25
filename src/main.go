package main

import (
	"os"
	"fmt"
	"context"
	"strings"
	"strconv"

	"golang.org/x/oauth2"
	"github.com/shurcooL/githubv4"
	"github.com/google/go-github/github"
)

type Issue struct {
	Title githubv4.String
	CreatedAt githubv4.String
	ID githubv4.String
	Number githubv4.Int
	Reactions struct {
		PageInfo struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
		Nodes []struct {
			Content githubv4.String
			User struct {
				ID githubv4.String
			}
		}
	} `graphql:"reactions(first: 10, after: $afterReact)"`
}

const (
	header = "<!-- START issue-notifier - please keep comment here to allow auto update -->\n" +
						"<!-- DON'T EDIT THIS SECTION, UPDATED AUTOMATICALLY BY ISSUE-NOTIFIER -->\n" +
						"## Current Issues\n"
	footer = "<!-- please keep comment here to allow auto update - END issue-notifier -->"
	startTicketRef = "<!-- START TICKET ref - issueID=\"idnull\" notified=\"notifiednull\" - place any relevant TICKET Link in between these comment sections -->"
	endTicketRef = "<!-- END TICKET ref -->"
	readmeFileName = "README.md"
)

var token, org, repoName string
var gqlClient *githubv4.Client
var ghClient *github.Client
var ctx context.Context
var smtpUser, smtpPass, smtpEndpoint, notifyAddress, notifyItemName, notifyEmailLogo string
var notifyEnabled bool
var smtpPort, notifyUpvotes int

func init() {
	var err error
	ctx = context.Background()
	gitRepo := os.Getenv("GITHUB_REPOSITORY")
	org = strings.Split(gitRepo, "/")[0]
	repoName = strings.Split(gitRepo, "/")[1]
	token = os.Getenv("GITHUB_TOKEN")
	if token == "" {
		panic("couldn't get github token")
	}

	smtpUser = os.Getenv("INPUT_SMTP_USER")
	smtpPass = os.Getenv("INPUT_SMTP_PASSWORD")
	smtpEndpoint = os.Getenv("INPUT_SMTP_ENDPOINT")
	port := os.Getenv("INPUT_SMTP_PORT")

	smtpPort, err = strconv.Atoi(port)
	if err != nil {
		panic("couldn't read SMTP_PORT as int")
	}

	notify := os.Getenv("INPUT_NOTIFY_ENABLED")
	if notify == "true" { notifyEnabled = true }

	if notifyEnabled {
		notifyAddress = os.Getenv("INPUT_NOTIFY_ADDRESS")
		notifyItemName = os.Getenv("INPUT_NOTIFY_ITEM_NAME")
		notifyEmailLogo = os.Getenv("INPUT_NOTIFY_EMAIL_LOGO")

		votes := os.Getenv("INPUT_NOTIFY_UPVOTES")
		notifyUpvotes, err = strconv.Atoi(votes)
		if err != nil {
			panic("couldn't read NOTIFY_UPVOTES as int")
		}
		fmt.Printf("notifications are enabled\n")
		fmt.Printf("number of upvotes required to notify: %d\n", notifyUpvotes)
		fmt.Printf("smtp_endpoint: %v:%d\n", smtpEndpoint, smtpPort)
		fmt.Printf("notification address: %v\n", notifyAddress)
		fmt.Printf("========================================\n\n")
	}

	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))	
	
	gqlClient = githubv4.NewClient(tc)

	ghClient = github.NewClient(tc)
}

func main() {
	issues, err := getIssues()
	if err != nil {
		panic(err)
	}

	sha, readme, err := getReadme()
	if err != nil {
		panic(err)
	}

	new, err := newReadme(string(readme), issues)
	if err != nil {
		panic(err)
	}

	if string(readme) == new {
		fmt.Printf("no changes made...done")
		return
	}
	fmt.Printf("found changes, committing updates directly to master...\n")
	
	opts := &github.RepositoryContentFileOptions{
		Message:   github.String("update table"),
		SHA:       &sha,
    Content:   []byte(new),
    Branch:    github.String("master"),
	}
	_, _, err = ghClient.Repositories.UpdateFile(ctx, org, repoName, readmeFileName, opts)
	if err != nil {
		panic(err)
	}
	fmt.Printf("done.\n")
}