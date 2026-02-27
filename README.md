#Mobile Coverage Checker (Go)

Check UK mobile coverage for all four operators (EE, O2, Three, Vodafone) by postcode — voice, 4G, and 5G — using free Ofcom open data.

## Data Sources

| Source | What it provides | Key |
|---|---|---|
| [postcodes.io](https://postcodes.io) | Postcode validation, region, lat/lon | None |
| [Ofcom Connected Nations](https://www.ofcom.org.uk/research-and-data/telecoms-research/connected-nations) | Mobile coverage per postcode (all operators) | None |

---

## Quick Start

```bash
git clone https://github.com/karthic180/mobile-checker-go.git
cd mobile-checker-go
go mod tidy
go build -o mobile-checker ./cmd/mobile

# One-time setup — downloads Ofcom mobile dataset
./mobile-checker setup

# Check a postcode
./mobile-checker check SW1A1AA
```

### Example output

```
────────────────────────────────────────────────────
  Postcode: SW1A1AA
────────────────────────────────────────────────────
  Region:   London
  District: Westminster
  Country:  England
  Lat/Lon:  51.501009, -0.141588

  Operator     Voice      4G         5G
  ────────────────────────────────────────────
  EE           ✓ 100%    ✓ 100%    ✓ 80%
  O2           ✓ 100%    ✓ 98%     ✗ 0%
  Three        ✓ 95%     ✓ 92%     ✗ 0%
  Vodafone     ✓ 90%     ✓ 88%     ✓ 72%
  ────────────────────────────────────────────
  4G operators: 4/4   5G operators: 2/4

  Source: Ofcom Connected Nations (open data)
```

### Multiple postcodes (concurrent)

```bash
./mobile-checker check SW1A1AA EC1A1BB W1A0AX
```

### JSON output

```bash
./mobile-checker check SW1A1AA --json
```

---

## REST API

```bash
go build -o mobile-server ./cmd/server
./mobile-server --addr :5001
```

| Method | Endpoint | Description |
|---|---|---|
| GET | `/health` | Health check |
| GET | `/api/mobile/{postcode}` | Coverage check |
| POST | `/api/mobile/bulk` | Up to 50 postcodes |

```bash
curl http://localhost:5001/api/mobile/SW1A1AA
```

---

## Project Structure

```
mobile-checker-go/
├── cmd/
│   ├── mobile/main.go       # CLI entry point
│   └── server/main.go       # HTTP API server
├── internal/
│   ├── postcode/postcode.go # postcodes.io client
│   ├── ofcom/
│   │   ├── ofcom.go         # Ofcom mobile data
│   │   └── ofcom_test.go
│   └── checker/checker.go   # Combines both sources
├── api/server.go            # HTTP handlers
├── go.mod
├── Makefile
└── README.md
```

---

## Licence

MIT. Data: Open Government Licence v3.0 (Ofcom + postcodes.io)
