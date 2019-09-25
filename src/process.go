package main

import (
	"fmt"
	"regexp"
	"sort"
	"time"
	"strconv"
	"strings"

	"github.com/shurcooL/githubv4"
)

type IssueState struct {
	Ref string
	Notified bool
}

func newReadme(readme string, issues []Issue) (string, error) {
	var err error

	regTable := regexp.MustCompile(`<!-- START issue-notifier([\s\S.\n]*?)END issue-notifier -->`)
	matches := regTable.FindAllStringSubmatch(string(readme), -1)
	state, err := tableState(matches)
	if err != nil {
		return "", err
	}

	if notifyEnabled {
		err = notify(issues, &state)
		if err != nil {
			return "", err
		}
	}

	table, err := issuesTable(issues, state)
	if err != nil {
		return "", err
	}

	if len(matches) != 1 {
		updated := regTable.ReplaceAllString(string(readme), "") + "\n\n" + table
		return updated, nil
	}

	return regTable.ReplaceAllString(string(readme), table), nil
}

func tableState(matches [][]string) (map[string]IssueState, error) {
	var notified = false
	state := make(map[string]IssueState)

	if len(matches) != 1 { return state, nil }
	table := matches[0][0]
	
	regRef := regexp.MustCompile(`<!-- START TICKET ref - issueID=\"(.*?)\" notified=\"(.*?)\"[\s\S]*?-->(.*?)<!--[\s\S]*?END TICKET ref -->`)
	issueMatches := regRef.FindAllStringSubmatch(table, -1)

	for _, m := range issueMatches {
		if m[2] == "true" { notified = true }
		state[m[1]] = IssueState{
			Ref: m[3],
			Notified: notified,
		}
	}
	return state, nil
}

func (i *Issue) numUpvotes() int {
	unique := make(map[githubv4.String]int)
	for _, r := range i.Reactions.Nodes {
		if r.Content == "HOORAY" || r.Content == "THUMBS_UP" || r.Content == "HEART" {
			unique[r.User.ID] = 0
		}
	}
	return len(unique)
}

func issuesTable(issues []Issue, state map[string]IssueState) (string, error) {
	var table string

	if len(issues) == 0 {
		table += header
		table += "There are currently no open issues..."
		table += footer
		return table, nil
	}

	sort.Slice(issues, func(i, j int) bool {
		return issues[i].numUpvotes() > issues[j].numUpvotes()
	})
	table += header
	table += "Automatically maintained table showing the currently open issues:\n\n\n"
	table += "| Created | Title | Upvotes | Ticket Ref |\n"
	table += "|---------|-------|---------|------------|\n"
	for _, issue := range issues {
		var ref, ticketRef string
		t, err := time.Parse(time.RFC3339, string(issue.CreatedAt))
		if err != nil {
			return "", err
		}
		upvotes := issue.numUpvotes()
		ticketRef = strings.Replace(startTicketRef, "idnull", string(issue.ID), 1)
		link := fmt.Sprintf("https://github.com/%v/%v/issues/%d", org, repoName, issue.Number)
		if r, ok := state[fmt.Sprintf("%v", issue.ID)]; ok {
			ref = r.Ref
			ticketRef = strings.Replace(ticketRef, "notifiednull", strconv.FormatBool(r.Notified), 1)
		}
		table += fmt.Sprintf("| %v | [%v](%v) | %d | %v%v%v |\n", t.Format("2-1-2006"), issue.Title, link, upvotes, ticketRef, ref, endTicketRef)
	}
	table += footer
	return table, nil
}