/*
Copyright 2019 Cornelius Weig

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*****************************************************
 * NOTE The original version of this script is due to
 *    Balint Pato and was published as part of Skaffold
 *    (https://github.com/GoogleContainerTools/skaffold)
 *    under the following license:

Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*****************************************************/

// listpullreqs.go lists pull requests since the last release.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"syscall"

	"github.com/blang/semver"
	"github.com/google/go-github/v32/github"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const (
	sinceAny   = "any"
	sincePatch = "patch"
	sinceMinor = "minor"
	sinceMajor = "major"
)

var (
	org   string
	repo  string
	token string
	since string

	// versionMatchRE matches the raw version number from a string
	versionMatchRE = regexp.MustCompile(`^\s*v?(.*)$`)
)

const longDescription = `The script uses the GitHub API to retrieve a list of all merged pull
requests since the last release. The found pull requests are then
printed as markdown changelog with their commit summary and a link
to the pull request on GitHub.`

var rootCmd = &cobra.Command{
	Use:     "release-notes {org} {repo}",
	Example: "release-notes GoogleContainerTools skaffold",
	Short:   "Generate a markdown changelog of merged pull requests since last release",
	Long:    longDescription,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		org, repo = args[0], args[1]
		printChangeLog()
	},
}

func main() {
	rootCmd.Flags().StringVar(&token, "token", os.Getenv("GITHUB_TOKEN"), "Specify personal Github Token if you are hitting a rate limit anonymously. https://github.com/settings/tokens")
	rootCmd.Flags().StringVar(&since, "since", "patch", "The previous tag up to which PRs should be collected (one of any, patch, minor, major, or a valid semver) [defaults to 'patch']")
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func printChangeLog() {
	prs := getPullRequests()
	prMap := map[string][]*github.PullRequest{
		"release/new-feature": make([]*github.PullRequest, 0),
		"release/fixes":       make([]*github.PullRequest, 0),
		"release/refactor":    make([]*github.PullRequest, 0),
		"release/doc-updates": make([]*github.PullRequest, 0),
	}

	for _, pr := range prs {
		for _, l := range pr.Labels {
			if _, ok := prMap[*l.Name]; ok {
				prMap[*l.Name] = append(prMap[*l.Name], pr)
				break
			}
		}
	}

	arr := make([]string, 0)

	if len(prMap["release/new-feature"]) > 0 {
		arr = append(arr, "\nNew Features:"+formatSection(prMap["release/new-feature"]))
	}
	if len(prMap["release/fixes"]) > 0 {
		arr = append(arr, "\nFixes:"+formatSection(prMap["release/fixes"]))
	}
	if len(prMap["release/refactor"]) > 0 {
		arr = append(arr, "\nUpdates & Refactors:"+formatSection(prMap["release/refactor"]))
	}
	if len(prMap["release/doc-updates"]) > 0 {
		arr = append(arr, "\nDocs updates:"+formatSection(prMap["release/doc-updates"]))
	}

	changes := strings.Join(arr, "\n")
	fmt.Printf("%s\n\n", changes)
	printContributors(prs)
}

func printContributors(prs []*github.PullRequest) {
	set := make(map[string]string)
	ctx := contextWithCtrlCHandler()
	client := getClient(ctx)

	for _, pr := range prs {
		id := pr.User.Login
		if id == nil {
			continue
		}
		if _, ok := set[*id]; ok {
			continue
		}

		name := pr.User.Name
		if name == nil {
			if user, _, err := client.Users.Get(ctx, *id); err == nil {
				name = user.Name
			}
		}

		if name == nil {
			name = id
		}
		set[*id] = *name
	}

	names := make([]string, 0)
	for _, name := range set {
		names = append(names, fmt.Sprintf("- %v", name))
	}
	sort.Strings(names)
	fmt.Printf("Huge thanks goes out to all of our contributors for this release:\n%v\n", strings.Join(names, "\n"))
}

func formatSection(prs []*github.PullRequest) string {
	arr := make([]string, len(prs))
	for _, pr := range prs {
		arr = append(arr, formatPR(pr))
	}
	return strings.Join(arr, "\n")
}

func formatPR(pr *github.PullRequest) string {
	return fmt.Sprintf("* %s [#%d](https://github.com/%s/%s/pull/%d)\n", pr.GetTitle(), pr.GetNumber(), org, repo, pr.GetNumber())
}

func getPullRequests() []*github.PullRequest {
	ctx := contextWithCtrlCHandler()
	client := getClient(ctx)

	lastRelease, err := fetchLastRelease(ctx, client)
	if err != nil {
		logrus.Fatal(err)
	}
	lastReleaseTime := lastRelease.GetPublishedAt().Time
	result := make([]*github.PullRequest, 0)

	for page := 1; page != 0; {
		pullRequests, resp, err := client.PullRequests.List(ctx, org, repo, &github.PullRequestListOptions{
			State:     "closed",
			Sort:      "updated",
			Direction: "desc",
			ListOptions: github.ListOptions{
				PerPage: 20,
				Page:    page,
			},
		})
		if err != nil {
			logrus.Fatalf("Failed to list pull requests: %v", err)
		}
		page = resp.NextPage
		for idx := range pullRequests {
			pr := pullRequests[idx]
			if pr.GetUpdatedAt().Before(lastReleaseTime) {
				page = 0 // we are done now
				break
			}
			if pr.MergedAt != nil && pr.MergedAt.After(lastReleaseTime) {
				result = append(result, pr)
			}
		}
	}
	return result
}

func fetchLastRelease(ctx context.Context, client *github.Client) (*github.RepositoryRelease, error) {
	releases, _, err := client.Repositories.ListReleases(ctx, org, repo, &github.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list releases")
	}
	matches, err := toVersionMatcher(since)
	if err != nil {
		return nil, err
	}
	for _, release := range releases {
		version, err := parseSemver(release.GetTagName())
		if err != nil {
			return nil, err
		}
		if matches(version) {
			return release, nil
		}
	}
	return nil, fmt.Errorf("no previous release found tag %s/%s", org, repo)
}

func toVersionMatcher(since string) (func(semver.Version) bool, error) {
	// magic version specifiers
	switch since {
	case sinceAny:
		return func(_ semver.Version) bool { return true }, nil
	case sincePatch:
		return func(v semver.Version) bool { return len(v.Pre) == 0 }, nil
	case sinceMinor:
		return func(v semver.Version) bool { return v.Patch == 0 && len(v.Pre) == 0 }, nil
	case sinceMajor:
		return func(v semver.Version) bool { return v.Minor == 0 && v.Patch == 0 && len(v.Pre) == 0 }, nil
	}

	previousVersion, err := parseSemver(since)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse semver %q", since)
	}

	return previousVersion.GTE, nil
}

func parseSemver(tagName string) (semver.Version, error) {
	parts := versionMatchRE.FindStringSubmatch(tagName)
	if parts == nil {
		return semver.Version{}, fmt.Errorf("%q does not look like a version string", tagName)
	}

	version, err := semver.Parse(parts[1])
	return version, errors.Wrapf(err, "could not parse as semver")
}

func getClient(ctx context.Context) *github.Client {
	if len(token) == 0 {
		return github.NewClient(nil)
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func contextWithCtrlCHandler() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGPIPE)

	go func() {
		<-sigs
		signal.Stop(sigs)
		cancel()
		logrus.Infof("Aborted.")
	}()

	return ctx
}
