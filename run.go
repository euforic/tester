package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

type CatalogInfo struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

type GitHubOps struct {
	Token string
}

func NewGitHubOps(token string) *GitHubOps {
	return &GitHubOps{Token: token}
}

func (g *GitHubOps) CreateBranchAndAddFile(repoPath string, newBranch string, filePath string, destinationPath string) (*git.Repository, error) {
	// Clone the repository
	fmt.Printf("git clone %s\n", repoPath)
	// Extract the repo name from the repoPath
	repoName := strings.Split(repoPath, "/")[len(strings.Split(repoPath, "/"))-1]
	repoName = strings.Replace(repoName, ".git", "", -1)

	if err := os.RemoveAll("/tmp/" + repoName); err != nil {
		return nil, err
	}

	r, err := git.PlainClone("/tmp/"+repoName, false, &git.CloneOptions{
		URL:      repoPath,
		Progress: os.Stdout,
		Auth: &http.BasicAuth{
			Username: "git",
			Password: g.Token,
		},
	})
	if err != nil {
		return nil, errors.New("failed to clone repository: " + err.Error())
	}

	// Create a new branch and checkout
	fmt.Printf("git checkout -b %s\n", newBranch)

	headRef, err := r.Head()
	if err != nil {
		return nil, err
	}

	// Create a new plumbing.HashReference object with the name of the branch
	refName := plumbing.ReferenceName("refs/heads/" + newBranch)
	newRef := plumbing.NewHashReference(refName, headRef.Hash())

	// The created reference is saved in the storage.
	err = r.Storer.SetReference(newRef)
	if err != nil {
		return nil, err
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	// Checkout to the new branch
	err = w.Checkout(&git.CheckoutOptions{
		Branch: refName,
	})
	if err != nil {
		return nil, err
	}

	// Copy the file
	err = os.Rename(filePath, path.Join("/tmp/"+repoName, destinationPath))
	if err != nil {
		return nil, err
	}

	catalogInfo := CatalogInfo{
		APIVersion: "catalog.cattle.io/v1",
		Kind:       "Catalog",
	}

	yaml, err := yaml.Marshal(catalogInfo)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(path.Join("/tmp/"+repoName, "catalog-info.yaml"), yaml, 0644); err != nil {
		return nil, err
	}

	// Add the new file to the staging area.
	fmt.Println("git add .")
	_, err = w.Add(".")
	if err != nil {
		return nil, err
	}

	// Commit the changes
	fmt.Println("git commit -m 'Add new file'")
	_, err = w.Commit("Add new file", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Your Name",
			Email: "you@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, err
	}

	// Push the changes to the repository
	fmt.Println("git push")
	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth: &http.BasicAuth{
			Username: "Your GitHub username", // this can be anything except an empty string
			Password: g.Token,
		},
		Progress: os.Stdout,
	})
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (g *GitHubOps) CreatePR(owner string, repo string, newBranch string, title string, body string) error {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: g.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// Create a new pull request
	newPR := &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(newBranch), // this is the branch with changes
		Base:  github.String("main"),    // this is the branch you want to merge into
		Body:  github.String(body),
	}

	pr, _, err := client.PullRequests.Create(ctx, owner, repo, newPR)
	if err != nil {
		return err
	}

	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
	return nil
}

func main() {
	g := NewGitHubOps(os.Getenv("GITHUB_TOKEN"))
	_, err := g.CreateBranchAndAddFile("https://github.com/euforic/tester", "add-catalog-info", "run.go", "run.go")
	if err != nil {
		fmt.Println(err)
		return
	}

	err = g.CreatePR("euforic", "tester", "add-catalog-info", "Add catalog-info.yaml", "Add catalog-info.yaml")
	if err != nil {
		fmt.Println(err)
		return
	}
}
