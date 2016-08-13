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

type Printer interface {
	Println(...interface{})
}

type Client struct {
	Logger     Printer
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

type SearchConfig struct {
	OlderThan int
	Random    bool
}

func (cl *Client) Search(query string, config SearchConfig) (*SearchResult, error) {
	uri := cl.baseUrl.ResolveReference(&url.URL{Path: "/query/" + query})

	// build uri paramters from config
	values := url.Values{}
	{
		if config.OlderThan > 0 {
			values.Set("older", strconv.Itoa(config.OlderThan))
		}

		if config.Random {
			values.Set("random", "true")
		}

		uri.RawQuery = values.Encode()
	}

	if cl.Logger != nil {
		cl.Logger.Println("Query tag api with url: ", uri.String())
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
