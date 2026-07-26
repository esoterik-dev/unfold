package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/esoterik-dev/unfold/db"
)

const (
	foodCategoryID     = "0b55293c-5b8d-48e4-91be-4312b87dd714"
	transferCategoryID = "247e3e5d-59bf-4bf8-82ff-c152569893ea"
)

type fixtureTxn struct {
	id, date, kind, merchant, categoryID, mode string
	amount                                     float64
}

func setupAnalyticsFixture(t *testing.T, txns ...fixtureTxn) {
	t.Helper()
	db.Conn = nil
	db.Init(filepath.Join(t.TempDir(), "analytics.sqlite"))
	for _, txn := range txns {
		timestamp, err := time.Parse(time.DateOnly, txn.date)
		if err != nil {
			t.Fatalf("parse fixture date %q: %v", txn.date, err)
		}
		if err := db.Conn.Create(&db.Transactions{
			UUID:       txn.id,
			Timestamp:  timestamp,
			Type:       txn.kind,
			Merchant:   txn.merchant,
			CategoryID: txn.categoryID,
			Mode:       txn.mode,
			Amount:     txn.amount,
			Account:    "Fixture account",
		}).Error; err != nil {
			t.Fatalf("insert fixture %q: %v", txn.id, err)
		}
	}
	t.Cleanup(func() { db.Conn = nil })
}

func runAnalyticsCommand(t *testing.T, command *cobra.Command) string {
	t.Helper()
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	command.Run(command, nil)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = oldStdout
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	return string(output)
}

func setFlag(t *testing.T, command *cobra.Command, name, value string) {
	t.Helper()
	flag := command.Flags().Lookup(name)
	if flag == nil {
		t.Fatalf("flag %q missing from %s", name, command.Name())
	}
	original := flag.Value.String()
	if err := flag.Value.Set(value); err != nil {
		t.Fatalf("set --%s: %v", name, err)
	}
	t.Cleanup(func() { _ = flag.Value.Set(original) })
}

func TestSearchFiltersFixtureByDocumentedFlags(t *testing.T) {
	setupAnalyticsFixture(t,
		fixtureTxn{"old", "2024-01-31", "OUTGOING", "SWIGGY", foodCategoryID, "UPI", 200},
		fixtureTxn{"match", "2024-02-10", "OUTGOING", "SWIGGY", foodCategoryID, "UPI", 500},
		fixtureTxn{"wrong-mode", "2024-02-11", "OUTGOING", "SWIGGY", foodCategoryID, "CARD", 500},
		fixtureTxn{"income", "2024-02-12", "INCOMING", "Salary", "", "UPI", 500},
	)
	setFlag(t, searchCmd, "query", "SWIGGY")
	setFlag(t, searchCmd, "start", "2024-02-01")
	setFlag(t, searchCmd, "end", "2024-02-29")
	setFlag(t, searchCmd, "type", "OUTGOING")
	setFlag(t, searchCmd, "mode", "UPI")
	setFlag(t, searchCmd, "min-amount", "400")
	setFlag(t, searchCmd, "max-amount", "600")
	setFlag(t, searchCmd, "limit", "10")

	output := runAnalyticsCommand(t, searchCmd)
	if !strings.Contains(output, "2024-02-10 | -₹500 | Swiggy | Food Delivery") {
		t.Fatalf("expected filtered transaction, got:\n%s", output)
	}
	for _, unexpected := range []string{"2024-01-31", "2024-02-11", "2024-02-12"} {
		if strings.Contains(output, unexpected) {
			t.Fatalf("unexpected transaction %s in output:\n%s", unexpected, output)
		}
	}
}

func TestAnalyticsAggregationsCategorizationAndRecurringCounts(t *testing.T) {
	setupAnalyticsFixture(t,
		fixtureTxn{"swiggy-jan", "2024-01-05", "OUTGOING", "SWIGGY", foodCategoryID, "UPI", 100},
		fixtureTxn{"swiggy-feb", "2024-02-05", "OUTGOING", "SWIGGY", foodCategoryID, "UPI", 150},
		fixtureTxn{"swiggy-mar", "2024-03-05", "OUTGOING", "SWIGGY", foodCategoryID, "UPI", 200},
		fixtureTxn{"salary", "2024-03-10", "INCOMING", "Employer", "", "NEFT", 1000},
		fixtureTxn{"transfer", "2024-03-11", "OUTGOING", "Own account", transferCategoryID, "UPI", 300},
	)
	setFlag(t, spendSummaryCmd, "start", "2024-01-01")
	setFlag(t, spendSummaryCmd, "end", "2024-03-31")
	setFlag(t, spendSummaryCmd, "exclude-transfers", "true")
	spendOutput := runAnalyticsCommand(t, spendSummaryCmd)
	for _, expected := range []string{"(91 days)", "Total Spent:    ₹450", "Total Income:   ₹1000", "Net:            ₹550", "Swiggy — ₹450"} {
		if !strings.Contains(spendOutput, expected) {
			t.Fatalf("spend summary missing %q:\n%s", expected, spendOutput)
		}
	}

	setFlag(t, recurringCmd, "start", "2024-01-01")
	setFlag(t, recurringCmd, "end", "2024-03-31")
	setFlag(t, recurringCmd, "min-months", "3")
	setFlag(t, recurringCmd, "limit", "10")
	recurringOutput := runAnalyticsCommand(t, recurringCmd)
	if !strings.Contains(recurringOutput, "Swiggy") || !strings.Contains(recurringOutput, "      3") || !strings.Contains(recurringOutput, "₹450") {
		t.Fatalf("recurring output did not report three distinct months:\n%s", recurringOutput)
	}

	setFlag(t, categoryCmd, "start", "2024-01-01")
	setFlag(t, categoryCmd, "end", "2024-03-31")
	categoryOutput := runAnalyticsCommand(t, categoryCmd)
	if !strings.Contains(categoryOutput, "Food & Drinks") || !strings.Contains(categoryOutput, "₹450") {
		t.Fatalf("category output did not aggregate Fold category ID:\n%s", categoryOutput)
	}
}

func TestCompareReportsInclusiveHistoricalDayCounts(t *testing.T) {
	setupAnalyticsFixture(t,
		fixtureTxn{"p1", "2024-02-01", "OUTGOING", "Store", "", "CARD", 100},
		fixtureTxn{"p2", "2024-03-01", "OUTGOING", "Store", "", "CARD", 200},
	)
	setFlag(t, compareCmd, "p1-start", "2024-02-01")
	setFlag(t, compareCmd, "p1-end", "2024-02-29")
	setFlag(t, compareCmd, "p2-start", "2024-03-01")
	setFlag(t, compareCmd, "p2-end", "2024-03-31")
	output := runAnalyticsCommand(t, compareCmd)
	for _, expected := range []string{"Period 1: 2024-02-01 to 2024-02-29 (29 days)", "Period 2: 2024-03-01 to 2024-03-31 (31 days)", "₹3", "₹6"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("comparison output missing %q:\n%s", expected, output)
		}
	}
}

func TestForecastUsesCurrentAndPreviousMonthFixtureData(t *testing.T) {
	now := time.Now()
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonth := thisMonthStart.AddDate(0, -1, 0)
	setupAnalyticsFixture(t,
		fixtureTxn{"last-month", lastMonth.Format(time.DateOnly), "OUTGOING", "Store", "", "CARD", 300},
		fixtureTxn{"this-month", thisMonthStart.Format(time.DateOnly), "OUTGOING", "Store", "", "CARD", 100},
	)
	setFlag(t, forecastCmd, "exclude-transfers", "false")
	output := runAnalyticsCommand(t, forecastCmd)
	for _, expected := range []string{"Month: " + now.Format("2006-01"), "Spent so far:  ₹100", "Last month:    ₹300"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("forecast output missing %q:\n%s", expected, output)
		}
	}
}

func TestStreakCountsConsecutiveLowSpendFixtureDays(t *testing.T) {
	now := time.Now()
	setupAnalyticsFixture(t,
		fixtureTxn{"low-1", now.AddDate(0, 0, -3).Format(time.DateOnly), "OUTGOING", "Low", "", "CARD", 50},
		fixtureTxn{"low-2", now.AddDate(0, 0, -2).Format(time.DateOnly), "OUTGOING", "Low", "", "CARD", 50},
		fixtureTxn{"high", now.AddDate(0, 0, -1).Format(time.DateOnly), "OUTGOING", "High", "", "CARD", 200},
	)
	setFlag(t, streakCmd, "threshold", "100")
	setFlag(t, streakCmd, "lookback", "7")
	output := runAnalyticsCommand(t, streakCmd)
	for _, expected := range []string{"Threshold: ₹100/day", "Lookback:  7 days", "Current Streak:  0 days", "Longest Streak:  2 days"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("streak output missing %q:\n%s", expected, output)
		}
	}
}

func TestUnusualFlagsTransactionAboveMerchantAverage(t *testing.T) {
	setupAnalyticsFixture(t,
		fixtureTxn{"coffee-1", "2024-01-01", "OUTGOING", "Coffee Shop", "", "CARD", 100},
		fixtureTxn{"coffee-2", "2024-01-02", "OUTGOING", "Coffee Shop", "", "CARD", 100},
		fixtureTxn{"coffee-3", "2024-01-03", "OUTGOING", "Coffee Shop", "", "CARD", 100},
		fixtureTxn{"coffee-big", "2024-01-04", "OUTGOING", "Coffee Shop", "", "CARD", 1000},
	)
	setFlag(t, unusualCmd, "start", "2024-01-01")
	setFlag(t, unusualCmd, "end", "2024-01-31")
	setFlag(t, unusualCmd, "multiplier", "2")
	setFlag(t, unusualCmd, "min-history", "3")
	setFlag(t, unusualCmd, "limit", "10")
	output := runAnalyticsCommand(t, unusualCmd)
	for _, expected := range []string{"Unusual transactions (2.0x+ merchant avg):", "2024-01-04", "Coffee Shop", "₹1000", "3.1x"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("unusual output missing %q:\n%s", expected, output)
		}
	}
}
