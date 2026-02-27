// Package checker combines postcodes.io and Ofcom mobile data.
package checker

import (
	"fmt"

	"github.com/yourusername/mobile-checker/internal/ofcom"
	"github.com/yourusername/mobile-checker/internal/postcode"
)

// Result is the unified output of a mobile coverage check.
type Result struct {
	Postcode   string                `json:"postcode"`
	Valid      bool                  `json:"valid"`
	Geographic *postcode.Result      `json:"geographic,omitempty"`
	Mobile     *ofcom.MobileSummary  `json:"mobile,omitempty"`
	Error      string                `json:"error,omitempty"`
	Note       string                `json:"note,omitempty"`
}

// Checker performs mobile coverage checks.
type Checker struct {
	postcodeClient *postcode.Client
	ofcomManager   *ofcom.Manager
}

// New creates a new Checker.
func New(dataDir string) *Checker {
	return &Checker{
		postcodeClient: postcode.NewClient(),
		ofcomManager:   ofcom.NewManager(dataDir),
	}
}

// Setup downloads and builds the Ofcom mobile database.
func (c *Checker) Setup(year string, force bool) error {
	return c.ofcomManager.Setup(year, force)
}

// Check performs a full mobile coverage check for a UK postcode.
func (c *Checker) Check(pc string) Result {
	normalised := postcode.Normalise(pc)
	result := Result{Postcode: normalised}

	geo, err := c.postcodeClient.Lookup(pc)
	if err != nil {
		result.Error = fmt.Sprintf("Postcode lookup failed: %v", err)
		return result
	}
	result.Valid = true
	result.Geographic = geo

	row, err := c.ofcomManager.QueryPostcode(normalised)
	if err != nil {
		result.Note = fmt.Sprintf("Mobile data unavailable: %v", err)
		return result
	}
	if row == nil {
		result.Note = "Postcode not found in Ofcom mobile dataset."
		return result
	}

	summary := ofcom.Interpret(row)
	result.Mobile = &summary
	return result
}

// CheckMultiple checks multiple postcodes concurrently.
func (c *Checker) CheckMultiple(postcodes []string) []Result {
	results := make([]Result, len(postcodes))
	ch := make(chan struct {
		idx int
		res Result
	}, len(postcodes))

	for i, pc := range postcodes {
		go func(idx int, p string) {
			ch <- struct {
				idx int
				res Result
			}{idx, c.Check(p)}
		}(i, pc)
	}

	for range postcodes {
		item := <-ch
		results[item.idx] = item.res
	}
	return results
}
