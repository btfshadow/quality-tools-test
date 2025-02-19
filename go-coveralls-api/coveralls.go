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

// Package coveralls provide functions to interact with Coveralls API
package coveralls

import (
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
)

const (
	defaultHostURL = "https://coveralls.io"
)

// Client is used to provide a single interface to interact with Coveralls API
type Client struct {
	client *resty.Client
	common service // Share the same client instance among all services

	// Host URL for Coveralls. Defaults to https://coveralls.io
	// Change this if you want to use private Coveralls server (untested)
	HostURL      *url.URL
	Repositories RepositoryService // Service to interact with repository-related endpoints
}

type service struct {
	client *Client
}

// NewClient returns a new Coveralls API Client
// t is the Coveralls API token
func NewClient(t string) *Client {
	cli := resty.New()
	cli.SetHeader("Accept", "application/json")
	cli.SetHeader("Authorization", fmt.Sprintf("token %s", t))

	url, _ := url.Parse(defaultHostURL)
	c := &Client{client: cli, HostURL: url}
	c.common.client = c
	c.Repositories = (*RepositoryServiceImpl)(&c.common)
	return c
}
