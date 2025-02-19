/*
Copyright (c) 2020 Loadsmart, Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package coveralls

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

var (
	// ErrRepoNotFound is returned when we receive a 404 Not Found status code
	ErrRepoNotFound = fmt.Errorf("repo was not found (status code %d)", http.StatusNotFound)

	// ErrNameIsTaken is returned when the API respondes to a POST saying that the repo
	// name has already been taken. The status code is UnprocessableEntity but we return
	// this more specific error for convenience.
	ErrNameIsTaken = fmt.Errorf("unprocessable entity (status code %d): repo name has already been taken", http.StatusUnprocessableEntity)
)

// ErrUnprocessableEntity is returned when the API returns 422 Unprocessable
// Entity status code and we don't have identified a more specific error condition.
//
// Its error message string includes the full body from the response.
// That includes some error in the RepositoryConfig spec, but may include other conditions.
type ErrUnprocessableEntity struct {
	ErrorBody string
}

func (e ErrUnprocessableEntity) Error() string {
	return fmt.Sprintf("unprocessable entity (status code %d). Error body: '%s'", http.StatusUnprocessableEntity, e.ErrorBody)
}

func newErrUnprocessableEntity(errorBody string) ErrUnprocessableEntity {
	return ErrUnprocessableEntity{
		ErrorBody: errorBody,
	}
}

// ErrUnexpectedStatusCode is returned when we receive an unexpected status code, not
// covered by our other sentinel errors.
type ErrUnexpectedStatusCode struct {
	StatusCode int
	ErrorBody  string
}

func (e ErrUnexpectedStatusCode) Error() string {
	return fmt.Sprintf("super unexpected status code %d. Error body: '%s'", e.StatusCode, e.ErrorBody)
}

func newErrUnexpectedStatusCode(c int, b string) ErrUnexpectedStatusCode {
	return ErrUnexpectedStatusCode{
		StatusCode: c,
		ErrorBody:  b,
	}
}

// RepositoryService holds information to access repository-related endpoints
type RepositoryService interface {
	Get(ctx context.Context, svc string, repo string) (*Repository, error)
	Add(ctx context.Context, data *RepositoryConfig) (*RepositoryConfig, error)
	Update(ctx context.Context, svc string, repo string, data *RepositoryConfig) (*RepositoryConfig, error)
}

// RepositoryServiceImpl holds information to access repository-related endpoints
type RepositoryServiceImpl service

// Repository holds information about one specific repository
type Repository struct {
	ID                              int      `json:"id,omitempty"`
	Name                            string   `json:"name,omitempty"`
	Service                         string   `json:"service,omitempty"`                             // Git provider. Options include: github, bitbucket, gitlab, stash, manual
	CommentOnPullRequests           *bool    `json:"comment_on_pull_requests,omitempty"`            // Whether comments should be posted on pull requests (defaults to true)
	SendBuildStatus                 *bool    `json:"send_build_status,omitempty"`                   // Whether build status should be sent to the git provider (defaults to true)
	CommitStatusFailThreshold       *float64 `json:"commit_status_fail_threshold,omitempty"`        // Minimum coverage that must be present on a build for the build to pass (default is null, meaning any decrease is a failure)
	CommitStatusFailChangeThreshold *float64 `json:"commit_status_fail_change_threshold,omitempty"` // If coverage decreases, the maximum allowed amount of decrease that will be allowed for the build to pass (default is null, meaning that any decrease is a failure)
	HasBadge                        bool     `json:"has_badge,omitempty"`
	Token                           string   `json:"token,omitempty"`
	CreatedAt                       string   `json:"created_at,omitempty"`
	UpdatedAt                       string   `json:"updated_at,omitempty"`
}

// RepositoryConfig represents config settings for a given repository
type RepositoryConfig struct {
	Service                         string   `json:"service"`                                       // Git provider. Options include: github, bitbucket, gitlab, stash, manual
	Name                            string   `json:"name"`                                          // Name of the repo. E.g. with Github, this is username/reponame.
	CommentOnPullRequests           *bool    `json:"comment_on_pull_requests,omitempty"`            // Whether comments should be posted on pull requests (defaults to true)
	SendBuildStatus                 *bool    `json:"send_build_status,omitempty"`                   // Whether build status should be sent to the git provider (defaults to true)
	CommitStatusFailThreshold       *float64 `json:"commit_status_fail_threshold,omitempty"`        // Minimum coverage that must be present on a build for the build to pass (default is null, meaning any decrease is a failure)
	CommitStatusFailChangeThreshold *float64 `json:"commit_status_fail_change_threshold,omitempty"` // If coverage decreases, the maximum allowed amount of decrease that will be allowed for the build to pass (default is null, meaning that any decrease is a failure)
}

// Get information about a repository already in Coveralls.
//
// Ctx is a context that's propagated to underlying client. You can use
// it to cancel the request in progress (when the program is terminated,
// for example).
//
// Svc indicates the service. Any value accepted by Coveralls API can be
// passed here. Soma valid inputs include 'github', 'bitbucket' or 'manual'.
//
// Repo is the repository name. In GitHub, for example, this is
// 'organization/repository'; other services could have different formats.
//
// If the request succeeded, it returns a Repository with the information
// available or an error if there was something wrong.
//
// It may return errors ErrRepoNotFound or ErrUnexpectedStatusCode
func (s RepositoryServiceImpl) Get(ctx context.Context, svc string, repo string) (*Repository, error) {
	url := fmt.Sprintf("%s/api/repos/%s/%s", s.client.HostURL, svc, repo)

	resp, err := s.client.client.R().
		SetContext(ctx).
		SetResult(&Repository{}).
		Get(url)

	if err != nil {
		return nil, err
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return resp.Result().(*Repository), nil
	case http.StatusNotFound:
		return nil, ErrRepoNotFound
	default:
		return nil, newErrUnexpectedStatusCode(resp.StatusCode(), string(resp.Body()))
	}
}

// Add a repository to Coveralls
//
// It may return errors ErrNameIsTaken, ErrUnprocessableEntity or ErrUnexpectedStatusCode
func (s RepositoryServiceImpl) Add(ctx context.Context, data *RepositoryConfig) (*RepositoryConfig, error) {
	url := fmt.Sprintf("%s/api/repos", s.client.HostURL)

	body := map[string]*RepositoryConfig{
		"repo": data,
	}

	resp, err := s.client.client.R().
		SetContext(ctx).
		SetBody(body).
		SetResult(&RepositoryConfig{}).
		Post(url)

	if err != nil {
		return nil, err
	}

	switch resp.StatusCode() {
	case http.StatusCreated:
		return resp.Result().(*RepositoryConfig), nil
	case http.StatusUnprocessableEntity:
		errorBody := string(resp.Body())
		if strings.Contains(errorBody, "has already been taken") {
			return nil, ErrNameIsTaken
		}
		return nil, newErrUnprocessableEntity(errorBody)
	default:
		return nil, newErrUnexpectedStatusCode(resp.StatusCode(), string(resp.Body()))
	}

}

// Update repository configuration in Coveralls
//
// It may return errors ErrRepoNotFound, ErrUnprocessableEntity or ErrUnexpectedStatusCode
func (s RepositoryServiceImpl) Update(ctx context.Context, svc string, repo string, data *RepositoryConfig) (*RepositoryConfig, error) {
	url := fmt.Sprintf("%s/api/repos/%s/%s", s.client.HostURL, svc, repo)

	body := map[string]*RepositoryConfig{
		"repo": data,
	}

	resp, err := s.client.client.R().
		SetContext(ctx).
		SetBody(body).
		SetResult(&RepositoryConfig{}).
		Put(url)

	if err != nil {
		return nil, err
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return resp.Result().(*RepositoryConfig), nil
	case http.StatusNotFound:
		return nil, ErrRepoNotFound
	case http.StatusUnprocessableEntity:
		return nil, newErrUnprocessableEntity(string(resp.Body()))
	default:
		return nil, newErrUnexpectedStatusCode(resp.StatusCode(), string(resp.Body()))
	}
}
