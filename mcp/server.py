#!/usr/bin/env python3
"""Generic MCP server for the Unfold CLI.

Wraps the `unfold` binary as MCP tools so any AI that speaks Model Context
Protocol (Claude Desktop, ChatGPT, Cursor, etc.) can query your Fold Money
transaction data.

Usage:
    python server.py
    UNFOLD_BIN=/path/to/unfold python server.py
"""

import shutil
import subprocess
import sys
import os

from mcp.server.fastmcp import FastMCP

# ── Find the binary ────────────────────────────────────────────────────────

UNFOLD_BIN = os.environ.get("UNFOLD_BIN") or shutil.which("unfold")
if not UNFOLD_BIN:
    print("ERROR: 'unfold' binary not found. Set UNFOLD_BIN or add it to PATH.", file=sys.stderr)
    sys.exit(1)

# ── Server ──────────────────────────────────────────────────────────────────

mcp = FastMCP(
    "unfold",
    
    instructions="Query your Fold Money transaction data stored in a local SQLite database.",
)


def _run(args: list[str], db_path: str | None = None) -> str:
    """Run an unfold CLI command and return stdout."""
    cmd = [UNFOLD_BIN]
    if db_path:
        cmd += ["--db-path", db_path]
    cmd += args
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
        output = result.stdout.strip()
        if result.returncode != 0:
            error = result.stderr.strip()
            return f"Error (exit {result.returncode}): {error}\n{output}" if output else f"Error: {error}"
        return output or "(no results)"
    except subprocess.TimeoutExpired:
        return "Error: command timed out after 30s"
    except FileNotFoundError:
        return f"Error: unfold binary not found at {UNFOLD_BIN}"


# ── Tools ───────────────────────────────────────────────────────────────────


@mcp.tool()
def login(phone: str = "", otp: str = "", send_otp: bool = False) -> str:
    """Log in to your Fold account. Pass phone number to initiate login, or phone + otp to complete verification. Use send_otp=True to request an OTP without waiting for input.

    Args:
        phone: Phone number registered with Fold (e.g. "+919876543210")
        otp: OTP code received via SMS (provide after requesting OTP)
        send_otp: If True, sends the OTP and exits immediately (headless mode)
    """
    args = ["login"]
    if phone:
        args += ["--phone", phone]
    if otp:
        args += ["--otp", otp]
    if send_otp:
        args += ["--send-otp"]
    return _run(args)


@mcp.tool()
def sync_transactions(
    since: str = "",
    till: str = "",
    db_path: str = "",
) -> str:
    """Fetch transactions from Fold Money and store them in the local SQLite database.

    Args:
        since: Fetch transactions from this date (YYYY-MM-DD). Defaults to 30 days ago.
        till: Fetch transactions up to this date (YYYY-MM-DD). Defaults to today.
        db_path: Path to SQLite database file. Default: db.sqlite
    """
    args = ["transactions", "--db"]
    # `transactions` has its own --db-path flag, which shadows the root flag.
    # Keep this path after the subcommand so sync writes to the same database
    # that the query tools read.
    if db_path:
        args += ["--db-path", db_path]
    if since:
        args += ["--since", since]
    if till:
        args += ["--till", till]
    return _run(args)


@mcp.tool()
def recent(limit: int = 20, db_path: str = "") -> str:
    """Show the most recent transactions.

    Args:
        limit: Number of transactions to show (default 20)
        db_path: Path to SQLite database. Default: db.sqlite
    """
    return _run(["db", "recent", "-n", str(limit)], db_path=db_path or None)


@mcp.tool()
def search(
    query: str = "",
    start: str = "",
    end: str = "",
    type: str = "",
    mode: str = "",
    min_amount: float = 0,
    max_amount: float = 0,
    limit: int = 50,
    db_path: str = "",
) -> str:
    """Search transactions by merchant, narration, amount, type, date, or payment mode.

    Args:
        query: Text search on merchant/narration/summary
        start: Start date (YYYY-MM-DD)
        end: End date (YYYY-MM-DD)
        type: Filter by type: INCOMING or OUTGOING
        mode: Filter by payment mode: UPI, CARD, NEFT, IMPS, etc.
        min_amount: Minimum transaction amount
        max_amount: Maximum transaction amount
        limit: Max results (default 50)
        db_path: Path to SQLite database. Default: db.sqlite
    """
    args = ["db", "search"]
    if query:
        args += ["-q", query]
    if start:
        args += ["--start", start]
    if end:
        args += ["--end", end]
    if type:
        args += ["--type", type]
    if mode:
        args += ["--mode", mode]
    if min_amount > 0:
        args += ["--min-amount", str(min_amount)]
    if max_amount > 0:
        args += ["--max-amount", str(max_amount)]
    args += ["-n", str(limit)]
    return _run(args, db_path=db_path or None)


@mcp.tool()
def spend_summary(since: str = "", until: str = "", db_path: str = "") -> str:
    """Show spending summary: income vs spending, top merchants, and average daily spend.

    Args:
        since: Start date (YYYY-MM-DD). Default: 30 days ago.
        until: End date (YYYY-MM-DD). Default: today.
        db_path: Path to SQLite database. Default: db.sqlite
    """
    args = ["db", "spend-summary"]
    if since:
        args += ["--start", since]
    if until:
        args += ["--end", until]
    return _run(args, db_path=db_path or None)


@mcp.tool()
def merchant_summary(
    limit: int = 10,
    sort: str = "spend",
    since: str = "",
    until: str = "",
    db_path: str = "",
) -> str:
    """Show top merchants ranked by total spend or transaction frequency.

    Args:
        limit: Number of merchants to show (default 10)
        sort: Sort by 'spend' (total amount) or 'frequency' (count). Default: spend.
        since: Start date (YYYY-MM-DD)
        until: End date (YYYY-MM-DD)
        db_path: Path to SQLite database. Default: db.sqlite
    """
    args = ["db", "merchant-summary", "-n", str(limit), "--sort", sort]
    if since:
        args += ["--start", since]
    if until:
        args += ["--end", until]
    return _run(args, db_path=db_path or None)


@mcp.tool()
def monthly_trend(db_path: str = "") -> str:
    """Show month-by-month income, spending, and net cash flow.

    Args:
        db_path: Path to SQLite database. Default: db.sqlite
    """
    return _run(["db", "monthly-trend"], db_path=db_path or None)


@mcp.tool()
def balance_history(db_path: str = "") -> str:
    """Show average, minimum, and maximum account balance per month.

    Args:
        db_path: Path to SQLite database. Default: db.sqlite
    """
    return _run(["db", "balance-history"], db_path=db_path or None)


@mcp.tool()
def mode_breakdown(since: str = "", until: str = "", db_path: str = "") -> str:
    """Show spending and income broken down by payment mode (UPI, CARD, NEFT, etc.).

    Args:
        since: Start date (YYYY-MM-DD)
        until: End date (YYYY-MM-DD)
        db_path: Path to SQLite database. Default: db.sqlite
    """
    args = ["db", "mode-breakdown"]
    if since:
        args += ["--start", since]
    if until:
        args += ["--end", until]
    return _run(args, db_path=db_path or None)


@mcp.tool()
def recurring(months: int = 3, db_path: str = "") -> str:
    """Find merchants that appear in multiple months — subscriptions, bills, recurring habits.

    Args:
        months: Minimum distinct months a merchant must appear in (default 3)
        db_path: Path to SQLite database. Default: db.sqlite
    """
    return _run(["db", "recurring", "--min-months", str(months)], db_path=db_path or None)


@mcp.tool()
def account_breakdown(since: str = "", until: str = "", db_path: str = "") -> str:
    """Show spending and income broken down by bank account.

    Args:
        since: Start date (YYYY-MM-DD)
        until: End date (YYYY-MM-DD)
        db_path: Path to SQLite database. Default: db.sqlite
    """
    args = ["db", "account-breakdown"]
    if since:
        args += ["--start", since]
    if until:
        args += ["--end", until]
    return _run(args, db_path=db_path or None)


@mcp.tool()
def day_patterns(db_path: str = "") -> str:
    """Show spending patterns by day of week and day of month.

    Args:
        db_path: Path to SQLite database. Default: db.sqlite
    """
    return _run(["db", "day-patterns"], db_path=db_path or None)


@mcp.tool()
def category_breakdown(since: str = "", until: str = "", db_path: str = "") -> str:
    """Show spending grouped by category (Food Delivery, Transport, Shopping, Bills, etc.).

    Args:
        since: Start date (YYYY-MM-DD)
        until: End date (YYYY-MM-DD)
        db_path: Path to SQLite database. Default: db.sqlite
    """
    args = ["db", "category"]
    if since:
        args += ["--start", since]
    if until:
        args += ["--end", until]
    return _run(args, db_path=db_path or None)


@mcp.tool()
def savings_rate(db_path: str = "") -> str:
    """Show month-by-month savings rate with rolling 3-month average and trend.

    Args:
        db_path: Path to SQLite database. Default: db.sqlite
    """
    return _run(["db", "savings-rate"], db_path=db_path or None)


@mcp.tool()
def weekly_digest(db_path: str = "") -> str:
    """Show 7-day spending summary with comparison to rolling 3-week average.

    Args:
        db_path: Path to SQLite database. Default: db.sqlite
    """
    return _run(["db", "weekly-digest"], db_path=db_path or None)


@mcp.tool()
def tax_report(year: int = 0, db_path: str = "") -> str:
    """Generate a full Indian financial year (Apr-Mar) report with income, spending, and savings rate.

    Args:
        year: FY start year (e.g. 2024 for FY 2024-25). Default: most recent complete FY.
        db_path: Path to SQLite database. Default: db.sqlite
    """
    args = ["db", "tax-report"]
    if year > 0:
        args += ["--year", str(year)]
    return _run(args, db_path=db_path or None)


@mcp.tool()
def unusual_transactions(
    multiplier: float = 3.0,
    since: str = "",
    until: str = "",
    db_path: str = "",
) -> str:
    """Find transactions that are unusually large compared to your normal spend at that merchant.

    Args:
        multiplier: Flag transactions exceeding this multiple of the merchant average (default 3.0)
        since: Start date (YYYY-MM-DD)
        until: End date (YYYY-MM-DD)
        db_path: Path to SQLite database. Default: db.sqlite
    """
    args = ["db", "unusual", "--multiplier", str(multiplier)]
    if since:
        args += ["--start", since]
    if until:
        args += ["--end", until]
    return _run(args, db_path=db_path or None)


@mcp.tool()
def compare_periods(
    start1: str,
    end1: str,
    start2: str,
    end2: str,
    db_path: str = "",
) -> str:
    """Compare spending side-by-side across two date ranges.

    Args:
        start1: First period start (YYYY-MM-DD)
        end1: First period end (YYYY-MM-DD)
        start2: Second period start (YYYY-MM-DD)
        end2: Second period end (YYYY-MM-DD)
        db_path: Path to SQLite database. Default: db.sqlite
    """
    args = [
        "db", "compare",
        "--p1-start", start1, "--p1-end", end1,
        "--p2-start", start2, "--p2-end", end2,
    ]
    return _run(args, db_path=db_path or None)


@mcp.tool()
def forecast(db_path: str = "") -> str:
    """Project month-end spending based on current daily pace vs last month.

    Args:
        db_path: Path to SQLite database. Default: db.sqlite
    """
    return _run(["db", "forecast"], db_path=db_path or None)


@mcp.tool()
def streak(
    threshold: float = 1000,
    lookback: int = 90,
    db_path: str = "") -> str:
    """Show current and longest streak of days where spending stayed below a threshold.

    Args:
        threshold: Daily spending limit in INR (default 1000)
        lookback: Days to look back (default 90)
        db_path: Path to SQLite database. Default: db.sqlite
    """
    args = ["db", "streak", "--threshold", str(threshold), "--lookback", str(lookback)]
    return _run(args, db_path=db_path or None)


@mcp.tool()
def export_csv(
    start: str = "",
    end: str = "",
    output: str = "export.csv",
    db_path: str = "",
) -> str:
    """Export filtered transactions to a CSV file.

    Args:
        start: Start date (YYYY-MM-DD)
        end: End date (YYYY-MM-DD)
        output: Output file path (default: export.csv)
        db_path: Path to SQLite database. Default: db.sqlite
    """
    args = ["db", "export-csv", "--output", output]
    if start:
        args += ["--start", start]
    if end:
        args += ["--end", end]
    return _run(args, db_path=db_path or None)


@mcp.tool()
def user_info() -> str:
    """Get your Fold account details (name, phone, email)."""
    return _run(["user"])


@mcp.tool()
def availability() -> str:
    """Check the date range of banking data available from Fold."""
    return _run(["availability"])


# ── Entry point ──────────────────────────────────────────────────────────────

if __name__ == "__main__":
    mcp.run(transport="stdio")
