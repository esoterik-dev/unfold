package cmd

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/esoterik-dev/unfold/db"
	"github.com/esoterik-dev/unfold/query"
)

// ─── Shared helpers ──────────────────────────────────────────────────────────

func ensureDB() {
	if db.Conn == nil {
		dbPath := viper.GetString("db-path")
		if dbPath == "" {
			dbPath = "db.sqlite"
		}
		db.Init(dbPath)
	}
}

func parseDateFlag(val, fallback string) time.Time {
	if val == "" {
		val = fallback
	}
	t, err := time.Parse(time.DateOnly, val)
	if err != nil {
		t = time.Now()
	}
	return t
}

func daysAgo(n int) string {
	return time.Now().AddDate(0, 0, -n).Format(time.DateOnly)
}

func today() string {
	return time.Now().Format(time.DateOnly)
}

func monthsAgo(n int) string {
	return time.Now().AddDate(0, -n, 0).Format(time.DateOnly)
}

func fmtAmt(n float64) string {
	abs := n
	if abs < 0 {
		abs = -abs
	}
	return fmt.Sprintf("₹%.0f", abs)
}

func printRow(cols ...interface{}) {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = fmt.Sprintf("%v", c)
	}
	fmt.Println(strings.Join(parts, "\t"))
}

type txnRow struct {
	UUID           string
	Amount         float64
	CurrentBalance float64
	Timestamp      time.Time
	Type           string
	Account        string
	Merchant       string
	Narration      string
	Category       string
	CategoryID     string
	Mode           string
	Summary        string
}

func queryTxns(startDate, endDate string, excludeTransfers bool) []txnRow {
	ensureDB()
	var rows []struct {
		UUID           string
		Amount         float64
		CurrentBalance float64
		Timestamp      time.Time
		Type           string
		Account        string
		Merchant       string
		Narration      string
		Category       string
		CategoryID     string
		Mode           string
		Summary        string
	}
	q := db.Conn.Model(&db.Transactions{}).Where("timestamp >= ? AND timestamp <= ?", startDate+" 00:00:00", endDate+" 23:59:59")
	if excludeTransfers {
		q = q.Where("category_id NOT IN (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"247e3e5d-59bf-4bf8-82ff-c152569893ea", // Self Transfer
			"a9c18a13-2b0a-4006-847a-83e16045870c", // Self Transfer
			"3cf94230-f5bf-4aa7-a3f0-bb0c4453413f", // Transfer / People
			"3d3a06cc-92cb-4d88-b9c9-4ce53de25c17", // Transfer / People
			"46ab2657-0e00-40cf-af5d-5d65e4e58152", // Transfer / People
			"6176feae-21c5-4874-9d9a-908a257e0ede", // Transfer / People
			"7505b993-d7fc-4461-9dc9-85c027a367ae", // Transfer / People
			"8281a1f2-df95-4299-97cc-c93b1e0a65ee", // Transfer / People
			"b3f7d021-7853-4ebd-836c-769f3f5539f0", // Transfer / People
			"f2d5dcf1-0c54-4f8d-957a-ae9b63e458f1", // Transfer / People
		)
	}
	q.Order("timestamp DESC").Find(&rows)
	result := make([]txnRow, len(rows))
	for i, r := range rows {
		result[i] = txnRow{
			UUID: r.UUID, Amount: r.Amount, CurrentBalance: r.CurrentBalance,
			Timestamp: r.Timestamp, Type: r.Type, Account: r.Account,
			Merchant: r.Merchant, Narration: r.Narration, Category: r.Category,
			CategoryID: r.CategoryID, Mode: r.Mode, Summary: r.Summary,
		}
	}
	return result
}

// ─── recent ──────────────────────────────────────────────────────────────────

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Query your local transaction database",
	Long:  `All subcommands read from the local SQLite database populated by 'unfold transactions -d'.`,
}

var recentCmd = &cobra.Command{
	Use:   "recent",
	Short: "Show the most recent transactions",
	Run: func(cmd *cobra.Command, args []string) {
		ensureDB()
		limit, _ := cmd.Flags().GetInt("limit")
		var rows []struct {
			UUID, Type, Account, Merchant, Narration string
			Amount, CurrentBalance                    float64
			Timestamp                                 time.Time
			CategoryID, Mode                          string
		}
		db.Conn.Model(&db.Transactions{}).Order("timestamp DESC").Limit(limit).Find(&rows)
		for _, r := range rows {
			merchant := r.Merchant
			if merchant == "" {
				merchant = r.Narration
			}
			cat := query.Categorize(r.CategoryID, r.Merchant)
			sign := "-"
			if r.Type == "INCOMING" {
				sign = "+"
			}
			fmt.Printf("%s | %s%s | %s | %s\n",
				r.Timestamp.Format(time.DateOnly),
				sign, fmtAmt(r.Amount),
				query.Truncate(query.CleanMerchant(merchant), 35),
				cat,
			)
		}
	},
}

// ─── search ──────────────────────────────────────────────────────────────────

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search transactions by merchant, narration, dates, amount, type, or mode",
	Run: func(cmd *cobra.Command, args []string) {
		ensureDB()
		q := db.Conn.Model(&db.Transactions{})
		if v, _ := cmd.Flags().GetString("query"); v != "" {
			q = q.Where("(merchant LIKE ? OR narration LIKE ? OR summary LIKE ?)", "%"+v+"%", "%"+v+"%", "%"+v+"%")
		}
		if v, _ := cmd.Flags().GetString("narration"); v != "" {
			q = q.Where("narration LIKE ?", "%"+v+"%")
		}
		if v, _ := cmd.Flags().GetString("summary"); v != "" {
			q = q.Where("summary LIKE ?", "%"+v+"%")
		}
		if v, _ := cmd.Flags().GetString("start"); v != "" {
			q = q.Where("timestamp >= ?", v+" 00:00:00")
		}
		if v, _ := cmd.Flags().GetString("end"); v != "" {
			q = q.Where("timestamp <= ?", v+" 23:59:59")
		}
		if v, _ := cmd.Flags().GetString("type"); v != "" {
			q = q.Where("type = ?", v)
		}
		if v, _ := cmd.Flags().GetString("mode"); v != "" {
			q = q.Where("mode = ?", v)
		}
		if v, _ := cmd.Flags().GetFloat64("min-amount"); v > 0 {
			q = q.Where("ABS(amount) >= ?", v)
		}
		if v, _ := cmd.Flags().GetFloat64("max-amount"); v > 0 {
			q = q.Where("ABS(amount) <= ?", v)
		}
		limit, _ := cmd.Flags().GetInt("limit")
		var rows []txnRow
		q.Order("timestamp DESC").Limit(limit).Find(&rows)
		for _, r := range rows {
			cat := query.Categorize(r.CategoryID, r.Merchant)
			sign := "-"
			if r.Type == "INCOMING" {
				sign = "+"
			}
			fmt.Printf("%s | %s%s | %s | %s\n",
				r.Timestamp.Format(time.DateOnly),
				sign, fmtAmt(r.Amount),
				query.Truncate(query.CleanMerchant(r.Merchant), 35),
				cat,
			)
		}
	},
}

// ─── spend-summary ───────────────────────────────────────────────────────────

var spendSummaryCmd = &cobra.Command{
	Use:   "spend-summary",
	Short: "Income vs spending breakdown with top merchants and avg daily spend",
	Run: func(cmd *cobra.Command, args []string) {
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		excl, _ := cmd.Flags().GetBool("exclude-transfers")
		if start == "" {
			start = daysAgo(30)
		}
		if end == "" {
			end = today()
		}
		txns := queryTxns(start, end, excl)

		var totalSpent, totalIncome float64
		merchantTotals := map[string]float64{}
		for _, t := range txns {
			if t.Type == "INCOMING" {
				totalIncome += t.Amount
			} else {
				totalSpent += t.Amount
				merchantTotals[t.Merchant] += t.Amount
			}
		}

		days := parseDateFlag(end, today()).Sub(parseDateFlag(start, daysAgo(30))).Hours()/24 + 1
		if days < 1 {
			days = 1
		}

		fmt.Printf("Period: %s to %s (%.0f days)\n", start, end, days)
		fmt.Printf("Total Spent:    %s\n", fmtAmt(totalSpent))
		fmt.Printf("Total Income:   %s\n", fmtAmt(totalIncome))
		fmt.Printf("Net:            %s\n", fmtAmt(totalIncome-totalSpent))
		fmt.Printf("Avg Daily Spend: %s\n\n", fmtAmt(totalSpent/days))

		// Top 5 merchants
		type kv struct {
			Key   string
			Value float64
		}
		var sorted []kv
		for k, v := range merchantTotals {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Value > sorted[j].Value })
		fmt.Println("Top Merchants:")
		for i := 0; i < len(sorted) && i < 5; i++ {
			fmt.Printf("  %s — %s\n", query.CleanMerchant(sorted[i].Key), fmtAmt(sorted[i].Value))
		}
	},
}

// ─── merchant-summary ────────────────────────────────────────────────────────

var merchantSummaryCmd = &cobra.Command{
	Use:   "merchant-summary",
	Short: "Top merchants by total spend or transaction frequency",
	Run: func(cmd *cobra.Command, args []string) {
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		limit, _ := cmd.Flags().GetInt("limit")
		sortBy, _ := cmd.Flags().GetString("sort")
		excl, _ := cmd.Flags().GetBool("exclude-transfers")
		if start == "" {
			start = daysAgo(30)
		}
		if end == "" {
			end = today()
		}
		txns := queryTxns(start, end, excl)

		type merchantStats struct {
			Total float64
			Count int
		}
		stats := map[string]*merchantStats{}
		for _, t := range txns {
			if t.Type != "INCOMING" {
				if _, ok := stats[t.Merchant]; !ok {
					stats[t.Merchant] = &merchantStats{}
				}
				stats[t.Merchant].Total += t.Amount
				stats[t.Merchant].Count++
			}
		}

		type kv struct {
			Key   string
			Total float64
			Count int
		}
		var sorted []kv
		for k, v := range stats {
			sorted = append(sorted, kv{k, v.Total, v.Count})
		}
		if sortBy == "count" {
			sort.Slice(sorted, func(i, j int) bool { return sorted[i].Count > sorted[j].Count })
		} else {
			sort.Slice(sorted, func(i, j int) bool { return sorted[i].Total > sorted[j].Total })
		}
		if limit > len(sorted) {
			limit = len(sorted)
		}
		fmt.Printf("%-35s %12s %6s\n", "Merchant", "Total", "Txns")
		fmt.Println(strings.Repeat("-", 56))
		for i := 0; i < limit; i++ {
			fmt.Printf("%-35s %12s %6d\n", query.CleanMerchant(sorted[i].Key), fmtAmt(sorted[i].Total), sorted[i].Count)
		}
	},
}

// ─── monthly-trend ──────────────────────────────────────────────────────────

var monthlyTrendCmd = &cobra.Command{
	Use:   "monthly-trend",
	Short: "Month-by-month income, spending, and net cash flow",
	Run: func(cmd *cobra.Command, args []string) {
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		excl, _ := cmd.Flags().GetBool("exclude-transfers")
		if start == "" {
			start = monthsAgo(12)
		}
		if end == "" {
			end = today()
		}
		txns := queryTxns(start, end, excl)

		type monthData struct {
			Income, Spending float64
		}
		months := map[string]*monthData{}
		for _, t := range txns {
			ym := t.Timestamp.Format("2006-01")
			if _, ok := months[ym]; !ok {
				months[ym] = &monthData{}
			}
			if t.Type == "INCOMING" {
				months[ym].Income += t.Amount
			} else {
				months[ym].Spending += t.Amount
			}
		}

		var keys []string
		for k := range months {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		fmt.Printf("%-10s %12s %12s %12s\n", "Month", "Income", "Spending", "Net")
		fmt.Println(strings.Repeat("-", 48))
		for _, k := range keys {
			m := months[k]
			fmt.Printf("%-10s %12s %12s %12s\n", k, fmtAmt(m.Income), fmtAmt(m.Spending), fmtAmt(m.Income-m.Spending))
		}
	},
}

// ─── balance-history ─────────────────────────────────────────────────────────

var balanceHistoryCmd = &cobra.Command{
	Use:   "balance-history",
	Short: "Average, min, and max account balance by month",
	Run: func(cmd *cobra.Command, args []string) {
		ensureDB()
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		if start == "" {
			start = monthsAgo(12)
		}
		if end == "" {
			end = today()
		}

		var rows []struct {
			Month    string
			Avg      float64
			Min      float64
			Max      float64
			Balance  float64
		}
		db.Conn.Raw(`
			SELECT strftime('%Y-%m', timestamp) as month,
				AVG(current_balance) as avg,
				MIN(current_balance) as minb,
				MAX(current_balance) as maxb,
				current_balance
			FROM transactions
			WHERE timestamp >= ? AND timestamp <= ?
			GROUP BY month ORDER BY month
		`, start+" 00:00:00", end+" 23:59:59").Scan(&rows)

		fmt.Printf("%-10s %12s %12s %12s\n", "Month", "Avg", "Min", "Max")
		fmt.Println(strings.Repeat("-", 48))
		for _, r := range rows {
			fmt.Printf("%-10s %12s %12s %12s\n", r.Month, fmtAmt(r.Avg), fmtAmt(r.Min), fmtAmt(r.Max))
		}
	},
}

// ─── mode-breakdown ─────────────────────────────────────────────────────────

var modeBreakdownCmd = &cobra.Command{
	Use:   "mode-breakdown",
	Short: "Spending and income by payment mode (UPI, CARD, etc.)",
	Run: func(cmd *cobra.Command, args []string) {
		ensureDB()
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		if start == "" {
			start = daysAgo(30)
		}
		if end == "" {
			end = today()
		}

		type modeData struct {
			Spending, Income float64
			Count            int
		}
		var rows []struct {
			Mode  string
			Type  string
			Total float64
			Count int
		}
		db.Conn.Raw(`
			SELECT mode, type, SUM(amount) as total, COUNT(*) as count
			FROM transactions WHERE timestamp >= ? AND timestamp <= ?
			GROUP BY mode, type ORDER BY total DESC
		`, start+" 00:00:00", end+" 23:59:59").Scan(&rows)

		modes := map[string]*modeData{}
		for _, r := range rows {
			if _, ok := modes[r.Mode]; !ok {
				modes[r.Mode] = &modeData{}
			}
			if r.Type == "INCOMING" {
				modes[r.Mode].Income += r.Total
			} else {
				modes[r.Mode].Spending += r.Total
			}
			modes[r.Mode].Count += r.Count
		}

		fmt.Printf("%-10s %12s %12s %6s\n", "Mode", "Spending", "Income", "Txns")
		fmt.Println(strings.Repeat("-", 42))
		for mode, d := range modes {
			fmt.Printf("%-10s %12s %12s %6d\n", mode, fmtAmt(d.Spending), fmtAmt(d.Income), d.Count)
		}
	},
}

// ─── recurring ───────────────────────────────────────────────────────────────

var recurringCmd = &cobra.Command{
	Use:   "recurring",
	Short: "Merchants appearing in N+ distinct months (subscriptions, bills, habits)",
	Run: func(cmd *cobra.Command, args []string) {
		ensureDB()
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		minMonths, _ := cmd.Flags().GetInt("min-months")
		limit, _ := cmd.Flags().GetInt("limit")
		if start == "" {
			start = monthsAgo(12)
		}
		if end == "" {
			end = today()
		}

		var rows []struct {
			Merchant   string
			DistMonths int
			Total      float64
			AvgPerMonth float64
		}
		db.Conn.Raw(`
			SELECT merchant, COUNT(DISTINCT strftime('%Y-%m', timestamp)) as dist_months,
				SUM(amount) as total, SUM(amount)/COUNT(DISTINCT strftime('%Y-%m', timestamp)) as avg_per_month
			FROM transactions
			WHERE timestamp >= ? AND timestamp <= ? AND type = 'OUTGOING'
			GROUP BY merchant HAVING dist_months >= ?
			ORDER BY total DESC LIMIT ?
		`, start+" 00:00:00", end+" 23:59:59", minMonths, limit).Scan(&rows)

		fmt.Printf("%-35s %7s %12s %12s\n", "Merchant", "Months", "Total", "Avg/Month")
		fmt.Println(strings.Repeat("-", 68))
		for _, r := range rows {
			fmt.Printf("%-35s %7d %12s %12s\n", query.CleanMerchant(r.Merchant), r.DistMonths, fmtAmt(r.Total), fmtAmt(r.AvgPerMonth))
		}
	},
}

// ─── account-breakdown ───────────────────────────────────────────────────────

var accountBreakdownCmd = &cobra.Command{
	Use:   "account-breakdown",
	Short: "Spending and income by bank account",
	Run: func(cmd *cobra.Command, args []string) {
		ensureDB()
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		if start == "" {
			start = daysAgo(30)
		}
		if end == "" {
			end = today()
		}

		var rows []struct {
			Account string
			Type    string
			Total   float64
			Count   int
		}
		db.Conn.Raw(`
			SELECT account, type, SUM(amount) as total, COUNT(*) as count
			FROM transactions WHERE timestamp >= ? AND timestamp <= ?
			GROUP BY account, type ORDER BY account, type
		`, start+" 00:00:00", end+" 23:59:59").Scan(&rows)

		type acctData struct {
			Spending, Income float64
			OutCount, InCount int
		}
		accounts := map[string]*acctData{}
		for _, r := range rows {
			if _, ok := accounts[r.Account]; !ok {
				accounts[r.Account] = &acctData{}
			}
			if r.Type == "INCOMING" {
				accounts[r.Account].Income += r.Total
				accounts[r.Account].InCount += r.Count
			} else {
				accounts[r.Account].Spending += r.Total
				accounts[r.Account].OutCount += r.Count
			}
		}

		fmt.Printf("%-20s %12s %12s\n", "Account", "Spending", "Income")
		fmt.Println(strings.Repeat("-", 46))
		for acct, d := range accounts {
			fmt.Printf("%-20s %12s %12s\n", query.Truncate(acct, 20), fmtAmt(d.Spending), fmtAmt(d.Income))
		}
	},
}

// ─── day-patterns ────────────────────────────────────────────────────────────

var dayPatternsCmd = &cobra.Command{
	Use:   "day-patterns",
	Short: "Spending by weekday or day-of-month",
	Run: func(cmd *cobra.Command, args []string) {
		ensureDB()
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		groupBy, _ := cmd.Flags().GetString("group")
		if start == "" {
			start = daysAgo(90)
		}
		if end == "" {
			end = today()
		}

		timeFmt := "%w"
		if groupBy == "monthday" {
			timeFmt = "%d"
		}

		var rows []struct {
			Period string
			Total  float64
			Count  int
		}
		db.Conn.Raw(`
			SELECT strftime('`+timeFmt+`', timestamp) as period, SUM(amount) as total, COUNT(*) as count
			FROM transactions WHERE timestamp >= ? AND timestamp <= ? AND type = 'OUTGOING'
			GROUP BY period ORDER BY period
		`, start+" 00:00:00", end+" 23:59:59").Scan(&rows)

		dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
		fmt.Printf("%-10s %12s %6s\n", "Period", "Total", "Txns")
		fmt.Println(strings.Repeat("-", 30))
		for _, r := range rows {
			label := r.Period
			if groupBy != "monthday" {
				idx, _ := strconv.Atoi(r.Period)
				if idx >= 0 && idx < 7 {
					label = dayNames[idx]
				}
			}
			fmt.Printf("%-10s %12s %6d\n", label, fmtAmt(r.Total), r.Count)
		}
	},
}

// ─── category ────────────────────────────────────────────────────────────────

var categoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Spending grouped by category (Food Delivery, Transport, Shopping, etc.)",
	Run: func(cmd *cobra.Command, args []string) {
		ensureDB()
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		if start == "" {
			start = daysAgo(30)
		}
		if end == "" {
			end = today()
		}

		var rows []struct {
			CategoryID string
			Total      float64
			Count      int
		}
		db.Conn.Raw(`
			SELECT category_id, SUM(amount) as total, COUNT(*) as count
			FROM transactions WHERE timestamp >= ? AND timestamp <= ? AND type = 'OUTGOING'
			GROUP BY category_id ORDER BY total DESC
		`, start+" 00:00:00", end+" 23:59:59").Scan(&rows)

		// Group by human category
		type catData struct {
			Total float64
			Count int
		}
		cats := map[string]*catData{}
		for _, r := range rows {
			cat := query.Categorize(r.CategoryID, "")
			if _, ok := cats[cat]; !ok {
				cats[cat] = &catData{}
			}
			cats[cat].Total += r.Total
			cats[cat].Count += r.Count
		}

		type kv struct {
			Name  string
			Total float64
			Count int
		}
		var sorted []kv
		for k, v := range cats {
			sorted = append(sorted, kv{k, v.Total, v.Count})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Total > sorted[j].Total })

		fmt.Printf("%-25s %12s %6s\n", "Category", "Total", "Txns")
		fmt.Println(strings.Repeat("-", 45))
		for _, s := range sorted {
			fmt.Printf("%-25s %12s %6d\n", s.Name, fmtAmt(s.Total), s.Count)
		}
	},
}

// ─── export-csv ──────────────────────────────────────────────────────────────

var exportCsvCmd = &cobra.Command{
	Use:   "export-csv",
	Short: "Export filtered transactions to a CSV file",
	Run: func(cmd *cobra.Command, args []string) {
		ensureDB()
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		output, _ := cmd.Flags().GetString("output")
		if start == "" {
			start = daysAgo(30)
		}
		if end == "" {
			end = today()
		}
		if output == "" {
			output = "transactions.csv"
		}

		txns := queryTxns(start, end, false)
		f, err := os.Create(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
			return
		}
		defer f.Close()
		w := csv.NewWriter(f)
		defer w.Flush()

		w.Write([]string{"Date", "UUID", "Amount", "Type", "Merchant", "Narration", "Category", "Mode", "Account", "Balance"})
		for _, t := range txns {
			w.Write([]string{
				t.Timestamp.Format(time.DateOnly),
				t.UUID,
				fmt.Sprintf("%.2f", t.Amount),
				t.Type,
				t.Merchant,
				t.Narration,
				query.Categorize(t.CategoryID, t.Merchant),
				t.Mode,
				t.Account,
				fmt.Sprintf("%.2f", t.CurrentBalance),
			})
		}
		fmt.Printf("Exported %d transactions to %s\n", len(txns), output)
	},
}

// ─── savings-rate ────────────────────────────────────────────────────────────

var savingsRateCmd = &cobra.Command{
	Use:   "savings-rate",
	Short: "Monthly savings rate with rolling 3-month average and trend",
	Run: func(cmd *cobra.Command, args []string) {
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		excl, _ := cmd.Flags().GetBool("exclude-transfers")
		if start == "" {
			start = monthsAgo(12)
		}
		if end == "" {
			end = today()
		}
		txns := queryTxns(start, end, excl)

		type monthData struct {
			Income, Spending float64
		}
		months := map[string]*monthData{}
		for _, t := range txns {
			ym := t.Timestamp.Format("2006-01")
			if _, ok := months[ym]; !ok {
				months[ym] = &monthData{}
			}
			if t.Type == "INCOMING" {
				months[ym].Income += t.Amount
			} else {
				months[ym].Spending += t.Amount
			}
		}

		var keys []string
		for k := range months {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		type rateEntry struct {
			Month  string
			Rate   float64
			Avg3   float64
		}
		var entries []rateEntry
		for _, k := range keys {
			m := months[k]
			rate := 0.0
			if m.Income > 0 {
				rate = (m.Income - m.Spending) / m.Income * 100
			}
			entries = append(entries, rateEntry{k, rate, 0})
		}

		// Rolling 3-month average
		for i := range entries {
			sum := 0.0
			count := 0
			for j := i - 2; j <= i; j++ {
				if j >= 0 {
					sum += entries[j].Rate
					count++
				}
			}
			if count > 0 {
				entries[i].Avg3 = sum / float64(count)
			}
		}

		fmt.Printf("%-10s %8s %10s\n", "Month", "Rate", "3M Avg")
		fmt.Println(strings.Repeat("-", 30))
		for _, e := range entries {
			fmt.Printf("%-10s %7.1f%% %9.1f%%\n", e.Month, e.Rate, e.Avg3)
		}
	},
}

// ─── weekly-digest ───────────────────────────────────────────────────────────

var weeklyDigestCmd = &cobra.Command{
	Use:   "weekly-digest",
	Short: "7-day spending summary with comparison to rolling average",
	Run: func(cmd *cobra.Command, args []string) {
		weeksBack, _ := cmd.Flags().GetInt("weeks-back")

		weekEnd := time.Now().AddDate(0, 0, -7*weeksBack)
		weekStart := weekEnd.AddDate(0, 0, -7)

		txns := queryTxns(weekStart.Format(time.DateOnly), weekEnd.Format(time.DateOnly), false)

		var totalSpent, totalIncome float64
		byDay := map[string]float64{}
		merchants := map[string]float64{}
		for _, t := range txns {
			if t.Type == "INCOMING" {
				totalIncome += t.Amount
			} else {
				totalSpent += t.Amount
				byDay[t.Timestamp.Format("Mon 02 Jan")] += t.Amount
				merchants[t.Merchant] += t.Amount
			}
		}

		// 3-week rolling avg
		prevEnd := weekStart
		prevStart := prevEnd.AddDate(0, 0, -21)
		prevTxns := queryTxns(prevStart.Format(time.DateOnly), prevEnd.Format(time.DateOnly), false)
		var prevSpent float64
		for _, t := range prevTxns {
			if t.Type != "INCOMING" {
				prevSpent += t.Amount
			}
		}
		prevAvg := prevSpent / 3.0

		fmt.Printf("Weekly Digest: %s to %s\n", weekStart.Format(time.DateOnly), weekEnd.Format(time.DateOnly))
		fmt.Printf("Spent:    %s\n", fmtAmt(totalSpent))
		fmt.Printf("Income:   %s\n", fmtAmt(totalIncome))
		fmt.Printf("3W Avg:   %s/week\n", fmtAmt(prevAvg))
		if prevAvg > 0 {
			pct := (totalSpent - prevAvg) / prevAvg * 100
			direction := "↑"
			if pct < 0 {
				direction = "↓"
			}
			fmt.Printf("vs Avg:   %s%.0f%%\n", direction, math.Abs(pct))
		}
		fmt.Println("\nDay-by-day:")
		var days []string
		for d := range byDay {
			days = append(days, d)
		}
		sort.Strings(days)
		for _, d := range days {
			fmt.Printf("  %s  %s\n", d, fmtAmt(byDay[d]))
		}

		fmt.Println("\nTop merchants:")
		type kv struct {
			Key   string
			Value float64
		}
		var sorted []kv
		for k, v := range merchants {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Value > sorted[j].Value })
		for i := 0; i < len(sorted) && i < 5; i++ {
			fmt.Printf("  %s — %s\n", query.CleanMerchant(sorted[i].Key), fmtAmt(sorted[i].Value))
		}
	},
}

// ─── tax-report ──────────────────────────────────────────────────────────────

var taxReportCmd = &cobra.Command{
	Use:   "tax-report",
	Short: "Full Indian financial year (Apr–Mar) income, spending, savings rate report",
	Run: func(cmd *cobra.Command, args []string) {
		fyYear, _ := cmd.Flags().GetInt("year")
		now := time.Now()
		if fyYear == 0 {
			if now.Month() >= 4 {
				fyYear = now.Year()
			} else {
				fyYear = now.Year() - 1
			}
		}
		start := fmt.Sprintf("%d-04-01", fyYear)
		end := fmt.Sprintf("%d-03-31", fyYear+1)

		txns := queryTxns(start, end, false)

		var totalSpent, totalIncome float64
		monthlyIncome := map[string]float64{}
		monthlySpending := map[string]float64{}
		merchants := map[string]float64{}

		for _, t := range txns {
			ym := t.Timestamp.Format("2006-01")
			if t.Type == "INCOMING" {
				totalIncome += t.Amount
				monthlyIncome[ym] += t.Amount
			} else {
				totalSpent += t.Amount
				monthlySpending[ym] += t.Amount
				merchants[t.Merchant] += t.Amount
			}
		}

		savingsRate := 0.0
		if totalIncome > 0 {
			savingsRate = (totalIncome - totalSpent) / totalIncome * 100
		}

		fmt.Printf("Tax Year Report: FY %d-%d (%s to %s)\n", fyYear, fyYear+1, start, end)
		fmt.Printf("Total Income:  %s\n", fmtAmt(totalIncome))
		fmt.Printf("Total Spent:   %s\n", fmtAmt(totalSpent))
		fmt.Printf("Net Savings:   %s\n", fmtAmt(totalIncome-totalSpent))
		fmt.Printf("Savings Rate:  %.1f%%\n\n", savingsRate)

		// Monthly breakdown
		fmt.Println("Monthly Breakdown:")
		fmt.Printf("%-10s %12s %12s %12s\n", "Month", "Income", "Spending", "Net")
		fmt.Println(strings.Repeat("-", 48))
		var months []string
		for m := range monthlyIncome {
			months = append(months, m)
		}
		for m := range monthlySpending {
			if _, ok := monthlyIncome[m]; !ok {
				months = append(months, m)
			}
		}
		sort.Strings(months)
		for _, m := range months {
			inc := monthlyIncome[m]
			spend := monthlySpending[m]
			fmt.Printf("%-10s %12s %12s %12s\n", m, fmtAmt(inc), fmtAmt(spend), fmtAmt(inc-spend))
		}

		// Top 10 merchants
		type kv struct {
			Key   string
			Value float64
		}
		var sorted []kv
		for k, v := range merchants {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Value > sorted[j].Value })
		fmt.Println("\nTop 10 Merchants:")
		for i := 0; i < len(sorted) && i < 10; i++ {
			fmt.Printf("  %s — %s\n", query.CleanMerchant(sorted[i].Key), fmtAmt(sorted[i].Value))
		}
	},
}

// ─── unusual ─────────────────────────────────────────────────────────────────

var unusualCmd = &cobra.Command{
	Use:   "unusual",
	Short: "Find transactions unusually large compared to your normal spend at that merchant",
	Run: func(cmd *cobra.Command, args []string) {
		ensureDB()
		start, _ := cmd.Flags().GetString("start")
		end, _ := cmd.Flags().GetString("end")
		multiplier, _ := cmd.Flags().GetFloat64("multiplier")
		minHistory, _ := cmd.Flags().GetInt("min-history")
		limit, _ := cmd.Flags().GetInt("limit")
		if start == "" {
			start = daysAgo(90)
		}
		if end == "" {
			end = today()
		}

		// Get merchant average stats
		type stats struct {
			Avg   float64
			Count int
		}
		var statRows []struct {
			Merchant string
			Avg      float64
			Count    int
		}
		db.Conn.Raw(`
			SELECT merchant, AVG(amount) as avg, COUNT(*) as count
			FROM transactions WHERE type = 'OUTGOING'
			GROUP BY merchant HAVING count >= ?
		`, minHistory).Scan(&statRows)
		merchantStats := map[string]stats{}
		for _, r := range statRows {
			merchantStats[r.Merchant] = stats{r.Avg, r.Count}
		}

		// Find unusual in date range
		var txns []txnRow
		db.Conn.Model(&db.Transactions{}).Where("timestamp >= ? AND timestamp <= ? AND type = 'OUTGOING'",
			start+" 00:00:00", end+" 23:59:59").Order("amount DESC").Find(&txns)

		type flaggedTxn struct {
			Date     string
			Merchant string
			Amount   float64
			Avg      float64
			Ratio    float64
		}
		var results []flaggedTxn
		for _, t := range txns {
			if s, ok := merchantStats[t.Merchant]; ok {
				ratio := t.Amount / s.Avg
				if ratio >= multiplier {
					results = append(results, flaggedTxn{
						Date: t.Timestamp.Format(time.DateOnly), Merchant: t.Merchant,
						Amount: t.Amount, Avg: s.Avg, Ratio: ratio,
					})
				}
			}
			if len(results) >= limit {
				break
			}
		}

		fmt.Printf("Unusual transactions (%sx+ merchant avg):\n\n", fmt.Sprintf("%.1f", multiplier))
		fmt.Printf("%-12s %-30s %12s %12s %6s\n", "Date", "Merchant", "Amount", "Avg", "Ratio")
		fmt.Println(strings.Repeat("-", 74))
		for _, f := range results {
			fmt.Printf("%-12s %-30s %12s %12s %5.1fx\n", f.Date, query.CleanMerchant(f.Merchant), fmtAmt(f.Amount), fmtAmt(f.Avg), f.Ratio)
		}
	},
}

// ─── compare ─────────────────────────────────────────────────────────────────

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Compare spending side-by-side across two date ranges",
	Run: func(cmd *cobra.Command, args []string) {
		p1s, _ := cmd.Flags().GetString("p1-start")
		p1e, _ := cmd.Flags().GetString("p1-end")
		p2s, _ := cmd.Flags().GetString("p2-start")
		p2e, _ := cmd.Flags().GetString("p2-end")
		excl, _ := cmd.Flags().GetBool("exclude-transfers")
		now := time.Now()
		if p1s == "" {
			p1s = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format(time.DateOnly)
		}
		if p1e == "" {
			p1e = today()
		}
		if p2s == "" {
			p2s = now.AddDate(0, -1, 0).Format(time.DateOnly)
		}
		if p2e == "" {
			p2e = today()
		}

		type periodStats struct {
			Spending, Income, AvgDaily float64
			Days                       float64
		}
		calc := func(start, end string) periodStats {
			txns := queryTxns(start, end, excl)
			var s, i float64
			for _, t := range txns {
				if t.Type == "INCOMING" {
					i += t.Amount
				} else {
					s += t.Amount
				}
			}
			d := parseDateFlag(end, today()).Sub(parseDateFlag(start, daysAgo(30))).Hours()/24 + 1
			avg := 0.0
			if d > 0 {
				avg = s / d
			}
			return periodStats{s, i, avg, d}
		}

		p1 := calc(p1s, p1e)
		p2 := calc(p2s, p2e)

		fmt.Printf("Period 1: %s to %s (%.0f days)\n", p1s, p1e, p1.Days)
		fmt.Printf("Period 2: %s to %s (%.0f days)\n\n", p2s, p2e, p2.Days)

		pctChange := func(a, b float64) string {
			if b == 0 {
				return "N/A"
			}
			pct := (a - b) / b * 100
			if pct > 0 {
				return fmt.Sprintf("+%.1f%%", pct)
			}
			return fmt.Sprintf("%.1f%%", pct)
		}

		fmt.Printf("%-15s %15s %15s %10s\n", "", "Period 1", "Period 2", "Change")
		fmt.Println(strings.Repeat("-", 56))
		fmt.Printf("%-15s %15s %15s %10s\n", "Spending", fmtAmt(p1.Spending), fmtAmt(p2.Spending), pctChange(p1.Spending, p2.Spending))
		fmt.Printf("%-15s %15s %15s %10s\n", "Income", fmtAmt(p1.Income), fmtAmt(p2.Income), pctChange(p1.Income, p2.Income))
		fmt.Printf("%-15s %15s %15s %10s\n", "Net", fmtAmt(p1.Income-p1.Spending), fmtAmt(p2.Income-p2.Spending), "")
		fmt.Printf("%-15s %15s %15s %10s\n", "Avg Daily", fmtAmt(p1.AvgDaily), fmtAmt(p2.AvgDaily), pctChange(p1.AvgDaily, p2.AvgDaily))
	},
}

// ─── forecast ────────────────────────────────────────────────────────────────

var forecastCmd = &cobra.Command{
	Use:   "forecast",
	Short: "Projected month-end spend based on current pace vs last month",
	Run: func(cmd *cobra.Command, args []string) {
		excl, _ := cmd.Flags().GetBool("exclude-transfers")
		now := time.Now()
		thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		lastMonthStart := thisMonthStart.AddDate(0, -1, 0)
		lastMonthEnd := thisMonthStart.Add(-time.Second)

		thisTxns := queryTxns(thisMonthStart.Format(time.DateOnly), today(), excl)
		lastTxns := queryTxns(lastMonthStart.Format(time.DateOnly), lastMonthEnd.Format(time.DateOnly), excl)

		var thisSpent, lastSpent float64
		for _, t := range thisTxns {
			if t.Type != "INCOMING" {
				thisSpent += t.Amount
			}
		}
		for _, t := range lastTxns {
			if t.Type != "INCOMING" {
				lastSpent += t.Amount
			}
		}

		daysInMonth := float64(time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day())
		daysSoFar := float64(now.Day())

		dailyPace := thisSpent / daysSoFar
		projected := dailyPace * daysInMonth

		fmt.Printf("Month: %s\n", now.Format("2006-01"))
		fmt.Printf("Spent so far:  %s (%.0f/%.0f days)\n", fmtAmt(thisSpent), daysSoFar, daysInMonth)
		fmt.Printf("Daily pace:    %s/day\n", fmtAmt(dailyPace))
		fmt.Printf("Projected:     %s\n", fmtAmt(projected))
		fmt.Printf("Last month:    %s\n", fmtAmt(lastSpent))
		if lastSpent > 0 {
			pct := (projected - lastSpent) / lastSpent * 100
			fmt.Printf("vs Last Month: %+.1f%%\n", pct)
		}
	},
}

// ─── streak ──────────────────────────────────────────────────────────────────

var streakCmd = &cobra.Command{
	Use:   "streak",
	Short: "Track your current and longest streak of low-spend days",
	Run: func(cmd *cobra.Command, args []string) {
		ensureDB()
		threshold, _ := cmd.Flags().GetFloat64("threshold")
		lookback, _ := cmd.Flags().GetInt("lookback")

		start := daysAgo(lookback)
		var rows []struct {
			Date   string
			Total  float64
		}
		db.Conn.Raw(`
			SELECT date(timestamp) as date, SUM(amount) as total
			FROM transactions WHERE timestamp >= ? AND type = 'OUTGOING'
			GROUP BY date ORDER BY date
		`, start+" 00:00:00").Scan(&rows)

		type daySpend struct {
			Date  string
			Total float64
		}
		days := []daySpend{}
		for _, r := range rows {
			days = append(days, daySpend{r.Date, r.Total})
		}

		// Current streak (from most recent day backwards)
		currentStreak := 0
		for i := len(days) - 1; i >= 0; i-- {
			if days[i].Total <= threshold {
				currentStreak++
			} else {
				break
			}
		}

		// Longest streak
		longestStreak := 0
		streak := 0
		for _, d := range days {
			if d.Total <= threshold {
				streak++
				if streak > longestStreak {
					longestStreak = streak
				}
			} else {
				streak = 0
			}
		}

		fmt.Printf("Threshold: %s/day\n", fmtAmt(threshold))
		fmt.Printf("Lookback:  %d days\n", lookback)
		fmt.Printf("Current Streak:  %d days\n", currentStreak)
		fmt.Printf("Longest Streak:  %d days\n", longestStreak)
	},
}

// ─── Register all query commands ─────────────────────────────────────────────

func init() {
	// recent
	recentCmd.Flags().IntP("limit", "n", 20, "Number of transactions to show")
	dbCmd.AddCommand(recentCmd)

	// search
	searchCmd.Flags().StringP("query", "q", "", "Text search on merchant/narration/summary")
	searchCmd.Flags().String("narration", "", "Text search on raw bank narration")
	searchCmd.Flags().String("summary", "", "Text search on summary")
	searchCmd.Flags().String("start", "", "Start date (YYYY-MM-DD)")
	searchCmd.Flags().String("end", "", "End date (YYYY-MM-DD)")
	searchCmd.Flags().String("type", "", "Filter by type: INCOMING or OUTGOING")
	searchCmd.Flags().String("mode", "", "Filter by payment mode: UPI, CARD, etc.")
	searchCmd.Flags().Float64("min-amount", 0, "Minimum amount")
	searchCmd.Flags().Float64("max-amount", 0, "Maximum amount")
	searchCmd.Flags().IntP("limit", "n", 50, "Max results")
	dbCmd.AddCommand(searchCmd)

	// spend-summary
	spendSummaryCmd.Flags().String("start", "", "Start date (default: 30 days ago)")
	spendSummaryCmd.Flags().String("end", "", "End date (default: today)")
	spendSummaryCmd.Flags().Bool("exclude-transfers", false, "Exclude internal transfers")
	dbCmd.AddCommand(spendSummaryCmd)

	// merchant-summary
	merchantSummaryCmd.Flags().String("start", "", "Start date")
	merchantSummaryCmd.Flags().String("end", "", "End date")
	merchantSummaryCmd.Flags().IntP("limit", "n", 10, "Number of top merchants")
	merchantSummaryCmd.Flags().String("sort", "amount", "Sort by 'amount' or 'count'")
	merchantSummaryCmd.Flags().Bool("exclude-transfers", false, "Exclude internal transfers")
	dbCmd.AddCommand(merchantSummaryCmd)

	// monthly-trend
	monthlyTrendCmd.Flags().String("start", "", "Start date (default: 12 months ago)")
	monthlyTrendCmd.Flags().String("end", "", "End date")
	monthlyTrendCmd.Flags().Bool("exclude-transfers", false, "Exclude internal transfers")
	dbCmd.AddCommand(monthlyTrendCmd)

	// balance-history
	balanceHistoryCmd.Flags().String("start", "", "Start date (default: 12 months ago)")
	balanceHistoryCmd.Flags().String("end", "", "End date")
	dbCmd.AddCommand(balanceHistoryCmd)

	// mode-breakdown
	modeBreakdownCmd.Flags().String("start", "", "Start date")
	modeBreakdownCmd.Flags().String("end", "", "End date")
	dbCmd.AddCommand(modeBreakdownCmd)

	// recurring
	recurringCmd.Flags().String("start", "", "Start date")
	recurringCmd.Flags().String("end", "", "End date")
	recurringCmd.Flags().Int("min-months", 3, "Minimum distinct months")
	recurringCmd.Flags().IntP("limit", "n", 20, "Max results")
	dbCmd.AddCommand(recurringCmd)

	// account-breakdown
	accountBreakdownCmd.Flags().String("start", "", "Start date")
	accountBreakdownCmd.Flags().String("end", "", "End date")
	dbCmd.AddCommand(accountBreakdownCmd)

	// day-patterns
	dayPatternsCmd.Flags().String("start", "", "Start date")
	dayPatternsCmd.Flags().String("end", "", "End date")
	dayPatternsCmd.Flags().StringP("group", "g", "weekday", "Group by 'weekday' or 'monthday'")
	dbCmd.AddCommand(dayPatternsCmd)

	// category
	categoryCmd.Flags().String("start", "", "Start date")
	categoryCmd.Flags().String("end", "", "End date")
	dbCmd.AddCommand(categoryCmd)

	// export-csv
	exportCsvCmd.Flags().String("start", "", "Start date")
	exportCsvCmd.Flags().String("end", "", "End date")
	exportCsvCmd.Flags().StringP("output", "o", "transactions.csv", "Output file path")
	dbCmd.AddCommand(exportCsvCmd)

	// savings-rate
	savingsRateCmd.Flags().String("start", "", "Start date")
	savingsRateCmd.Flags().String("end", "", "End date")
	savingsRateCmd.Flags().Bool("exclude-transfers", false, "Exclude internal transfers")
	dbCmd.AddCommand(savingsRateCmd)

	// weekly-digest
	weeklyDigestCmd.Flags().Int("weeks-back", 0, "Weeks back (0=last 7 days)")
	dbCmd.AddCommand(weeklyDigestCmd)

	// tax-report
	taxReportCmd.Flags().Int("year", 0, "FY start year (default: current/recent)")
	dbCmd.AddCommand(taxReportCmd)

	// unusual
	unusualCmd.Flags().String("start", "", "Start date (default: 90 days ago)")
	unusualCmd.Flags().String("end", "", "End date")
	unusualCmd.Flags().Float64("multiplier", 2.5, "Flag transactions Nx above average")
	unusualCmd.Flags().Int("min-history", 3, "Minimum past transactions at merchant")
	unusualCmd.Flags().IntP("limit", "n", 20, "Max results")
	dbCmd.AddCommand(unusualCmd)

	// compare
	compareCmd.Flags().String("p1-start", "", "Period 1 start (default: first of this month)")
	compareCmd.Flags().String("p1-end", "", "Period 1 end (default: today)")
	compareCmd.Flags().String("p2-start", "", "Period 2 start (default: same day last month)")
	compareCmd.Flags().String("p2-end", "", "Period 2 end (default: today)")
	compareCmd.Flags().Bool("exclude-transfers", false, "Exclude internal transfers")
	dbCmd.AddCommand(compareCmd)

	// forecast
	forecastCmd.Flags().Bool("exclude-transfers", false, "Exclude internal transfers")
	dbCmd.AddCommand(forecastCmd)

	// streak
	streakCmd.Flags().Float64("threshold", 1000, "Daily spending limit in ₹")
	streakCmd.Flags().Int("lookback", 90, "Days to look back")
	dbCmd.AddCommand(streakCmd)
}
