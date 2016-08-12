package tagsapi

import (
	"net/http"
	"net/url"
	"io/ioutil"
	"encoding/json"
	"io"
	"strconv"
)

type SearchResult struct {
	Duration string `json:"duration"`
	Items    []int32 `json:"items"`
}

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	baseUrl    *url.URL
	httpClient HttpClient
}

func NewClient(httpClient HttpClient, baseUrl string) (*Client, error) {
	parsed, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseUrl: parsed,
		httpClient: httpClient,
	}, nil
}

func (cl *Client) Search(query string, olderThan int) (*SearchResult, error) {
	uri := cl.baseUrl.ResolveReference(&url.URL{Path: "/query/" + query})
	if olderThan > 0 {
		values := url.Values{}
		values.Set("older", strconv.Itoa(olderThan))
		uri.RawQuery = values.Encode()
	}

	request, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, err
	}

	response, err := cl.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	// defer cleanup of the response body/connection
	defer func() {
		io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
	}()

	result := &SearchResult{}
	if err := json.NewDecoder(response.Body).Decode(result); err != nil {
		return nil, err
	}

	return result, nil
}
