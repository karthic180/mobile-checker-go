// Package ofcom handles downloading, storing, and querying
// Ofcom Connected Nations mobile coverage data.
// Free open data, no API key required.
// Source: https://www.ofcom.org.uk/research-and-data/telecoms-research/connected-nations
package ofcom

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// MobileDataURLs maps dataset year to Ofcom mobile coverage download URL.
// Check https://www.ofcom.org.uk/research-and-data/telecoms-research/connected-nations
// for updated URLs when a new edition is released.
var MobileDataURLs = map[string]string{
	"2023": "https://www.ofcom.org.uk/siteassets/resources/documents/research-and-data/telecoms-research/connected-nations/connected-nations-2023/interactive-report/2023_mobile_pc_r01.zip",
	"2022": "https://www.ofcom.org.uk/siteassets/resources/documents/research-and-data/telecoms-research/connected-nations/connected-nations-2022/interactive-report/2022_mobile_pc_r03.zip",
}

// MobileRow represents mobile coverage data for a postcode.
type MobileRow struct {
	Postcode        string
	EE4G            float64
	O24G            float64
	Three4G         float64
	Vodafone4G      float64
	EE5G            float64
	O25G            float64
	Three5G         float64
	Vodafone5G      float64
	EEVoice         float64
	O2Voice         float64
	ThreeVoice      float64
	VodafoneVoice   float64
	AnyCoverage     float64
}

// MobileSummary holds human-readable mobile coverage for a postcode.
type MobileSummary struct {
	Postcode  string
	Operators []OperatorCoverage
	Overall   OverallCoverage
}

// OperatorCoverage holds coverage data for a single operator.
type OperatorCoverage struct {
	Name        string
	Voice       string
	FourG       string
	FiveG       string
	HasVoice    bool
	HasFourG    bool
	HasFiveG    bool
}

// OverallCoverage summarises coverage across all operators.
type OverallCoverage struct {
	AnyOperator string
	FourGCount  int // number of operators with 4G
	FiveGCount  int // number of operators with 5G
}

// Manager handles the Ofcom mobile dataset lifecycle.
type Manager struct {
	DataDir string
	DBPath  string
}

// NewManager creates a new Manager.
func NewManager(dataDir string) *Manager {
	return &Manager{
		DataDir: dataDir,
		DBPath:  filepath.Join(dataDir, "mobile.db"),
	}
}

// Setup downloads and builds the local SQLite database.
func (m *Manager) Setup(year string, force bool) error {
	if err := os.MkdirAll(m.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	csvPath := filepath.Join(m.DataDir, fmt.Sprintf("ofcom_mobile_%s.csv", year))

	if _, err := os.Stat(csvPath); os.IsNotExist(err) || force {
		if err := m.downloadData(year, csvPath); err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
	} else {
		fmt.Printf("Mobile CSV already exists at %s, skipping download.\n", csvPath)
	}

	if _, err := os.Stat(m.DBPath); os.IsNotExist(err) || force {
		if err := m.buildDatabase(csvPath); err != nil {
			return fmt.Errorf("database build failed: %w", err)
		}
	} else {
		fmt.Printf("Mobile database already exists at %s.\n", m.DBPath)
	}

	return nil
}

func (m *Manager) downloadData(year, csvPath string) error {
	url, ok := MobileDataURLs[year]
	if !ok {
		return fmt.Errorf("no URL for year %q, available: 2022, 2023", year)
	}

	fmt.Printf("Downloading Ofcom mobile %s dataset...\n", year)
	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from Ofcom", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to open ZIP: %w", err)
	}

	var csvFile *zip.File
	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			csvFile = f
			break
		}
	}
	if csvFile == nil {
		return fmt.Errorf("no CSV found inside Ofcom ZIP")
	}

	rc, err := csvFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	fmt.Println("Download complete.")
	return err
}

func (m *Manager) buildDatabase(csvPath string) error {
	fmt.Println("Building mobile database from Ofcom data (one-time setup)...")

	if _, err := os.Stat(m.DBPath); err == nil {
		os.Remove(m.DBPath)
	}

	db, err := sql.Open("sqlite3", m.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")

	f, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}

	for i, h := range headers {
		headers[i] = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(h, " ", "_")))
	}

	cols := make([]string, len(headers))
	for i, h := range headers {
		cols[i] = fmt.Sprintf(`"%s" TEXT`, h)
	}
	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS mobile (%s)`, strings.Join(cols, ", "))
	if _, err := db.Exec(createSQL); err != nil {
		return err
	}

	tx, _ := db.Begin()
	placeholders := strings.TrimRight(strings.Repeat("?,", len(headers)), ",")
	insertSQL := fmt.Sprintf(`INSERT INTO mobile VALUES (%s)`, placeholders)
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return err
	}

	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		for i, h := range headers {
			if h == "postcode" {
				record[i] = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(record[i]), " ", ""))
			}
		}
		args := make([]interface{}, len(record))
		for i, v := range record {
			args[i] = v
		}
		stmt.Exec(args...)
		count++
		if count%50000 == 0 {
			tx.Commit()
			tx, _ = db.Begin()
			stmt, _ = tx.Prepare(insertSQL)
			fmt.Printf("  Inserted %d rows...\n", count)
		}
	}
	tx.Commit()
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_postcode ON mobile(postcode)`)
	fmt.Printf("Mobile database built with %d rows.\n", count)
	return nil
}

// QueryPostcode returns the raw row for a postcode, or nil if not found.
func (m *Manager) QueryPostcode(postcode string) (map[string]string, error) {
	if _, err := os.Stat(m.DBPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database not found — run 'setup' first")
	}

	db, err := sql.Open("sqlite3", m.DBPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	pc := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(postcode), " ", ""))
	rows, err := db.Query("SELECT * FROM mobile WHERE postcode = ? LIMIT 1", pc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		return nil, nil
	}

	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}

	result := make(map[string]string, len(cols))
	for i, col := range cols {
		if vals[i] != nil {
			result[col] = fmt.Sprintf("%v", vals[i])
		}
	}
	return result, nil
}

// Interpret converts a raw Ofcom mobile row into a MobileSummary.
func Interpret(row map[string]string) MobileSummary {
	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := row[k]; ok && v != "" {
				return v
			}
		}
		return ""
	}

	covered := func(keys ...string) bool {
		v := get(keys...)
		if v == "" {
			return false
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return false
		}
		return f >= 0.5 // treat ≥50% as covered
	}

	pct := func(keys ...string) string {
		v := get(keys...)
		if v == "" {
			return "N/A"
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "N/A"
		}
		return fmt.Sprintf("%.0f%%", f*100)
	}

	operators := []OperatorCoverage{
		{
			Name:     "EE",
			Voice:    pct("ee_voice", "ee_voice_indoor"),
			FourG:    pct("ee_4g", "ee4g"),
			FiveG:    pct("ee_5g", "ee5g"),
			HasVoice: covered("ee_voice", "ee_voice_indoor"),
			HasFourG: covered("ee_4g", "ee4g"),
			HasFiveG: covered("ee_5g", "ee5g"),
		},
		{
			Name:     "O2",
			Voice:    pct("o2_voice", "o2_voice_indoor"),
			FourG:    pct("o2_4g", "o24g"),
			FiveG:    pct("o2_5g", "o25g"),
			HasVoice: covered("o2_voice", "o2_voice_indoor"),
			HasFourG: covered("o2_4g", "o24g"),
			HasFiveG: covered("o2_5g", "o25g"),
		},
		{
			Name:     "Three",
			Voice:    pct("three_voice", "three_voice_indoor"),
			FourG:    pct("three_4g", "three4g"),
			FiveG:    pct("three_5g", "three5g"),
			HasVoice: covered("three_voice", "three_voice_indoor"),
			HasFourG: covered("three_4g", "three4g"),
			HasFiveG: covered("three_5g", "three5g"),
		},
		{
			Name:     "Vodafone",
			Voice:    pct("vodafone_voice", "vodafone_voice_indoor"),
			FourG:    pct("vodafone_4g", "vodafone4g"),
			FiveG:    pct("vodafone_5g", "vodafone5g"),
			HasVoice: covered("vodafone_voice", "vodafone_voice_indoor"),
			HasFourG: covered("vodafone_4g", "vodafone4g"),
			HasFiveG: covered("vodafone_5g", "vodafone5g"),
		},
	}

	fourGCount := 0
	fiveGCount := 0
	for _, op := range operators {
		if op.HasFourG {
			fourGCount++
		}
		if op.HasFiveG {
			fiveGCount++
		}
	}

	return MobileSummary{
		Postcode:  get("postcode"),
		Operators: operators,
		Overall: OverallCoverage{
			AnyOperator: pct("any_operator", "any_coverage"),
			FourGCount:  fourGCount,
			FiveGCount:  fiveGCount,
		},
	}
}
