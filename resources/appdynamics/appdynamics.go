/*
 * Copyright 2017-2022 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package appdynamics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"resources/check"
	"resources/in"
	"resources/internal"
)

const latestURI = "https://download.appdynamics.com/download/downloadfilelatest/"
const fetchURI = "https://download.appdynamics.com/download/downloadfile/"

type AppDynamics struct {
	Source  source           `json:"source"`
	Version internal.Version `json:"version"`
}

type source struct {
	Type     string `json:"type"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type AppDynamicsAPIResponse struct {
	DownloadPath string `json:"download_path"`
	FileType     string `json:"filetype"`
	Version      string `json:"version"`
	Checksum     string `json:"sha256_checksum"`
}

type AppDynamicsAPIPageResponse struct {
	Count    int
	Next     string
	Previous string
	Results  []AppDynamicsAPIResponse
}

func (a AppDynamics) Check() (check.Result, error) {
	fmt.Fprintf(os.Stderr, "Checking... Input: '%s', '%s'", a.Version.Ref, a.Source.Type)
	result := check.Result{Since: a.Version}

	latest, err := a.latestVersion()
	if err != nil {
		return check.Result{}, fmt.Errorf("unable to get latest versions\n%w", err)
	}

	fmt.Fprintf(os.Stderr, "Latest found: %v", latest)
	if latest.Version != "" {
		result.Add(internal.Version{Ref: latest.Version})
	}

	return result, err
}

func (a AppDynamics) In(destination string) (in.Result, error) {
	latest, err := a.fetchVersion(a.Version.Ref)
	if err != nil {
		return in.Result{}, fmt.Errorf("unable to get latest versions\n%w", err)
	}

	addToken := func(request *http.Request) *http.Request {
		token, err := a.fetchAPIToken()
		if err != nil {
			panic(fmt.Errorf("unable to fetch token\n%w", err))
		}

		request.Header.Add("Authorization", token)

		return request
	}

	sha256, err := in.Artifact{
		Name:        fmt.Sprintf("appdynamics_linux_%s.tar.gz", a.Version.Ref),
		Version:     a.Version,
		URI:         latest.DownloadPath,
		Destination: destination,
	}.Download(addToken)
	if err != nil {
		return in.Result{}, err
	}

	if sha256 != latest.Checksum {
		return in.Result{}, fmt.Errorf("downloaded checksum [%s] does not match expected checksum [%s]", sha256, latest.Checksum)
	}

	return in.Result{
		Version: a.Version,
		Metadata: []in.Metadata{
			{Name: "uri", Value: latest.DownloadPath},
			{Name: "sha256", Value: latest.Checksum},
		},
	}, nil
}

func (a AppDynamics) fetchAPIToken() (string, error) {
	uri := "https://identity.msrv.saas.appdynamics.com/v2.0/oauth/token"

	resp, err := http.Post(uri, "application/json",
		bytes.NewBufferString(
			fmt.Sprintf(`{"username": "%s","password": "%s","scopes": ["download"]}`, a.Source.User, a.Source.Password)))
	if err != nil {
		return "", fmt.Errorf("unable to post %s\n%w", uri, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unable to read token %s: %d", uri, resp.StatusCode)
	}

	var raw struct {
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", fmt.Errorf("unable to decode payload\n%w", err)
	}
	return fmt.Sprintf("%s %s", raw.TokenType, raw.AccessToken), nil
}

func (a AppDynamics) latestVersion() (AppDynamicsAPIResponse, error) {
	resp, err := http.Get(latestURI)
	if err != nil {
		return AppDynamicsAPIResponse{}, fmt.Errorf("unable to get %s\n%w", latestURI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return AppDynamicsAPIResponse{}, fmt.Errorf("unable to download %s: %d", latestURI, resp.StatusCode)
	}

	var raw []AppDynamicsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return AppDynamicsAPIResponse{}, fmt.Errorf("unable to decode payload\n%w", err)
	}

	for _, r := range raw {
		if a.Source.Type == r.FileType {
			return r, nil
		}
	}

	return AppDynamicsAPIResponse{}, nil
}

func (a AppDynamics) fetchVersion(version string) (AppDynamicsAPIResponse, error) {
	req, err := http.NewRequest("GET", fetchURI, nil)
	if err != nil {
		return AppDynamicsAPIResponse{}, fmt.Errorf("unable to create GET %s request\n%w", fetchURI, err)
	}

	q := req.URL.Query()
	q.Add("apm", a.Source.Type)
	q.Add("apm_os", "linux")
	q.Add("version", version)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return AppDynamicsAPIResponse{}, fmt.Errorf("unable to get %s\n%w", fetchURI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return AppDynamicsAPIResponse{}, fmt.Errorf("unable to download %s: %d", latestURI, resp.StatusCode)
	}

	var raw AppDynamicsAPIPageResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return AppDynamicsAPIResponse{}, fmt.Errorf("unable to decode payload\n%w", err)
	}

	for _, r := range raw.Results {
		if a.Source.Type == r.FileType {
			return r, nil
		}
	}

	return AppDynamicsAPIResponse{}, nil
}
