package buildkite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Adapted from https://github.com/buildkite/go-buildkite/blob/568b6651b687ccf6893ada08086ce58b072538b6/buildkite/buildkite.go.

const (
	restBaseURL   = "https://api.buildkite.com/"
	restUserAgent = "terraform-provider-buildkite/" + Version
)

type RESTClient struct {
	// HTTP client used to communicate with the API.
	client *http.Client

	// Base URL for API requests.  Defaults to the public Buildkite API. BaseURL should
	// always be specified with a trailing slash.
	baseURL *url.URL

	// User agent used when communicating with the Buildkite API.
	userAgent string

	organization string

	Pipelines *PipelinesService
}

func NewRESTClient(httpClient *http.Client, organization string) *RESTClient {
	baseURL, _ := url.Parse(restBaseURL)

	c := &RESTClient{
		client:       httpClient,
		baseURL:      baseURL,
		userAgent:    restUserAgent,
		organization: organization,
	}

	c.Pipelines = &PipelinesService{c}

	return c
}

// NewRequest creates an API request. A relative URL can be provided in urlStr,
// in which case it is resolved relative to the BaseURL of the Client.
// Relative URLs should always be specified without a preceding slash.  If
// specified, the value pointed to by body is JSON encoded and included as the
// request body.
func (c *RESTClient) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	u := c.baseURL.ResolveReference(rel)

	buf := new(bytes.Buffer)
	if body != nil {
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if c.userAgent != "" {
		req.Header.Add("User-Agent", c.userAgent)
	}

	return req, nil
}

// Do sends an API request and returns the API response.  The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred.  If v implements the io.Writer
// interface, the raw response body will be written to v, without attempting to
// first decode it.
func (c *RESTClient) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return resp, err
	}

	defer resp.Body.Close()
	defer io.Copy(ioutil.Discard, resp.Body)

	if err := checkResponse(resp); err != nil {
		// even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		return resp, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
		}
	}

	return resp, err
}

// ErrorResponse provides a message.
type ErrorResponse struct {
	Response *http.Response // HTTP response that caused this error
	Message  string         `json:"message"` // error message
	RawBody  []byte         `json:"-"`       // Raw Response Body
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %v",
		r.Response.Request.Method, r.Response.Request.URL,
		r.Response.StatusCode, r.Message)
}

func checkResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	data, err := ioutil.ReadAll(r.Body)
	errorResponse := &ErrorResponse{Response: r, RawBody: data}
	if err == nil && data != nil {
		json.Unmarshal(data, errorResponse)
	}
	return errorResponse
}
