package display

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/user/leetcode-cli/internal/api"
)

// ANSI color codes
const (
	Reset        = "\033[0m"
	Bold         = "\033[1m"
	Dim          = "\033[2m"
	Red          = "\033[31m"
	Green        = "\033[32m"
	Yellow       = "\033[33m"
	Blue         = "\033[34m"
	Magenta      = "\033[35m"
	Cyan         = "\033[36m"
	White        = "\033[37m"
	BrightRed    = "\033[91m"
	BrightGreen  = "\033[92m"
	BrightYellow = "\033[93m"
	BrightBlue   = "\033[94m"
	BrightCyan   = "\033[96m"
	BrightWhite  = "\033[97m"
	BgGreen      = "\033[42m"
	BgRed        = "\033[41m"
	BgYellow     = "\033[43m"
	BgBlue       = "\033[44m"
)

func color(c, s string) string { return c + s + Reset }
func bold(s string) string     { return Bold + s + Reset }

func DifficultyColor(d string) string {
	switch strings.ToLower(d) {
	case "easy":
		return color(BrightGreen, d)
	case "medium":
		return color(BrightYellow, d)
	case "hard":
		return color(BrightRed, d)
	}
	return d
}

func Banner() {
	fmt.Println(color(BrightCyan, `
  _              _    ____          _
 | |    ___  ___| |_ / ___|___   __| | ___
 | |   / _ \/ _ \ __| |   / _ \ / _  |/ _ \
 | |__|  __/  __/ |_| |__| (_) | (_| |  __/
 |_____\___|\___|\__|\____\___/ \__,_|\___|  CLI`))
	fmt.Println(color(Dim, "  The unofficial LeetCode terminal client\n"))
}

func Header(title string) {
	line := strings.Repeat("─", 60)
	fmt.Println(color(Blue, line))
	fmt.Printf("%s %s\n", color(BrightBlue, "▶"), bold(title))
	fmt.Println(color(Blue, line))
}

func Success(msg string) {
	fmt.Printf("%s %s\n", color(BrightGreen, "✔"), color(BrightGreen, msg))
}

func Fail(msg string) {
	fmt.Printf("%s %s\n", color(BrightRed, "✘"), color(BrightRed, msg))
}

func Info(msg string) {
	fmt.Printf("%s %s\n", color(BrightBlue, "ℹ"), msg)
}

func Warn(msg string) {
	fmt.Printf("%s %s\n", color(BrightYellow, "⚠"), color(BrightYellow, msg))
}

func Spinner(msg string) {
	fmt.Printf("%s %s...\n", color(Cyan, "⟳"), color(Dim, msg))
}

// ImageRef holds an image URL found in the question HTML
type ImageRef struct {
	URL string
	Alt string
}

// ExtractImages pulls <img> src/alt from HTML before stripping tags
func ExtractImages(html string) []ImageRef {
	var images []ImageRef
	imgRe := regexp.MustCompile(`(?i)<img[^>]+>`)
	srcRe := regexp.MustCompile(`(?i)src=["']([^"']+)["']`)
	altRe := regexp.MustCompile(`(?i)alt=["']([^"']*)"`)

	for _, imgTag := range imgRe.FindAllString(html, -1) {
		srcMatch := srcRe.FindStringSubmatch(imgTag)
		if len(srcMatch) < 2 {
			continue
		}
		src := srcMatch[1]

		// Resolve relative URLs
		if strings.HasPrefix(src, "/") {
			src = "https://assets.leetcode.com" + src
		}
		if !strings.HasPrefix(src, "http") {
			continue
		}

		alt := ""
		altMatch := altRe.FindStringSubmatch(imgTag)
		if len(altMatch) >= 2 {
			alt = altMatch[1]
		}

		images = append(images, ImageRef{URL: src, Alt: alt})
	}
	return images
}

// StripHTML converts HTML to readable terminal text, replacing <img> with placeholders
func StripHTML(s string) (string, []ImageRef) {
	images := ExtractImages(s)

	// Replace <img> tags with an inline placeholder
	imgRe := regexp.MustCompile(`(?i)<img[^>]+>`)
	imgIdx := 0
	s = imgRe.ReplaceAllStringFunc(s, func(_ string) string {
		placeholder := fmt.Sprintf("\n  [IMAGE %d]\n", imgIdx+1)
		imgIdx++
		return placeholder
	})

	// Block elements → newlines
	blockRe := regexp.MustCompile(`(?i)</?(p|div|section|article|header|footer|h[1-6]|blockquote|hr|table|tr|td|th|ol|ul|li|pre|figure|figcaption)[^>]*>`)
	s = blockRe.ReplaceAllStringFunc(s, func(tag string) string {
		lc := strings.ToLower(tag)
		if strings.Contains(lc, "<li") {
			return "\n  • "
		}
		if strings.Contains(lc, "<hr") {
			return "\n" + strings.Repeat("─", 58) + "\n"
		}
		return "\n"
	})

	// Inline formatting
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = strings.ReplaceAll(s, "<strong>", Bold)
	s = strings.ReplaceAll(s, "</strong>", Reset)
	s = strings.ReplaceAll(s, "<b>", Bold)
	s = strings.ReplaceAll(s, "</b>", Reset)
	s = strings.ReplaceAll(s, "<em>", color(Cyan, ""))
	s = strings.ReplaceAll(s, "</em>", Reset)
	s = strings.ReplaceAll(s, "<i>", color(Cyan, ""))
	s = strings.ReplaceAll(s, "</i>", Reset)
	s = strings.ReplaceAll(s, "<code>", color(Yellow, ""))
	s = strings.ReplaceAll(s, "</code>", Reset)
	s = strings.ReplaceAll(s, "<pre>", "\n"+color(Dim, ""))
	s = strings.ReplaceAll(s, "</pre>", Reset+"\n")
	s = strings.ReplaceAll(s, "<sup>", "^")
	s = strings.ReplaceAll(s, "</sup>", "")
	s = strings.ReplaceAll(s, "<sub>", "_")
	s = strings.ReplaceAll(s, "</sub>", "")

	// HTML entities
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&le;", "≤")
	s = strings.ReplaceAll(s, "&ge;", "≥")
	s = strings.ReplaceAll(s, "&times;", "×")
	s = strings.ReplaceAll(s, "&divide;", "÷")
	s = strings.ReplaceAll(s, "&minus;", "−")
	s = strings.ReplaceAll(s, "&plusmn;", "±")

	// Strip remaining tags
	tagRe := regexp.MustCompile(`<[^>]*>`)
	s = tagRe.ReplaceAllString(s, "")

	// Collapse 3+ blank lines → 2
	blankRe := regexp.MustCompile(`\n{3,}`)
	s = blankRe.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s), images
}

// ImageRenderer is called to render images found in questions.
// It's set by the CLI layer so display.go doesn't import imagerender
// (avoiding circular imports / keeping display lightweight).
var ImageRenderer func(data []byte, alt string) error

func PrintQuestion(q *api.Question, client interface {
	FetchImage(url string) ([]byte, string, error)
}) {
	fmt.Println()
	fmt.Printf("  %s. %s  %s\n\n",
		color(BrightWhite, q.QuestionFrontendId),
		bold(color(BrightWhite, q.Title)),
		DifficultyColor(q.Difficulty),
	)

	if len(q.TopicTags) > 0 {
		tags := make([]string, 0, len(q.TopicTags))
		for _, t := range q.TopicTags {
			tags = append(tags, color(Magenta, "["+t.Name+"]"))
		}
		fmt.Printf("  %s\n\n", strings.Join(tags, " "))
	}

	fmt.Println(color(Blue, "  "+strings.Repeat("─", 58)))

	desc, images := StripHTML(q.Content)

	// Print description line-by-line, rendering images inline when reached
	imageIdx := 0
	for _, line := range strings.Split(desc, "\n") {
		trimmed := strings.TrimSpace(line)
		// Check if this is an image placeholder
		if strings.HasPrefix(trimmed, "[IMAGE ") && strings.HasSuffix(trimmed, "]") {
			if imageIdx < len(images) {
				img := images[imageIdx]
				imageIdx++
				renderInlineImage(img, client)
			}
			continue
		}
		fmt.Printf("  %s\n", line)
	}
	fmt.Println()

	if len(q.Hints) > 0 {
		fmt.Printf("  %s\n", color(BrightYellow, "💡 Hints (use --hints to reveal)"))
	}

	if q.ExampleTestcases != "" {
		fmt.Println(color(Blue, "  "+strings.Repeat("─", 58)))
		fmt.Printf("  %s\n", bold("Example Test Cases:"))
		for _, tc := range strings.Split(q.ExampleTestcases, "\n") {
			fmt.Printf("    %s %s\n", color(Cyan, "▸"), color(Yellow, tc))
		}
	}

	fmt.Println()
}

func renderInlineImage(img ImageRef, client interface {
	FetchImage(url string) ([]byte, string, error)
}) {
	if ImageRenderer == nil || client == nil {
		// No renderer set — print the URL as a clickable link
		label := img.Alt
		if label == "" {
			label = "image"
		}
		fmt.Printf("  %s %s\n", color(BrightBlue, "🖼"), color(Dim, img.URL))
		return
	}

	data, _, err := client.FetchImage(img.URL)
	if err != nil {
		fmt.Printf("  %s %s %s\n", color(BrightYellow, "⚠"), color(Dim, "[image unavailable]"), color(Dim, img.URL))
		return
	}

	if err := ImageRenderer(data, img.Alt); err != nil {
		fmt.Printf("  %s %s\n", color(BrightBlue, "🖼"), color(Dim, img.URL))
	}
}

func PrintCodeSnippet(lang string, snippets []api.Snippet) {
	for _, s := range snippets {
		if s.LangSlug == lang || s.Lang == lang {
			Header(fmt.Sprintf("Starter code — %s", s.Lang))
			printCode(s.Code, lang)
			fmt.Println()
			return
		}
	}
	Warn(fmt.Sprintf("No snippet found for language: %s", lang))
	fmt.Println("  Available languages:")
	for _, s := range snippets {
		fmt.Printf("    • %s (%s)\n", s.Lang, s.LangSlug)
	}
}

func printCode(code, lang string) {
	lines := strings.Split(code, "\n")
	fmt.Println()
	for i, line := range lines {
		var highlighted string
		switch lang {
		case "cpp":
			highlighted = syntaxHighlightCPP(line)
		case "golang", "go":
			highlighted = syntaxHighlightGo(line)
		default:
			highlighted = line
		}
		fmt.Printf("  %s  %s\n",
			color(Dim, fmt.Sprintf("%3d", i+1)),
			highlighted,
		)
	}
}

// ─── C++ Syntax Highlighting ──────────────────────────────────────────────────

func syntaxHighlightCPP(line string) string {
	cppKeywords := []string{
		"int", "long", "short", "char", "bool", "float", "double", "void",
		"string", "auto", "const", "static", "inline", "virtual", "override",
		"public", "private", "protected", "class", "struct", "enum", "namespace",
		"return", "if", "else", "for", "while", "do", "switch", "case", "break",
		"continue", "default", "new", "delete", "nullptr", "true", "false",
		"vector", "map", "set", "unordered_map", "unordered_set", "pair",
		"queue", "stack", "deque", "priority_queue", "list", "array",
		"include", "using",
	}

	// Comments
	if idx := strings.Index(line, "//"); idx >= 0 {
		comment := line[idx:]
		line = line[:idx] + color(Dim, comment)
	}

	// Preprocessor
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return color(Magenta, line)
	}

	// String literals
	strRe := regexp.MustCompile(`"([^"]*)"`)
	line = strRe.ReplaceAllStringFunc(line, func(s string) string {
		return color(Green, s)
	})

	// Keywords
	for _, kw := range cppKeywords {
		line = replaceWord(line, kw, color(Blue, kw))
	}

	// Numbers
	numRe := regexp.MustCompile(`\b(\d+)\b`)
	line = numRe.ReplaceAllStringFunc(line, func(s string) string {
		return color(Cyan, s)
	})

	return line
}

// ─── Go Syntax Highlighting ───────────────────────────────────────────────────

func syntaxHighlightGo(line string) string {
	goKeywords := []string{
		"func", "return", "if", "else", "for", "range", "var", "const",
		"type", "struct", "interface", "map", "make", "append", "len",
		"cap", "nil", "true", "false", "package", "import", "switch",
		"case", "default", "break", "continue", "go", "defer", "select",
		"chan", "int", "int64", "int32", "string", "bool", "float64",
		"byte", "rune", "error",
	}

	strRe := regexp.MustCompile(`"([^"]*)"`)
	line = strRe.ReplaceAllStringFunc(line, func(s string) string {
		return color(Green, s)
	})

	if idx := strings.Index(line, "//"); idx >= 0 {
		comment := line[idx:]
		line = line[:idx] + color(Dim, comment)
	}

	for _, kw := range goKeywords {
		line = replaceWord(line, kw, color(Blue, kw))
	}

	numRe := regexp.MustCompile(`\b(\d+)\b`)
	line = numRe.ReplaceAllStringFunc(line, func(s string) string {
		return color(Cyan, s)
	})

	return line
}

func replaceWord(s, word, replacement string) string {
	result := s
	i := 0
	for i < len(result) {
		idx := strings.Index(result[i:], word)
		if idx == -1 {
			break
		}
		abs := i + idx
		before := abs == 0 || !isAlphaNum(result[abs-1])
		after := abs+len(word) >= len(result) || !isAlphaNum(result[abs+len(word)])
		if before && after {
			result = result[:abs] + replacement + result[abs+len(word):]
			i = abs + len(replacement)
		} else {
			i = abs + len(word)
		}
	}
	return result
}

func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func Box(title, content, clr string) {
	width := 60
	top := clr + "╭" + strings.Repeat("─", width-2) + "╮" + Reset
	bot := clr + "╰" + strings.Repeat("─", width-2) + "╯" + Reset
	fmt.Println(top)
	fmt.Printf("%s  %s%s%s\n", clr+"│"+Reset, bold(clr+title+Reset), strings.Repeat(" ", width-3-len(title)), clr+"│"+Reset)
	fmt.Printf("%s  %s%s\n", clr+"│"+Reset, content, Reset)
	fmt.Println(bot)
}

func Divider() {
	fmt.Println(color(Dim, strings.Repeat("─", 62)))
}

func PrintStat(label, value, clr string) {
	fmt.Printf("  %-22s %s\n", color(Dim, label+":"), color(clr, value))
}

func PrintTestResult(cr interface{ GetState() string }, isTest bool) {}

