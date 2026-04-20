package display

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/user/leetcode-cli/internal/api"
)

const pageSize = 20

// OpenQuestionFn is set by the CLI layer so browser can directly open questions
var OpenQuestionFn func(num string)

func PrintTopicList(tags []api.TopicTag) {
	Header("Topics — Pick a number to browse")
	fmt.Println()

	cols := 3
	colW := 30
	for i, t := range tags {
		num := fmt.Sprintf("%3d.", i+1)
		label := t.Name
		runes := []rune(label)
		maxLabel := colW - 7
		if len(runes) > maxLabel {
			label = string(runes[:maxLabel-1]) + "…"
		}
		fmt.Printf("  %s %-*s", color(BrightCyan, num), maxLabel, label)
		if (i+1)%cols == 0 {
			fmt.Println()
		}
	}
	if len(tags)%cols != 0 {
		fmt.Println()
	}
	fmt.Println()
}

const (
	colStatus   = 3
	colNum      = 5
	colID       = 6
	colTitle    = 44
	colDiff     = 8
	colAC       = 6
	tableInnerW = colStatus + 1 + colNum + 1 + colID + 1 + colTitle + 1 + colDiff + 1 + colAC
)

func tableLine(left, mid, right, fill string) string {
	widths := []int{colStatus + 2, colNum + 2, colID + 2, colTitle + 2, colDiff + 2, colAC + 2}
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = strings.Repeat(fill, w)
	}
	return left + strings.Join(parts, mid) + right
}

func padRight(s string, width int) string {
	vis := utf8.RuneCountInString(s)
	if vis >= width {
		runes := []rune(s)
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-vis)
}

func PrintProblemTable(problems []api.ProblemItem, topic, difficulty string, page, total int) {
	title := "All Problems"
	if topic != "" && topic != "All Problems" {
		title = topic
	}
	if difficulty != "" && difficulty != "ALL" {
		title += "  [" + difficulty + "]"
	}
	Header(title)
	fmt.Println()

	totalPages := (total + pageSize - 1) / pageSize
	fmt.Printf("  Page %s/%s     Total: %s problems\n\n",
		color(BrightWhite, strconv.Itoa(page)),
		color(Dim, strconv.Itoa(totalPages)),
		color(BrightWhite, strconv.Itoa(total)),
	)

	fmt.Printf("  %s Solved   %s Attempted   %s Not solved   %s Premium\n\n",
		color(BrightGreen, "✔"),
		color(BrightYellow, "~"),
		color(Dim, "·"),
		color(BrightYellow, "$"),
	)

	topBorder := color(Dim, tableLine("┌", "┬", "┐", "─"))
	midBorder := color(Dim, tableLine("├", "┼", "┤", "─"))
	botBorder := color(Dim, tableLine("└", "┴", "┘", "─"))
	sep := color(Dim, "│")

	fmt.Println("  " + topBorder)

	hdrStatus := padRight(" St", colStatus)
	hdrNum    := padRight("  #", colNum)
	hdrID     := padRight(" ID", colID)
	hdrTitle  := padRight(" Title", colTitle)
	hdrDiff   := padRight(" Diff", colDiff)
	hdrAC     := padRight(" AC%", colAC)
	fmt.Printf("  %s %s %s %s %s %s %s %s %s %s %s %s %s\n",
		sep, color(Bold, hdrStatus),
		sep, color(Bold, hdrNum),
		sep, color(Bold, hdrID),
		sep, color(Bold, hdrTitle),
		sep, color(Bold, hdrDiff),
		sep, color(Bold, hdrAC),
		sep,
	)
	fmt.Println("  " + midBorder)

	for i, p := range problems {
		rowNum := i + 1 + (page-1)*pageSize

		var statusStr string
		if p.IsPaidOnly {
			statusStr = color(BrightYellow, " $ ")
		} else {
			switch p.Status {
			case "ac":
				statusStr = color(BrightGreen, " ✔ ")
			case "notac":
				statusStr = color(BrightYellow, " ~ ")
			default:
				statusStr = color(Dim, " · ")
			}
		}

		numStr := color(Dim, fmt.Sprintf("%*s.", colNum-1, strconv.Itoa(rowNum)))
		idStr  := color(BrightCyan, padRight(" "+p.QuestionFrontendId, colID))

		titleRunes := []rune(p.Title)
		titleStr := p.Title
		if len(titleRunes) > colTitle-1 {
			titleStr = string(titleRunes[:colTitle-2]) + "…"
		}
		titleStr = " " + padRight(titleStr, colTitle-1)

		var diffStr string
		switch strings.ToLower(p.Difficulty) {
		case "easy":
			diffStr = color(BrightGreen, padRight(" Easy", colDiff))
		case "medium":
			diffStr = color(BrightYellow, padRight(" Med.", colDiff))
		case "hard":
			diffStr = color(BrightRed, padRight(" Hard", colDiff))
		default:
			diffStr = padRight(" "+p.Difficulty, colDiff)
		}

		acStr := color(Dim, padRight(fmt.Sprintf(" %.1f%%", p.AcRate), colAC))

		fmt.Printf("  %s %s %s %s %s %s %s %s %s %s %s %s %s\n",
			sep, statusStr,
			sep, numStr,
			sep, idStr,
			sep, titleStr,
			sep, diffStr,
			sep, acStr,
			sep,
		)
	}

	fmt.Println("  " + botBorder)
	fmt.Println()
}

// BrowseProblems is the full interactive problem browser.
func BrowseProblems(client *api.Client) {
	reader := bufio.NewReader(os.Stdin)

	Spinner("Loading topic tags")
	tags, err := client.GetAllTopicTags()
	if err != nil {
		Fail(fmt.Sprintf("Failed to load topics: %v", err))
		return
	}
	allTag := api.TopicTag{Name: "All Problems", Slug: ""}
	tags = append([]api.TopicTag{allTag}, tags...)

	PrintTopicList(tags)

	var chosenTag api.TopicTag
	for {
		fmt.Printf("  %s ", color(BrightCyan, "Topic number (Enter = all):"))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			chosenTag = allTag
			break
		}
		n, err := strconv.Atoi(input)
		if err != nil || n < 1 || n > len(tags) {
			Warn(fmt.Sprintf("Enter 1–%d", len(tags)))
			continue
		}
		chosenTag = tags[n-1]
		break
	}

	fmt.Println()
	diffs := []string{"ALL", "EASY", "MEDIUM", "HARD"}
	diffColors := []string{BrightCyan, BrightGreen, BrightYellow, BrightRed}
	for i, d := range diffs {
		fmt.Printf("  %s %s\n", color(Dim, fmt.Sprintf("%d.", i+1)), color(diffColors[i], d))
	}
	fmt.Println()
	fmt.Printf("  %s ", color(BrightCyan, "Difficulty (1-4, Enter = all):"))
	diffInput, _ := reader.ReadString('\n')
	diffInput = strings.TrimSpace(diffInput)
	chosenDiff := "ALL"
	if n, err := strconv.Atoi(diffInput); err == nil && n >= 1 && n <= 4 {
		chosenDiff = diffs[n-1]
	}

	page := 1
	var lastResult *api.ProblemListResult

	for {
		fmt.Println()
		Spinner(fmt.Sprintf("Loading %s (page %d)", chosenTag.Name, page))

		diff := ""
		if chosenDiff != "ALL" {
			diff = chosenDiff
		}
		result, err := client.GetProblemsByTopic(chosenTag.Slug, diff, "",
			(page-1)*pageSize, pageSize)
		if err != nil {
			Fail(fmt.Sprintf("Failed: %v", err))
			return
		}
		lastResult = result
		PrintProblemTable(result.Problems, chosenTag.Name, chosenDiff, page, result.Total)
		totalPages := (result.Total + pageSize - 1) / pageSize

		fmt.Printf("  %s\n", color(Dim, "  n=next  p=prev  <row#>=open  t=topic  q=quit"))
		fmt.Printf("  %s ", color(BrightCyan, "→"))
		nav, _ := reader.ReadString('\n')
		nav = strings.TrimSpace(strings.ToLower(nav))

		switch nav {
		case "n", "next":
			if page < totalPages {
				page++
			} else {
				Warn("Already on last page.")
			}
		case "p", "prev", "b":
			if page > 1 {
				page--
			} else {
				Warn("Already on first page.")
			}
		case "t", "topic":
			BrowseProblems(client)
			return
		case "q", "quit", "exit", "":
			return
		default:
			n, err := strconv.Atoi(nav)
			if err != nil || lastResult == nil {
				Warn("Unknown command. Use n/p/<row number>/t/q")
				continue
			}
			offset := (page - 1) * pageSize
			localIdx := n - 1 - offset
			var qID string
			if localIdx >= 0 && localIdx < len(lastResult.Problems) {
				qID = lastResult.Problems[localIdx].QuestionFrontendId
			} else {
				qID = strconv.Itoa(n)
			}
			fmt.Println()
			if OpenQuestionFn != nil {
				Info(fmt.Sprintf("Opening question #%s...", qID))
				fmt.Println()
				OpenQuestionFn(qID)
			} else {
				Info(fmt.Sprintf("Run: lc show %s", qID))
				fmt.Println()
			}
		}
	}
}
