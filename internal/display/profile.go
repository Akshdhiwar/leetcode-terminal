package display

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/user/leetcode-cli/internal/api"
)

// ─── ANSI helpers (local) ─────────────────────────────────────────────────────

func stripANSI(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// visLen returns the visible (printable) character width of a string,
// ignoring ANSI escape codes and counting each rune as width 1.
func visLen(s string) int {
	return len([]rune(stripANSI(s)))
}

// pad pads/truncates s to exactly `width` visible chars.
func pad(s string, width int) string {
	v := visLen(s)
	if v >= width {
		// truncate visible chars
		clean := stripANSI(s)
		r := []rune(clean)
		if len(r) > width {
			return string(r[:width])
		}
		return s
	}
	return s + strings.Repeat(" ", width-v)
}

// ─── Bento card helpers ───────────────────────────────────────────────────────

// card renders a titled box. lines are the content rows (no padding needed).
func card(title string, innerW int, lines []string) {
	top := color(Dim, "┌─ ") + bold(color(BrightWhite, title)) + " " + color(Dim, strings.Repeat("─", max(0, innerW-visLen(title)-4)))  + color(Dim, "┐")
	bot := color(Dim, "└"+strings.Repeat("─", innerW+2)+"┘")
	fmt.Println("  " + top)
	for _, l := range lines {
		vis := visLen(l)
		pad2 := max(0, innerW-vis)
		fmt.Printf("  %s %s%s %s\n", color(Dim, "│"), l, strings.Repeat(" ", pad2), color(Dim, "│"))
	}
	fmt.Println("  " + bot)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// twoCol prints two cards side by side. Left and right inner widths must sum to terminal space.
func twoCol(leftTitle string, leftW int, leftLines []string, rightTitle string, rightW int, rightLines []string) {
	// Ensure same height
	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, "")
	}

	leftTop := color(Dim, "┌─ ") + bold(color(BrightWhite, leftTitle)) + " " +
		color(Dim, strings.Repeat("─", max(0, leftW-visLen(leftTitle)-4))) + color(Dim, "┐")
	rightTop := color(Dim, "┌─ ") + bold(color(BrightWhite, rightTitle)) + " " +
		color(Dim, strings.Repeat("─", max(0, rightW-visLen(rightTitle)-4))) + color(Dim, "┐")
	leftBot := color(Dim, "└"+strings.Repeat("─", leftW+2)+"┘")
	rightBot := color(Dim, "└"+strings.Repeat("─", rightW+2)+"┘")

	fmt.Printf("  %s   %s\n", leftTop, rightTop)
	for i := range leftLines {
		lv := visLen(leftLines[i])
		rv := visLen(rightLines[i])
		lpad := max(0, leftW-lv)
		rpad := max(0, rightW-rv)
		fmt.Printf("  %s %s%s %s   %s %s%s %s\n",
			color(Dim, "│"), leftLines[i], strings.Repeat(" ", lpad), color(Dim, "│"),
			color(Dim, "│"), rightLines[i], strings.Repeat(" ", rpad), color(Dim, "│"),
		)
	}
	fmt.Printf("  %s   %s\n", leftBot, rightBot)
}

// ─── Progress bars ────────────────────────────────────────────────────────────

func progressBar(filled, total, width int, clr string) string {
	pct := 0.0
	if total > 0 {
		pct = float64(filled) / float64(total)
	}
	f := int(math.Round(float64(width) * pct))
	if f > width {
		f = width
	}
	return color(clr, strings.Repeat("█", f)) + color(Dim, strings.Repeat("░", width-f))
}

func miniBar(val, maxVal, width int, clr string) string {
	f := 0
	if maxVal > 0 {
		f = int(math.Round(float64(width) * float64(val) / float64(maxVal)))
	}
	if f > width {
		f = width
	}
	return color(clr, strings.Repeat("▇", f)) + color(Dim, strings.Repeat("░", width-f))
}

// ─── Main PrintProfile ────────────────────────────────────────────────────────

func PrintProfile(p *api.UserProfile, cal map[string]int) {
	u := p.MatchedUser
	fmt.Println()

	// Collect solved/total maps
	solvedMap := map[string]int{}
	submMap := map[string]int{}
	for _, s := range u.SubmitStats.AcSubmissionNum {
		solvedMap[s.Difficulty] = s.Count
		submMap[s.Difficulty] = s.Submissions
	}
	totalSubm := 0
	for _, s := range u.SubmitStats.TotalSubmissionNum {
		totalSubm += s.Count
	}
	totalMap := map[string]int{}
	for _, t := range p.AllQuestionsCount {
		totalMap[t.Difficulty] = t.Count
	}
	allSolved := solvedMap["All"]
	allTotal := totalMap["All"]
	if allTotal == 0 {
		allTotal = totalMap["Easy"] + totalMap["Medium"] + totalMap["Hard"]
	}

	// ── Row 1: Identity + Solve Summary (two columns) ─────────────────────────
	name := u.Profile.RealName
	if name == "" {
		name = u.Username
	}

	// Left: identity card (width 28)
	idW := 28
	var idLines []string
	idLines = append(idLines, bold(color(BrightCyan, name)))
	idLines = append(idLines, color(Dim, "@"+u.Username))
	idLines = append(idLines, "")
	if u.Profile.CountryName != "" {
		idLines = append(idLines, color(Dim, "📍 ")+u.Profile.CountryName)
	}
	if u.Profile.Company != "" {
		idLines = append(idLines, color(Dim, "🏢 ")+u.Profile.Company)
	}
	if u.Profile.School != "" {
		idLines = append(idLines, color(Dim, "🎓 ")+u.Profile.School)
	}
	if u.Profile.Ranking > 0 {
		idLines = append(idLines, color(Dim, "Rank  ")+bold(color(BrightWhite, "#"+strconv.Itoa(u.Profile.Ranking))))
	}
	idLines = append(idLines, color(Dim, "Rep   ")+color(BrightYellow, strconv.Itoa(u.Profile.Reputation)))

	// Right: solve summary card (width 34)
	solW := 34
	pct := 0.0
	if allTotal > 0 {
		pct = float64(allSolved) / float64(allTotal) * 100
	}
	bigBar := progressBar(allSolved, allTotal, 24, BrightGreen)
	var solLines []string
	solLines = append(solLines, bold(color(BrightGreen, strconv.Itoa(allSolved)))+
		color(Dim, " / "+strconv.Itoa(allTotal)+" solved")+
		color(Dim, fmt.Sprintf("  %.1f%%", pct)))
	solLines = append(solLines, bigBar)
	solLines = append(solLines, "")

	tiers := []struct{ label, diff, clr string }{
		{"Easy  ", "Easy", BrightGreen},
		{"Medium", "Medium", BrightYellow},
		{"Hard  ", "Hard", BrightRed},
	}
	for _, t := range tiers {
		sv := solvedMap[t.diff]
		tv := totalMap[t.diff]
		bar := progressBar(sv, tv, 12, t.clr)
		acr := ""
		if submMap[t.diff] > 0 {
			acr = fmt.Sprintf("%.0f%%", float64(sv)/float64(submMap[t.diff])*100)
		}
		row := color(t.clr, t.label) + "  " + bar +
			"  " + color(t.clr, fmt.Sprintf("%d", sv)) + color(Dim, "/"+strconv.Itoa(tv)) +
			"  " + color(Dim, acr)
		solLines = append(solLines, row)
	}
	solLines = append(solLines, "")
	solLines = append(solLines, color(Dim, "Total submissions: ")+color(BrightCyan, strconv.Itoa(totalSubm)))

	twoCol("Profile", idW, idLines, "Solving Stats", solW, solLines)
	fmt.Println()

	// ── Row 2: Contest + Languages (two columns) ──────────────────────────────
	contW := 28
	var contLines []string
	if p.UserContestRanking != nil {
		cr := p.UserContestRanking
		ratingClr := BrightCyan
		if cr.Rating >= 1600 {
			ratingClr = BrightGreen
		}
		if cr.Rating >= 2000 {
			ratingClr = BrightYellow
		}
		if cr.Rating >= 2400 {
			ratingClr = BrightRed
		}
		rBar := miniBar(int(cr.Rating), 3000, 18, ratingClr)
		contLines = append(contLines,
			color(Dim, "Rating  ")+rBar+" "+bold(color(ratingClr, fmt.Sprintf("%.0f", cr.Rating))))
		contLines = append(contLines, "")
		contLines = append(contLines,
			color(Dim, "Rank    ")+bold(color(BrightWhite, "#"+strconv.Itoa(cr.GlobalRanking))))
		contLines = append(contLines,
			color(Dim, "Top     ")+color(BrightCyan, fmt.Sprintf("%.1f%%", cr.TopPercentage)))
		contLines = append(contLines,
			color(Dim, "Contests ")+color(BrightWhite, strconv.Itoa(cr.AttendedContestsCount)))
	} else {
		contLines = append(contLines, color(Dim, "No contest data"))
	}

	// Badges in same left column area
	if len(u.Badges) > 0 || (u.ContestBadge != nil && !u.ContestBadge.Expired) {
		contLines = append(contLines, "")
		contLines = append(contLines, color(Dim, "── Badges ──────────────"))
		for _, b := range u.Badges {
			contLines = append(contLines, color(BrightYellow, "🏅 ")+b.Name)
		}
		if u.ContestBadge != nil && !u.ContestBadge.Expired {
			contLines = append(contLines, color(BrightYellow, "🏆 ")+u.ContestBadge.Name)
		}
	}

	// Right: Languages
	langW := 34
	var langLines []string
	langs := u.LanguageProblemCount
	sort.Slice(langs, func(i, j int) bool {
		return langs[i].ProblemsSolved > langs[j].ProblemsSolved
	})
	maxLang := 1
	for _, l := range langs {
		if l.ProblemsSolved > maxLang {
			maxLang = l.ProblemsSolved
		}
	}
	for _, l := range langs {
		if l.ProblemsSolved == 0 {
			continue
		}
		bar := miniBar(l.ProblemsSolved, maxLang, 16, BrightBlue)
		name := pad(l.LanguageName, 12)
		langLines = append(langLines,
			color(BrightCyan, name)+" "+bar+" "+color(Dim, strconv.Itoa(l.ProblemsSolved)))
	}
	if len(langLines) == 0 {
		langLines = append(langLines, color(Dim, "No language data"))
	}

	twoCol("Contest", contW, contLines, "Languages", langW, langLines)
	fmt.Println()

	// ── Row 3: Submission Heatmap (full width) ────────────────────────────────
	printHeatmap(cal)
}

// ─── Heatmap ──────────────────────────────────────────────────────────────────

func printHeatmap(cal map[string]int) {
	// Build lookup: normalize all cal keys to midnight UTC
	// LeetCode stores unix timestamps at start of day UTC
	normalized := map[int64]int{}
	for k, v := range cal {
		ts, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			continue
		}
		// Normalize to midnight UTC by flooring to day boundary
		t := time.Unix(ts, 0).UTC()
		midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
		normalized[midnight] += v
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Build 53 weeks grid ending at today
	// Start from Sunday 52 full weeks before this week's Sunday
	weekStart := today
	for weekStart.Weekday() != time.Sunday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekStart = weekStart.AddDate(0, 0, -52*7)

	type cell struct {
		date  time.Time
		count int
	}
	var weeks [][]cell
	cur := weekStart
	for !cur.After(today) {
		week := make([]cell, 7)
		for d := 0; d < 7; d++ {
			midnight := time.Date(cur.Year(), cur.Month(), cur.Day(), 0, 0, 0, 0, time.UTC).Unix()
			week[d] = cell{date: cur, count: normalized[midnight]}
			cur = cur.AddDate(0, 0, 1)
		}
		weeks = append(weeks, week)
	}

	// Compute stats
	totalSubs := 0
	activeDays := 0
	for _, w := range weeks {
		for _, c := range w {
			if !c.date.After(today) {
				totalSubs += c.count
				if c.count > 0 {
					activeDays++
				}
			}
		}
	}
	// Current streak: walk backwards from today
	streak := 0
	d := today
	for {
		ts := d.Unix()
		if normalized[ts] > 0 {
			streak++
			d = d.AddDate(0, 0, -1)
		} else {
			break
		}
	}

	// Max for color scaling
	maxCount := 1
	for _, w := range weeks {
		for _, c := range w {
			if c.count > maxCount {
				maxCount = c.count
			}
		}
	}

	// ── Render ────────────────────────────────────────────────────────────────
	innerW := len(weeks) + 6 // approx
	heatTitle := fmt.Sprintf("Submission Heatmap — %s subm · %s active days · %s streak",
		color(BrightGreen, strconv.Itoa(totalSubs)),
		color(BrightCyan, strconv.Itoa(activeDays)),
		color(BrightYellow, strconv.Itoa(streak)+" days"),
	)

	topBorder := color(Dim, "┌─ ") + bold(color(BrightWhite, "Submission Activity")) + " " +
		color(Dim, strings.Repeat("─", max(0, innerW-20))) + color(Dim, "┐")
	fmt.Println("  " + topBorder)

	// Stats line inside card
	statsLine := fmt.Sprintf("  %s subm  ·  %s active days  ·  %s day streak",
		color(BrightGreen, strconv.Itoa(totalSubs)),
		color(BrightCyan, strconv.Itoa(activeDays)),
		color(BrightYellow, strconv.Itoa(streak)),
	)
	fmt.Printf("  %s %s\n", color(Dim, "│"), statsLine)
	fmt.Printf("  %s\n", color(Dim, "│"))

	// Month labels
	fmt.Printf("  %s  %s  ", color(Dim, "│"), color(Dim, "    ")) // day-label indent
	prevMonth := -1
	for _, w := range weeks {
		firstDay := w[0].date
		m := int(firstDay.Month())
		if firstDay.Day() <= 7 && m != prevMonth {
			label := firstDay.Format("Jan")
			fmt.Printf("%s", color(Dim, label))
			prevMonth = m
		} else {
			fmt.Print(" ")
		}
	}
	fmt.Println()

	// Day rows
	dayLabels := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	for d := 0; d < 7; d++ {
		lbl := "   "
		if d == 1 || d == 3 || d == 5 {
			lbl = dayLabels[d]
		}
		fmt.Printf("  %s  %s  ", color(Dim, "│"), color(Dim, lbl))
		for _, w := range weeks {
			c := w[d]
			if c.date.After(today) {
				fmt.Print(" ")
				continue
			}
			fmt.Print(heatBlock(c.count, maxCount))
		}
		fmt.Println()
	}

	// Legend
	fmt.Printf("  %s  %s ", color(Dim, "│"), color(Dim, "Less "))
	for i := 0; i <= 4; i++ {
		fmt.Print(heatBlock(i*maxCount/4, maxCount))
	}
	fmt.Printf(" %s\n", color(Dim, "More"))
	fmt.Printf("  %s\n", color(Dim, "└"+strings.Repeat("─", innerW+2)+"┘"))
	fmt.Println()

	_ = heatTitle // used for length calc
}

func heatBlock(count, maxCount int) string {
	if count == 0 {
		return color("\033[38;5;236m", "░")
	}
	pct := float64(count) / float64(maxCount)
	switch {
	case pct <= 0.25:
		return color("\033[38;5;22m", "▓") // darkest green
	case pct <= 0.5:
		return color("\033[38;5;28m", "▓") // mid green
	case pct <= 0.75:
		return color("\033[38;5;34m", "█") // bright green
	default:
		return color("\033[38;5;46m", "█") // vivid green
	}
}

func utf8RuneCount(s string) int {
	return visLen(s)
}
