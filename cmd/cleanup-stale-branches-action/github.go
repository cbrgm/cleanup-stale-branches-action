package main

import (
	"context"
	"log"
	"os"

	"github.com/google/go-github/v76/github"
)

type GitHubAPI interface {
	ListBranches(ctx context.Context, owner, repo string, opts *github.BranchListOptions) ([]*github.Branch, *github.Response, error)
	ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error)
	GetRepository(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error)
	ListPullRequests(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	DeleteRef(ctx context.Context, owner, repo, ref string) (*github.Response, error)
	RateLimits(ctx context.Context) (*github.RateLimits, *github.Response, error)
}

const rateLimitedMessage = "GitHub API rate limit close to being exceeded. Stopping execution."

type gitHubAPI struct {
	client                *github.Client
	rateLimitCheckEnabled bool
}

func NewGitHubAPI(client *github.Client, rateLimitCheckEnabled bool) GitHubAPI {
	return &gitHubAPI{
		client:                client,
		rateLimitCheckEnabled: rateLimitCheckEnabled,
	}
}

func (g *gitHubAPI) ListBranches(ctx context.Context, owner, repo string, opts *github.BranchListOptions) ([]*github.Branch, *github.Response, error) {
	if g.rateLimitCheckEnabled && g.isRateLimitExceeded(ctx) {
		log.Println(rateLimitedMessage)
		os.Exit(0)
	}
	return g.client.Repositories.ListBranches(ctx, owner, repo, opts)
}

func (g *gitHubAPI) ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error) {
	if g.rateLimitCheckEnabled && g.isRateLimitExceeded(ctx) {
		log.Println(rateLimitedMessage)
		os.Exit(0)
	}
	return g.client.Repositories.ListCommits(ctx, owner, repo, opts)
}

func (g *gitHubAPI) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error) {
	if g.rateLimitCheckEnabled && g.isRateLimitExceeded(ctx) {
		log.Println(rateLimitedMessage)
		os.Exit(0)
	}
	return g.client.Repositories.Get(ctx, owner, repo)
}

func (g *gitHubAPI) ListPullRequests(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	if g.rateLimitCheckEnabled && g.isRateLimitExceeded(ctx) {
		log.Println(rateLimitedMessage)
		os.Exit(0)
	}
	return g.client.PullRequests.List(ctx, owner, repo, opts)
}

func (g *gitHubAPI) DeleteRef(ctx context.Context, owner, repo, ref string) (*github.Response, error) {
	if g.rateLimitCheckEnabled && g.isRateLimitExceeded(ctx) {
		log.Println(rateLimitedMessage)
		os.Exit(0)
	}
	return g.client.Git.DeleteRef(ctx, owner, repo, ref)
}

func (g *gitHubAPI) RateLimits(ctx context.Context) (*github.RateLimits, *github.Response, error) {
	if g.rateLimitCheckEnabled && g.isRateLimitExceeded(ctx) {
		log.Println(rateLimitedMessage)
		os.Exit(0)
	}
	return g.client.RateLimit.Get(ctx)
}

func (g *gitHubAPI) isRateLimitExceeded(ctx context.Context) bool {
	rateLimitStatus, _, err := g.client.RateLimit.Get(ctx)
	if err != nil {
		log.Printf("Error fetching rate limit status: %v\n", err)
		return false
	}

	limit := rateLimitStatus.Core.Limit
	remaining := rateLimitStatus.Core.Remaining

	return float64(remaining)/float64(limit) <= 0.05
}
