package ofcom_test

import (
	"testing"

	"github.com/yourusername/mobile-checker/internal/ofcom"
)

func TestInterpret_FullRow(t *testing.T) {
	row := map[string]string{
		"postcode":       "SW1A1AA",
		"ee_4g":          "1.0",
		"o2_4g":          "0.95",
		"three_4g":       "0.88",
		"vodafone_4g":    "0.72",
		"ee_5g":          "0.60",
		"o2_5g":          "0.0",
		"three_5g":       "0.0",
		"vodafone_5g":    "0.55",
		"ee_voice":       "1.0",
		"o2_voice":       "1.0",
		"three_voice":    "0.90",
		"vodafone_voice": "0.85",
	}

	result := ofcom.Interpret(row)

	if result.Postcode != "SW1A1AA" {
		t.Errorf("expected SW1A1AA, got %s", result.Postcode)
	}
	if len(result.Operators) != 4 {
		t.Errorf("expected 4 operators, got %d", len(result.Operators))
	}
	if result.Overall.FourGCount != 4 {
		t.Errorf("expected 4G count 4, got %d", result.Overall.FourGCount)
	}
	if result.Overall.FiveGCount != 2 {
		t.Errorf("expected 5G count 2, got %d", result.Overall.FiveGCount)
	}

	ee := result.Operators[0]
	if ee.Name != "EE" {
		t.Errorf("expected first operator EE, got %s", ee.Name)
	}
	if !ee.HasFourG {
		t.Error("expected EE to have 4G")
	}
	if !ee.HasFiveG {
		t.Error("expected EE to have 5G")
	}
}

func TestInterpret_MissingColumns(t *testing.T) {
	row := map[string]string{"postcode": "EC1A1BB"}
	result := ofcom.Interpret(row)
	for _, op := range result.Operators {
		if op.HasFourG || op.HasFiveG || op.HasVoice {
			t.Errorf("expected no coverage for empty row, operator %s has coverage", op.Name)
		}
	}
}

func TestInterpret_PartialCoverage(t *testing.T) {
	row := map[string]string{
		"postcode": "LS11AA",
		"ee_4g":    "0.3", // below 50% threshold — not covered
		"o2_4g":    "0.8", // above threshold — covered
	}
	result := ofcom.Interpret(row)
	if result.Operators[0].HasFourG { // EE
		t.Error("EE 4G at 30% should not be marked as covered")
	}
	if !result.Operators[1].HasFourG { // O2
		t.Error("O2 4G at 80% should be marked as covered")
	}
}
