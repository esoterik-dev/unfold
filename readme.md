# Unfold

Unfold is an unofficial [Fold Money](https://fold.money) CLI client, which
covers the bare minimum API routes to fetch your transactions for a given
period and even write them to a sqlite database. It also provides options for
running an internal cron job through which this CLI acts as a daemon and
fetches transactions every time the cron job's timer is met.

> **Note:** This is a maintained fork of [wantguns/unfold](https://github.com/wantguns/unfold),
> which is no longer actively maintained. The original repository does not work
> as-is because Fold's API has since moved from v1 to v3, and several endpoints
> broke. This fork applies all fixes from [PR #2](https://github.com/wantguns/unfold/pull/2)
> and additional community fixes to restore full functionality, along with new
> local query commands.

> **⚠️ Warning (original author):** Fold's API is not publicly available, I had
> to MITM their app to write this tool, and so **there might be unforeseen
> consequences for your Fold account if you use this tool**.

### What's different from the original

- **Working API** — Content-Type header and v1 → v3 endpoint upgrade
- **Persistent tokens** — refreshed tokens are actually saved to disk
- **Stable device hash** — not regenerated every run
- **Crash-proof pagination** — bounds checks on empty responses
- **Enriched metadata** — 31 DB columns (up from 7) capturing full v3 transaction data
- **Non-interactive auth** — `--phone`, `--otp`, `--send-otp` flags for headless use
- **19 query subcommands** — local analytics without an MCP server

### Prerequisites

- You need a Fold Account, which is currently only available on an invite basis
- You need to connect to whichever banks you have using the Fold app first

### Installation

- Using golang's build system:
  ```bash
  go install github.com/wantguns/unfold
  ```
- Building locally
  ```bash
  go build -o unfold .
  ```

### Usage

1. First, login to your account:
    ```bash
    unfold login
    ```

2. Then you can fetch your transactions:

    a. In plaintext:
      ```bash
      unfold transactions
      ```

    b. In plaintext and also write to a db:
      ```bash
      # Write to a local file called `db.sqlite` by default
      unfold transactions -s 2023-09-20 --db
      ```

    c. Create an internal cron job to fetch transactions every 20 seconds and save them to a db:
      ```bash
      # Note: You need to enable the `-d` or `--db` flag to ensure that the changes are written to a database
      unfold transactions -s 2023-09-20 --db -w '@every 20s'
      12:19AM INF Cron job set for fetching transactions, going into daemon mode
      12:19AM INF Fetched transactions till 2023-10-17
      12:20AM INF Fetched transactions till 2023-10-17
      ```

3. Query your local transaction database:

    All query commands read from the local SQLite database populated by
    `unfold transactions -d`. They live under the `db` subcommand:

    ```bash
    # Recent transactions
    unfold db recent --limit 20

    # Search by merchant, amount, type, date range
    unfold db search --query zomato --start 2024-01-01

    # Spending summary (income vs spending, top merchants, avg daily)
    unfold db spend-summary --since 2024-01-01

    # Month-by-month income/spending/net
    unfold db monthly-trend

    # Top merchants by spend or frequency
    unfold db merchant-summary --limit 10 --sort frequency

    # Spending by category (Food, Transport, Shopping, etc.)
    unfold db category

    # Spending by payment mode (UPI, CARD, etc.)
    unfold db mode-breakdown

    # Average/min/max account balance by month
    unfold db balance-history

    # Spending by bank account
    unfold db account-breakdown

    # Merchants appearing in N+ distinct months (subscriptions, bills)
    unfold db recurring --months 3

    # Spending by weekday or day-of-month
    unfold db day-patterns

    # Monthly savings rate with rolling 3-month average
    unfold db savings-rate

    # 7-day spending summary vs 3-week rolling average
    unfold db weekly-digest

    # Full Indian financial year (Apr-Mar) report
    unfold db tax-report --year 2024

    # Find transactions unusually large compared to your normal spend
    unfold db unusual --multiplier 3.0

    # Side-by-side comparison of two date ranges
    unfold db compare --start1 2024-01-01 --end1 2024-06-30 --start2 2024-07-01 --end2 2024-12-31

    # Projected month-end spend based on current pace
    unfold db forecast

    # Current and longest streak of low-spend days
    unfold db streak --threshold 1000

    # Export filtered transactions to CSV
    unfold db export-csv --start 2024-01-01 --end 2024-12-31 --output fold-data.csv
    ```

    For all flags on a subcommand:
    ```bash
    unfold db search --help
    unfold db tax-report --help
    ```

There are a few more subcommands which Unfold provides and uses internally. You
can get a list by:
```
$ unfold
An unofficial cli client for fold.money

Usage:
  unfold [command]

Available Commands:
  availability  Returns a range of dates for when your banking data is available
  completion    Generate the autocompletion script for the specified shell
  db            Query your local transaction database
  help          Help about any command
  login         Log in to your fold account
  refresh       Refresh your auth tokens
  transactions  Prints the transactions from all of your accounts (default period: 1 month)
  user          Get your account details

Flags:
      --config string    config file (default is $HOME/.config/unfold/config.yaml)
      --db-path string   Path to SQLite database for query commands (default "db.sqlite")
  -v, --debug            Enable debug mode
  -h, --help             help for unfold

Use "unfold [command] --help" for more information about a command.
```

### Credits

[Fold Money](https://fold.money), for their Account Aggregator integration

[PR #2 by coherent-cache](https://github.com/wantguns/unfold/pull/2) — API fixes
and enriched v2 metadata

[naman0815/unfold-mcp](https://github.com/naman0815/unfold-mcp) — derivative that
inspired the query subcommands and additional bug fixes
