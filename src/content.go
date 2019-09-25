package main

import (
	"fmt"

	"github.com/shurcooL/githubv4"
	"github.com/google/go-github/github"
)

func getIssues() ([]Issue, error) {
	var issues []Issue
	var err error
	hasNextPage := true
	var issueCursor *githubv4.String

	for hasNextPage {
		var reactCursor *githubv4.String
		var query struct {
			Organization struct {
				Repository struct {
					Issues struct {
						PageInfo struct {
							EndCursor   githubv4.String
							HasNextPage bool
						}
						Nodes []Issue
					} `graphql:"issues(first: 1, states: OPEN, after: $afterIssue)"`
				} `graphql:"repository(name: $repoName)"`
			} `graphql:"organization(login: $org)"`
		}
	
		variables := map[string]interface{}{
			"repoName": githubv4.String(repoName),
			"org" : githubv4.String(org),
			"afterIssue" : issueCursor,
			"afterReact": reactCursor,
		}
	
		if err := gqlClient.Query(ctx, &query, variables); err != nil {
			return issues, err
		}

		if len(query.Organization.Repository.Issues.Nodes) == 0 {
			return issues, err
		}
		
		issuePageInfo := query.Organization.Repository.Issues.PageInfo
		issueNode := query.Organization.Repository.Issues.Nodes[0]

		reactions := issueNode.Reactions.Nodes		
		reactPageInfo := issueNode.Reactions.PageInfo
		hasMoreReactions := reactPageInfo.HasNextPage
		
		for hasMoreReactions {
			reactCursor = githubv4.NewString(reactPageInfo.EndCursor)
			variables["afterReact"] = reactCursor

			if err := gqlClient.Query(ctx, &query, variables); err != nil {
				return issues, err
			}
			

			reactPageInfo = query.Organization.Repository.Issues.Nodes[0].Reactions.PageInfo
			reactNodes := query.Organization.Repository.Issues.Nodes[0].Reactions.Nodes

			reactions = append(reactions, reactNodes...)
			
			hasMoreReactions = reactPageInfo.HasNextPage
		}
		
		issueNode.Reactions.Nodes = reactions
		issues = append(issues, issueNode)

		hasNextPage = issuePageInfo.HasNextPage
		issueCursor = githubv4.NewString(issuePageInfo.EndCursor)
	}
	return issues, err
}

func getReadme() (string, []byte, error) {
	var sha, commit string
	var readme []byte
	var queryRef struct {
		Organization struct {
			Repository struct {
				Ref struct {
					Target struct {
						Oid githubv4.String
					}
				}`graphql:"ref(qualifiedName:\"refs/heads/master\")"`
			} `graphql:"repository(name: $repoName)"`
		} `graphql:"organization(login: $org)"`
	}

	variables := map[string]interface{}{
		"repoName": githubv4.String(repoName),
		"org" : githubv4.String(org),
	}

	if err := gqlClient.Query(ctx, &queryRef, variables); err != nil {
		return sha, readme, err
	}

	commit = string(queryRef.Organization.Repository.Ref.Target.Oid)
	
	tree, _, err := ghClient.Git.GetTree(ctx, org, repoName, commit, false)
	if err != nil {
		return sha, readme, err
	}

	sha, ok := makeFileToShaMap(tree)[readmeFileName]

	if !ok {
		return "", readme, fmt.Errorf("cannot find file https://github.com/%s/%s/commits/%s/%s", org, repoName, commit, readmeFileName)
	}

	readme, _, err = ghClient.Git.GetBlobRaw(ctx, org, repoName, sha)
	if err != nil {
		return sha, readme, err
	}

	return sha, readme, err
}

func makeFileToShaMap(tree *github.Tree) map[string]string {
	files := make(map[string]string)

	for _, entry := range tree.Entries {
		files[*entry.Path] = entry.GetSHA()
	}

	return files
}