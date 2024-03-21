package repo

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/google/go-github/v59/github"
)

// CheckLicenses checks all repos for licenses that need to be managed/updated
func (c *Content) CheckLicenses() error {
	var reposToCheck []string

	opts := &github.ListOptions{
		Page:    0,
		PerPage: 100,
	}
	for {
		opts.Page++
		result, resp, err := c.githubClient.Organizations.ListCustomPropertyValues(context.TODO(), GitHubOrg, opts)
		if err != nil {
			return err
		}

		for _, repo := range result {
			for _, property := range repo.Properties {
				if property.PropertyName == "manage-license" && property.Value != nil && *property.Value == "yes" {
					reposToCheck = append(reposToCheck, repo.RepositoryName)
				}
			}
		}

		if resp.NextPage == 0 {
			break
		}
	}

	for _, repo := range reposToCheck {
		log.Printf("Need to check %s\n", repo)
		err := c.UpdateLicense(repo)
		if err != nil {
			log.Printf("Error updating %s: %s\n", repo, err.Error())
			continue
		}
	}

	return nil
}

// UpdateLicense ensures the license is up to date for the given repo
func (c *Content) UpdateLicense(repoName string) error {
	defer removeDirIfExists(repoDir(repoName))

	r, w, err := c.cloneRepo(repoName)
	if err != nil {
		return err
	}

	repoConfig, err := c.LoadRepoConfig(repoDir(repoName))
	if err != nil {
		log.Printf("Error loading config for %s: %v\n", repoName, err)
	}

	// If we are targeting a different branch with PRs, then our base also needs to start from that branch
	if repoConfig.PrTargetBranch != nil {
		err = c.checkoutBranch(r, w, *repoConfig.PrTargetBranch)
		if err != nil {
			return err
		}
	}

	file, err := os.ReadFile(path.Join(c.templates, "LICENSE"))
	if err != nil {
		return err
	}
	content, err := ProcessTemplate(file, repoConfig.VarOverrides)
	if err != nil {
		return err
	}

	branchName := "update-license"
	err = c.createBranch(r, w, branchName)
	if err != nil {
		return err
	}

	// To be more consistent, we delete alternate forms of the LICENSE first
	// then write the LICENSE file
	// If similar enough, the commit should see a rename with minor changes
	alternateForms := []string{"LICENSE_APACHE", "LICENSE.txt", "LICENSE.md", "license-apache", "License"}
	for _, form := range alternateForms {
		// Ignoring errors since these files may not exist
		_ = os.Remove(fmt.Sprintf("%s/%s", repoDir(repoName), form))
	}

	err = os.WriteFile(fmt.Sprintf("%s/LICENSE", repoDir(repoName)), content, 0644)
	if err != nil {
		return err
	}

	// Stage the changes
	_, err = w.Add("LICENSE")
	if err != nil {
		return err
	}

	status, err := w.Status()
	if err != nil {
		return err
	}

	if status.IsClean() {
		return nil
	}

	var message string
	if repoConfig.CommitPrefix != nil {
		message = fmt.Sprintf("%s Update license", *repoConfig.CommitPrefix)
	} else {
		// Handle the case where CommitPrefix is nil
		// For example, use a default message
		message = "Update license"
	}
	err = c.commit(w, repoName, message)
	if err != nil {
		return err
	}

	repo, _, err := c.githubClient.Repositories.Get(context.TODO(), GitHubOrg, repoName)
	if err != nil {
		return fmt.Errorf("error getting repo info: %w", err)
	}

	var DefaultBranch string
	if repoConfig.PrTargetBranch == nil || *repoConfig.PrTargetBranch == "" {
		DefaultBranch = *repo.DefaultBranch
	} else {
		DefaultBranch = *repoConfig.PrTargetBranch
	}

	return c.pushAndPR(r, repoName, branchName, "Updated License", &pushAndPROptions{
		PrTargetBranch: &DefaultBranch,         // Directly using the pointer from repoConfig
		AssignUsers:    repoConfig.AssignUsers, // Assuming AssignUsers is a slice of strings
		AssignGroup:    repoConfig.AssignGroup, // Directly using the pointer from repoConfig
	})
}
