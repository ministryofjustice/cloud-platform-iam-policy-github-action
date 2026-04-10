package main

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"iam-role-policy-changes-check/identifyiam"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-github/v84/github"
)

func NewGitHubAppClient() (*github.Client, error) {
	appID, err := strconv.ParseInt(os.Getenv("GITHUB_APP_ID"), 10, 64)
	if err != nil {
		return nil, err
	}
	installationID, err := strconv.ParseInt(os.Getenv("GITHUB_APP_INSTALLATION_ID"), 10, 64)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode([]byte(os.Getenv("GITHUB_APP_PRIVATE_KEY")))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block from private key")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),
		Issuer:    strconv.FormatInt(appID, 10),
	})
	jwtStr, err := token.SignedString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign JWT: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+jwtStr)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get installation token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode installation token response: %w", err)
	}
	if result.Token == "" {
		return nil, fmt.Errorf("empty installation token received")
	}

	return github.NewClient(nil).WithAuthToken(result.Token), nil
}

func main() {
	flag.Parse()
	// fileName is the file created by a GitHub action, it contains the output of a git diff.
	fileName := "changes"
	// prRelevant will return true or false depending on the contents of fileName. We don't want
	// the GH action to error here so we just log the error and take no action.
	prRelevant, err := identifyiam.ParsePR(fileName)
	if err != nil {
		log.Println("Unable to parse the PR - ", err)
	}

	// Conditional check to see if we should pass or fail the step. We don't want a hard fail so we set
	// the output to false and log.
	// If the PR is relevant we want to request a review from the cloud platform team, if not we just log that the check has passed.

	client, err := NewGitHubAppClient()
	if err != nil {
		log.Fatalf("Failed to create GitHub App client: %v", err)
	}
	ctx := context.Background()

	if !prRelevant {
		log.Println("Fail: Attention - Either the PR contains changes that potentially relate to IAM roles or IAM Policies .")
		review := &github.PullRequestReviewRequest{
			Event: github.Ptr("REQUEST_CHANGES"),
			Body:  github.Ptr("There are potential IAM Role and/or policy Changes/additions. Reviewer - If satisfied with the changes/additions - dismiss this request"),
		}
		prNumber, err := strconv.Atoi(os.Getenv("PR_NUMBER"))
		if err != nil {
			log.Fatalf("Invalid pull request number: %v", err)
		}

		parts := strings.SplitN(os.Getenv("GITHUB_REPOSITORY"), "/", 2)

		_, _, err = client.PullRequests.CreateReview(ctx, parts[0], parts[1], prNumber, review)
		if err != nil {
			log.Fatalf("Error creating review: %v", err)
		}
	} else {
		log.Println("Success: The changes in this PR are not related IAM roles/Policies.")
	}
}
