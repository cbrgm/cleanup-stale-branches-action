package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/google/go-github/v79/github"
	"golang.org/x/oauth2"
)

var args struct {
	AllowedPrefixes     string `arg:"--allowed-prefixes,env:ALLOWED_PREFIXES"`
	DryRun              bool   `arg:"--dry-run,env:DRY_RUN"`
	GithubRepo          string `arg:"--github-repo,required,env:GITHUB_REPOSITORY"`
	GithubToken         string `arg:"--github-token,required,env:GITHUB_TOKEN"`
	IgnoreBranches      string `arg:"--ignore-branches,env:IGNORE_BRANCHES"`
	IgnoredPrefixes     string `arg:"--ignored-prefixes,env:IGNORED_PREFIXES"`
	LastCommitAgeDays   int    `arg:"--last-commit-age-days,env:LAST_COMMIT_AGE_DAYS"`
	RateLimit           bool   `arg:"--rate-limit,env:RATE_LIMIT"`
	GitHubEnterpriseUrl string `arg:"--github-enterprise-url,env:GITHUB_ENTERPRISE_URL"`
}

type GitHubClientWrapper struct {
	client GitHubAPI
	owner  string
	repo   string
	ctx    context.Context
}

func main() {
	arg.MustParse(&args)

	if !isValidRepoFormat(args.GithubRepo) {
		fmt.Println("Invalid repository format provided, must be in format owner/repository")
		os.Exit(1)
	}

	ctx := context.Background()
	githubClient := NewGitHubClientWrapper(ctx, args.GithubToken, parseRepoOwner(args.GithubRepo), parseRepoName(args.GithubRepo), args.RateLimit, args.GitHubEnterpriseUrl)

	deletableBranches, err := githubClient.getDeletableBranches(ctx)
	if err != nil {
		fmt.Printf("Error fetching branches: %v\n", err)
		os.Exit(1)
	}

	if args.DryRun {
		log.Printf("Dry-run enabled, %d branches would have been deleted...\n", len(deletableBranches))
		for _, branch := range deletableBranches {
			log.Printf("- %s\n", branch)
		}
	} else {
		log.Printf("Dry-run DISABLED, will delete %d branches now...\n", len(deletableBranches))
		err = githubClient.deleteBranches(ctx, deletableBranches)
		if err != nil {
			log.Printf("Error deleting branches: %v\n", err)
			os.Exit(1)
		}
	}
}

// isValidRepoFormat checks if the repository name follows the 'owner/repository' format.
func isValidRepoFormat(repoName string) bool {
	if !isValidRepoNameFormat(repoName) {
		fmt.Printf("Repository name is in the wrong format. Expected 'owner/repository'\n")
		return false
	}
	return true
}

// isValidRepoNameFormat checks if a given repository name is in the 'owner/repository' format.
func isValidRepoNameFormat(repoName string) bool {
	parts := strings.Split(repoName, "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

// parseRepoOwner extracts the repository owner from the full repository name.
func parseRepoOwner(repoName string) string {
	parts := strings.Split(repoName, "/")
	return parts[0]
}

// parseRepoName extracts the repository name from the full repository name.
func parseRepoName(repoName string) string {
	parts := strings.Split(repoName, "/")
	if len(parts) > 1 {
		return parts[1]
	}
	return repoName
}

func NewGitHubClientWrapper(ctx context.Context, token, owner, repo string, rateLimitCheckEnabled bool, gitHubEnterpriseUrl string) *GitHubClientWrapper {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	if gitHubEnterpriseUrl != "" {
		var err error
		ghClient, err = ghClient.WithEnterpriseURLs(gitHubEnterpriseUrl, gitHubEnterpriseUrl)
		if err != nil {
			log.Printf("Failed to set GitHub Enterprise URLs: %v", err)
			return nil
		}
	}

	apiClient := NewGitHubAPI(ghClient, rateLimitCheckEnabled)

	return &GitHubClientWrapper{
		client: apiClient,
		owner:  owner,
		repo:   repo,
		ctx:    ctx,
	}
}

func (g *GitHubClientWrapper) getDeletableBranches(ctx context.Context) ([]string, error) {
	defaultBranch, err := g.getDefaultBranch(ctx)
	if err != nil {
		log.Printf("Unable to get default branch: %v\n", err)
		return nil, err
	}

	openPullRequests, err := g.getAllOpenPullRequests(ctx)
	if err != nil {
		log.Printf("Error fetching open pull requests: %v\n", err)
		return nil, err
	}

	ignoreBranches := splitNonEmpty(args.IgnoreBranches)
	allowedPrefixes := splitNonEmpty(args.AllowedPrefixes)
	ignoredPrefixes := splitNonEmpty(args.IgnoredPrefixes)

	deletableBranches := []string{}

	opts := &github.BranchListOptions{
		// todo(cbrgm): decide whether it makes sense to only query unprotected branches..
		// Protected:   github.Bool(false),
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		branches, resp, err := g.client.ListBranches(ctx, g.owner, g.repo, opts)
		if err != nil {
			log.Printf("Error listing branches: %v\n", err)
			break
		}

		log.Printf("Checking %d branches...\n", len(branches))
		for _, branch := range branches {
			branchName := branch.GetName()

			if branchName == defaultBranch {
				log.Printf("- Skipping `%s`: it is the default branch\n", branchName)
				continue
			}

			if branch.GetProtected() {
				log.Printf("- Skipping `%s`: it is a protected branch\n", branchName)
				continue
			}

			if contains(ignoreBranches, branchName) {
				log.Printf("- Skipping `%s`: it is in the list of ignored branches\n", branchName)
				continue
			}

			if len(allowedPrefixes) > 0 && !startsWith(allowedPrefixes, branchName) {
				log.Printf("- Skipping `%s`: does not match allowed prefixes\n", branchName)
				continue
			}

			if len(ignoredPrefixes) > 0 && startsWith(ignoredPrefixes, branchName) {
				log.Printf("- Skipping `%s`: does match ignored prefixes\n", branchName)
				continue
			}

			commitDate, err := g.getLatestCommitDate(ctx, branchName)
			if err != nil {
				log.Printf("- Skipping `%s`: %v\n", branchName, err)
				continue
			}

			if time.Since(commitDate).Hours() < float64(args.LastCommitAgeDays*24) {
				log.Printf("- Skipping `%s`: last commit is newer than %d days\n", branchName, args.LastCommitAgeDays)
				continue
			}

			if isBranchInOpenPullRequests(branchName, openPullRequests) {
				log.Printf("- Skipping `%s`: has open pull requests\n", branchName)
				continue
			}

			basePulls, err := g.isPullRequestBase(ctx, branchName)
			if err != nil {
				log.Printf("- Skipping `%s`: error checking if branch is a base for a pull request - %v\n", branchName, err)
				continue
			}

			if basePulls {
				log.Printf("- Skipping `%s`: is the base for a pull request\n", branchName)
				continue
			}

			deletableBranches = append(deletableBranches, branchName)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return deletableBranches, nil
}

func (g *GitHubClientWrapper) getLatestCommitDate(ctx context.Context, branchName string) (time.Time, error) {
	commitOpts := &github.CommitsListOptions{
		SHA:         branchName,
		ListOptions: github.ListOptions{PerPage: 1},
	}

	commits, _, err := g.client.ListCommits(ctx, g.owner, g.repo, commitOpts)
	if err != nil {
		return time.Time{}, err
	}

	if len(commits) == 0 {
		return time.Time{}, fmt.Errorf("no commits found in branch %s", branchName)
	}

	lastCommit := commits[0]
	commitDate := lastCommit.GetCommit().GetCommitter().GetDate()
	return *commitDate.GetTime(), nil
}

// isPullRequestBase checks if the branch is the base for any pull request.
func (g *GitHubClientWrapper) isPullRequestBase(ctx context.Context, branchName string) (bool, error) {
	pulls, _, err := g.client.ListPullRequests(ctx, g.owner, g.repo, &github.PullRequestListOptions{
		State: "open",
		Base:  branchName,
	})
	if err != nil {
		return false, err
	}
	return len(pulls) > 0, nil
}

func (g *GitHubClientWrapper) getDefaultBranch(ctx context.Context) (string, error) {
	repo, _, err := g.client.GetRepository(ctx, g.owner, g.repo)
	if err != nil {
		return "", err
	}
	return repo.GetDefaultBranch(), nil
}

// isBranchInOpenPullRequests checks if a branch is associated with any open pull requests.
func isBranchInOpenPullRequests(branchName string, pullRequests []*github.PullRequest) bool {
	for _, pr := range pullRequests {
		if pr.Head.GetRef() == branchName {
			return true
		}
	}
	return false
}

func (g *GitHubClientWrapper) getAllOpenPullRequests(ctx context.Context) ([]*github.PullRequest, error) {
	var allPullRequests []*github.PullRequest
	opts := &github.PullRequestListOptions{
		State:       "open",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		pullRequests, resp, err := g.client.ListPullRequests(ctx, g.owner, g.repo, opts)
		if err != nil {
			return nil, err
		}
		allPullRequests = append(allPullRequests, pullRequests...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allPullRequests, nil
}

func (g *GitHubClientWrapper) deleteBranches(ctx context.Context, branches []string) error {
	for _, branch := range branches {
		_, err := g.client.DeleteRef(ctx, g.owner, g.repo, "refs/heads/"+branch)
		if err != nil {
			return err
		}
		log.Printf("- Branch '%s' successfully deleted.\n", branch)
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, pattern := range slice {
		// Replace * with .*, which is the regex equivalent
		regexPattern := "^" + regexp.QuoteMeta(pattern)
		regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*") + "$"

		matched, _ := regexp.MatchString(regexPattern, item)
		if matched {
			return true
		}
	}
	return false
}

func startsWith(prefixes []string, str string) bool {
	if len(prefixes) == 0 {
		return false
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}
	return false
}

func splitNonEmpty(input string) []string {
	if input == "" {
		return []string{}
	}
	return strings.Split(input, ",")
}
