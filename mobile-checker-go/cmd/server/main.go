package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/yourusername/mobile-checker/api"
)

func main() {
	addr := flag.String("addr", ":5001", "HTTP server address")
	dataDir := flag.String("data-dir", defaultDataDir(), "Ofcom database directory")
	flag.Parse()

	fmt.Println("Note: Run 'mobile-checker setup' first if you haven't already.")
	srv := api.NewServer(*dataDir)
	log.Fatal(srv.ListenAndServe(*addr))
}

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mobile-checker", "data")
}
