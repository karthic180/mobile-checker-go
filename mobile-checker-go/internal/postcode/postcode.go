// Package postcode provides a client for the postcodes.io API.
// Free, no API key required. Docs: https://postcodes.io
package postcode

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.postcodes.io"

// Client is an HTTP client for postcodes.io.
type Client struct {
	http *http.Client
}

// NewClient returns a new postcodes.io Client.
func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 10 * time.Second}}
}

// Result holds geographic data for a postcode.
type Result struct {
	Postcode                  string  `json:"postcode"`
	Country                   string  `json:"country"`
	Region                    string  `json:"region"`
	AdminDistrict             string  `json:"admin_district"`
	ParliamentaryConstituency string  `json:"parliamentary_constituency"`
	Latitude                  float64 `json:"latitude"`
	Longitude                 float64 `json:"longitude"`
	Eastings                  int     `json:"eastings"`
	Northings                 int     `json:"northings"`
}

type apiResponse struct {
	Status int     `json:"status"`
	Result *Result `json:"result"`
}

// Normalise strips spaces and uppercases a postcode.
func Normalise(pc string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(pc), " ", ""))
}

// Lookup returns geographic data for a UK postcode.
func (c *Client) Lookup(postcode string) (*Result, error) {
	pc := Normalise(postcode)
	resp, err := c.http.Get(fmt.Sprintf("%s/postcodes/%s", baseURL, pc))
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("postcode %q not found or invalid", postcode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("postcodes.io returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed apiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if parsed.Result == nil {
		return nil, fmt.Errorf("postcode %q returned no data", postcode)
	}
	return parsed.Result, nil
}
