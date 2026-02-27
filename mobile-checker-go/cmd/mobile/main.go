// main.go — UK Mobile Coverage Checker CLI
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourusername/mobile-checker/internal/checker"
)

const banner = `
╔══════════════════════════════════════════════╗
║        UK Mobile Coverage Checker            ║
║  Data: Ofcom Connected Nations + postcodes.io║
╚══════════════════════════════════════════════╝
`

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mobile-checker", "data")
}

func main() {
	var dataDir string
	var jsonOutput bool
	var year string
	var force bool

	c := checker.New(defaultDataDir())

	root := &cobra.Command{
		Use:   "mobile-checker",
		Short: "UK Mobile Coverage Checker",
		Long:  banner + "Check UK mobile coverage using free Ofcom open data and postcodes.io.",
	}
	root.PersistentFlags().StringVar(&dataDir, "data-dir", defaultDataDir(), "Directory to store the Ofcom database")

	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Download and build the Ofcom mobile database (run once)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c = checker.New(dataDir)
			fmt.Println(banner)
			fmt.Printf("Setting up Ofcom mobile %s dataset...\n", year)
			if err := c.Setup(year, force); err != nil {
				return err
			}
			fmt.Println("\n✓ Setup complete.")
			fmt.Println("  You can now run: mobile-checker check <POSTCODE>")
			return nil
		},
	}
	setupCmd.Flags().StringVar(&year, "year", "2023", "Ofcom dataset year (2022 or 2023)")
	setupCmd.Flags().BoolVar(&force, "force", false, "Force re-download even if data exists")

	checkCmd := &cobra.Command{
		Use:     "check [POSTCODE...]",
		Short:   "Check mobile coverage for one or more postcodes",
		Args:    cobra.MinimumNArgs(1),
		Example: "  mobile-checker check SW1A1AA\n  mobile-checker check SW1A1AA EC1A1BB --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			c = checker.New(dataDir)
			var results []checker.Result
			if len(args) == 1 {
				results = []checker.Result{c.Check(args[0])}
			} else {
				results = c.CheckMultiple(args)
			}
			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}
			for i, r := range results {
				printResult(r)
				if i < len(results)-1 {
					fmt.Println()
				}
			}
			return nil
		},
	}
	checkCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")

	root.AddCommand(setupCmd, checkCmd)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func printResult(r checker.Result) {
	sep := strings.Repeat("─", 52)
	fmt.Printf("\n%s\n", sep)
	fmt.Printf("  Postcode: %s\n", r.Postcode)
	fmt.Printf("%s\n", sep)

	if r.Error != "" {
		fmt.Printf("  ✗ %s\n", r.Error)
		return
	}

	if g := r.Geographic; g != nil {
		fmt.Printf("  Region:   %s\n", g.Region)
		fmt.Printf("  District: %s\n", g.AdminDistrict)
		fmt.Printf("  Country:  %s\n", g.Country)
		fmt.Printf("  Lat/Lon:  %.6f, %.6f\n", g.Latitude, g.Longitude)
	}

	if r.Note != "" {
		fmt.Printf("\n  Note: %s\n", r.Note)
		return
	}

	if r.Mobile == nil {
		fmt.Println("\n  Mobile data: Not available")
		return
	}

	mob := r.Mobile
	fmt.Printf("\n  %-12s %-10s %-10s %-10s\n", "Operator", "Voice", "4G", "5G")
	fmt.Printf("  %s\n", strings.Repeat("─", 44))
	for _, op := range mob.Operators {
		voice := icon(op.HasVoice) + " " + op.Voice
		fg := icon(op.HasFourG) + " " + op.FourG
		ffg := icon(op.HasFiveG) + " " + op.FiveG
		fmt.Printf("  %-12s %-10s %-10s %-10s\n", op.Name, voice, fg, ffg)
	}
	fmt.Printf("  %s\n", strings.Repeat("─", 44))
	fmt.Printf("  4G operators: %d/4   5G operators: %d/4\n",
		mob.Overall.FourGCount, mob.Overall.FiveGCount)
	fmt.Println("\n  Source: Ofcom Connected Nations (open data)")
}

func icon(b bool) string {
	if b {
		return "✓"
	}
	return "✗"
}
