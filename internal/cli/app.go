package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/user/leetcode-cli/internal/api"
	"github.com/user/leetcode-cli/internal/codegen"
	"github.com/user/leetcode-cli/internal/config"
	"github.com/user/leetcode-cli/internal/display"
	"github.com/user/leetcode-cli/internal/imagerender"
	"github.com/user/leetcode-cli/internal/keyring"
	"github.com/user/leetcode-cli/internal/storage"
)

type App struct {
	cfg    *config.Config
	client *api.Client
}

func NewApp() *App {
	cfg, _ := config.Load()

	creds, err := keyring.Load()
	if err == nil {
		cfg.Session = creds.Session
		cfg.CSRF = creds.CSRF
	}

	client := api.NewClient(cfg.Session, cfg.CSRF)

	// Wire up the image renderer
	display.ImageRenderer = func(data []byte, alt string) error {
		return imagerender.Render(data, alt)
	}

	app := &App{cfg: cfg, client: client}

	// Wire up the browse → show question function so typing a row number
	// directly opens the question inline without leaving the browser
	display.OpenQuestionFn = func(num string) {
		app.cmdShow([]string{num})
	}

	return app
}

func (a *App) Run(args []string) error {
	if len(args) < 2 {
		a.printHelp()
		return nil
	}

	cmd := args[1]
	rest := args[2:]

	switch cmd {
	case "help", "--help", "-h":
		a.printHelp()
	case "auth":
		return a.cmdAuth(rest)
	case "config":
		return a.cmdConfig(rest)
	case "today", "qod":
		return a.cmdToday(rest)
	case "show", "view":
		return a.cmdShow(rest)
	case "code":
		return a.cmdCode(rest)
	case "test":
		return a.cmdTest(rest)
	case "submit":
		return a.cmdSubmit(rest)
	case "lang":
		return a.cmdLang(rest)
	case "profile":
		return a.cmdProfile(rest)
	case "browse", "topics", "list":
		return a.cmdBrowse(rest)
	case "version":
		fmt.Printf("leetcode-cli %s\n", display.BrightCyan+"v1.1.0"+display.Reset)
	default:
		display.Fail(fmt.Sprintf("Unknown command: %s", cmd))
		fmt.Println()
		a.printHelp()
	}
	return nil
}

// ─── Help ─────────────────────────────────────────────────────────────────────

func (a *App) printHelp() {
	display.Banner()

	fmt.Printf("  %s\n\n", display.Reset+display.BrightWhite+"USAGE"+display.Reset)
	fmt.Printf("    %s [command] [flags]\n\n", display.BrightCyan+"lc"+display.Reset)

	fmt.Printf("  %s\n\n", display.BrightWhite+"COMMANDS"+display.Reset)
	cmds := [][]string{
		{"auth", "Set your LeetCode session cookie"},
		{"auth logout", "Remove saved credentials"},
		{"auth status", "Check auth status"},
		{"config", "View/edit settings"},
		{"today", "View the Question of the Day"},
		{"show <number>", "View a question → enters interactive session"},
		{"code <number>", "Generate a runnable solution file"},
		{"test <number> [file]", "Test against example cases"},
		{"submit <number> [file]", "Submit to LeetCode"},
		{"lang [language]", "View/set language (default: cpp)"},
		{"profile [username]", "View profile, stats & contest rating"},
		{"browse", "Browse problems by topic (shows solved count)"},
	}
	for _, c := range cmds {
		fmt.Printf("    %-32s %s\n",
			display.BrightCyan+c[0]+display.Reset,
			display.Reset+display.Dim+c[1]+display.Reset,
		)
	}

	fmt.Printf("\n  %s\n", display.BrightWhite+"SESSION MODE"+display.Reset)
	fmt.Printf("  %s\n\n", display.Dim+"After `lc show <number>` you enter a session — no more retyping the number:"+display.Reset)
	sessionCmds := [][]string{
		{"code [lang]", "Generate solution file"},
		{"test [file]", "Run against example cases"},
		{"submit [file]", "Submit to LeetCode"},
		{"show", "Re-display the question"},
		{"url", "Print problem URL"},
		{"q / quit", "Exit session"},
	}
	for _, c := range sessionCmds {
		fmt.Printf("    %-22s %s\n",
			display.BrightCyan+c[0]+display.Reset,
			display.Dim+c[1]+display.Reset,
		)
	}

	fmt.Printf("\n  %s\n\n", display.BrightWhite+"EXAMPLES"+display.Reset)
	examples := [][]string{
		{"lc show 1", "# View Two Sum → enter session"},
		{"  → code", "# Generate file (no number needed)"},
		{"  → test", "# Test it"},
		{"  → submit", "# Submit"},
		{"  → q", "# Exit"},
		{"", ""},
		{"lc today", "# Today's challenge"},
		{"lc browse", "# Browse by topic (shows your solved count)"},
		{"lc lang python3", "# Switch language"},
		{"lc profile", "# Your stats"},
	}
	for _, e := range examples {
		if e[0] == "" {
			fmt.Println()
		} else {
			fmt.Printf("    %-24s %s\n", display.BrightCyan+e[0]+display.Reset, display.Dim+e[1]+display.Reset)
		}
	}

	fmt.Printf("\n  %s\n", display.BrightWhite+"IMAGE RENDERING"+display.Reset)
	fmt.Printf("    %s %s\n\n",
		display.Dim+"Current terminal method:"+display.Reset,
		display.BrightCyan+imagerender.MethodName()+display.Reset,
	)
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

func (a *App) cmdAuth(args []string) error {
	if len(args) > 0 && args[0] == "logout" {
		if err := keyring.Delete(); err != nil {
			display.Warn(fmt.Sprintf("Could not remove from keychain: %v", err))
		}
		a.cfg.Session = ""
		a.cfg.CSRF = ""
		display.Success("Logged out — credentials removed from " + keyring.Backend())
		return nil
	}

	if len(args) > 0 && args[0] == "status" {
		display.Header("Auth Status")
		fmt.Println()
		fmt.Printf("  %-20s %s\n", display.Dim+"Backend:"+display.Reset,
			display.BrightCyan+keyring.Backend()+display.Reset)

		creds, err := keyring.Load()
		if err != nil {
			fmt.Printf("  %-20s %s\n", display.Dim+"Status:"+display.Reset,
				display.BrightRed+"not logged in"+display.Reset)
			fmt.Printf("  %-20s %s\n", display.Dim+"Error:"+display.Reset,
				display.Dim+err.Error()+display.Reset)
		} else if creds.Session == "" {
			fmt.Printf("  %-20s %s\n", display.Dim+"Status:"+display.Reset,
				display.BrightRed+"credentials stored but empty"+display.Reset)
			fmt.Println()
			display.Warn("Run `lc auth` to re-enter your session cookie.")
		} else {
			fmt.Printf("  %-20s %s\n", display.Dim+"Status:"+display.Reset,
				display.BrightGreen+"logged in ✔"+display.Reset)
			// Show first/last few chars of session for verification
			s := creds.Session
			preview := s
			if len(s) > 12 {
				preview = s[:6] + "..." + s[len(s)-4:]
			}
			fmt.Printf("  %-20s %s\n", display.Dim+"Session (preview):"+display.Reset,
				display.Dim+preview+display.Reset)
			csrf := creds.CSRF
			if len(csrf) > 8 {
				csrf = csrf[:4] + "..." + csrf[len(csrf)-4:]
			}
			fmt.Printf("  %-20s %s\n", display.Dim+"CSRF (preview):"+display.Reset,
				display.Dim+csrf+display.Reset)
		}
		fmt.Println()
		return nil
	}

	display.Header("Authentication Setup")
	fmt.Println()
	fmt.Println("  To authenticate, you need your LeetCode session cookie.")
	fmt.Println()
	fmt.Println(display.BrightYellow + "  How to get your session cookie:" + display.Reset)
	fmt.Println("  1. Log in to " + display.BrightCyan + "https://leetcode.com" + display.Reset)
	fmt.Println("  2. Open DevTools  →  F12  →  Application  →  Cookies  →  leetcode.com")
	fmt.Println("  3. Copy the value of " + display.BrightYellow + "LEETCODE_SESSION" + display.Reset)
	fmt.Println("  4. Copy the value of " + display.BrightYellow + "csrftoken" + display.Reset)
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("  %s ", display.BrightCyan+"LEETCODE_SESSION"+display.Reset+":")
	session, _ := reader.ReadString('\n')
	session = strings.TrimSpace(session)

	fmt.Printf("  %s ", display.BrightCyan+"csrftoken"+display.Reset+":")
	csrf, _ := reader.ReadString('\n')
	csrf = strings.TrimSpace(csrf)

	if session == "" || csrf == "" {
		display.Fail("Session or CSRF token cannot be empty.")
		return nil
	}

	a.cfg.Session = session
	a.cfg.CSRF = csrf
	a.client = api.NewClient(session, csrf)

	if err := keyring.Save(keyring.Credentials{Session: session, CSRF: csrf}); err != nil {
		display.Warn(fmt.Sprintf("Could not save to OS keychain: %v", err))
		display.Warn("Credentials will only persist for this session.")
	} else {
		display.Success("Credentials saved securely!")
		fmt.Printf("  %s %s\n\n", display.Dim+"Storage backend:"+display.Reset,
			display.BrightCyan+keyring.Backend()+display.Reset)
	}

	if err := config.Save(a.cfg); err != nil {
		display.Fail(fmt.Sprintf("Failed to save config: %v", err))
		return nil
	}

	display.Info("You can now use `lc test` and `lc submit`")
	return nil
}

// ─── Config ───────────────────────────────────────────────────────────────────

func (a *App) cmdConfig(args []string) error {
	if len(args) == 0 {
		return a.printConfig()
	}

	switch args[0] {
	case "path":
		if len(args) < 2 {
			display.Fail("Usage: lc config path <directory>")
			return nil
		}
		dir := strings.Join(args[1:], " ")
		expanded := expandHome(strings.TrimSpace(dir))
		if err := os.MkdirAll(expanded, 0755); err != nil {
			display.Fail(fmt.Sprintf("Cannot create directory: %v", err))
			return nil
		}
		a.cfg.SolutionsDir = strings.TrimSpace(dir)
		config.Save(a.cfg)
		display.Success(fmt.Sprintf("Solutions path set to: %s", display.BrightCyan+expanded+display.Reset))

	case "editor":
		if len(args) < 2 {
			display.Fail("Usage: lc config editor <editor>")
			fmt.Println("\n  Examples: code, vim, nano, notepad, subl")
			return nil
		}
		a.cfg.Editor = args[1]
		config.Save(a.cfg)
		display.Success(fmt.Sprintf("Editor set to: %s", display.BrightCyan+args[1]+display.Reset))

	case "reset":
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("  Reset all settings? %s ", display.BrightYellow+"[y/N]"+display.Reset)
		ans, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(ans)) != "y" {
			display.Info("Reset cancelled.")
			return nil
		}
		a.cfg.SolutionsDir = ""
		a.cfg.Editor = ""
		a.cfg.Language = "cpp"
		_ = keyring.Delete()
		config.Save(a.cfg)
		display.Success("Settings reset to defaults.")

	default:
		display.Fail(fmt.Sprintf("Unknown config key: %s", args[0]))
		fmt.Println("  Available: path, editor, reset")
	}
	return nil
}

func (a *App) printConfig() error {
	display.Header("Current Configuration")
	fmt.Println()

	cfgPath, _ := config.ConfigFilePath()
	solutionsDir, _ := a.cfg.SolutionsDirResolved()

	authed := display.BrightRed + "not logged in" + display.Reset
	if a.cfg.Session != "" || keyring.IsLoggedIn() {
		authed = display.BrightGreen + "logged in ✔" + display.Reset
	}

	editor := a.cfg.Editor
	if editor == "" {
		editor = display.Dim + "(not set — file path printed on `lc code`)" + display.Reset
	} else {
		editor = display.BrightCyan + editor + display.Reset
	}

	rows := [][]string{
		{"Auth status", authed},
		{"Credential store", display.BrightCyan + keyring.Backend() + display.Reset},
		{"Language", display.BrightCyan + a.cfg.Language + display.Reset},
		{"Solutions path", display.BrightCyan + solutionsDir + display.Reset},
		{"Editor", editor},
		{"Image renderer", display.BrightCyan + imagerender.MethodName() + display.Reset},
		{"Config file", display.Dim + cfgPath + display.Reset},
	}

	for _, r := range rows {
		fmt.Printf("  %-18s %s\n", display.Dim+r[0]+display.Reset, r[1])
	}

	fmt.Println()
	fmt.Println(display.Dim + "  ─────────────────────────────────────────────────────" + display.Reset)
	fmt.Println(display.Dim + "  lc config path <dir>       set solutions directory" + display.Reset)
	fmt.Println(display.Dim + "  lc config editor <name>    set editor" + display.Reset)
	fmt.Println(display.Dim + "  lc config reset            reset all settings" + display.Reset)
	fmt.Println(display.Dim + "  lc auth logout             remove saved credentials" + display.Reset)
	fmt.Println()
	return nil
}

// ─── Today ────────────────────────────────────────────────────────────────────

func (a *App) cmdToday(args []string) error {
	display.Spinner("Fetching Question of the Day")

	q, date, err := a.client.GetQuestionOfDay()
	if err != nil {
		display.Fail(fmt.Sprintf("Failed to fetch QOD: %v", err))
		return nil
	}

	display.Header(fmt.Sprintf("Question of the Day — %s", date))
	display.PrintQuestion(q, a.client)

	if hasFlag(args, "--code") {
		display.PrintCodeSnippet(a.cfg.Language, q.CodeSnippets)
	}

	fmt.Printf("  %s %s\n\n",
		display.Dim+"URL:"+display.Reset,
		display.BrightCyan+fmt.Sprintf("https://leetcode.com/problems/%s/", q.TitleSlug)+display.Reset,
	)
	return nil
}

// ─── Show ─────────────────────────────────────────────────────────────────────

func (a *App) cmdShow(args []string) error {
	if len(args) == 0 {
		display.Fail("Usage: lc show <question-number>")
		return nil
	}

	num := args[0]
	display.Spinner(fmt.Sprintf("Fetching question #%s", num))

	q, err := a.client.GetQuestionByNumber(num)
	if err != nil {
		display.Fail(fmt.Sprintf("Failed to fetch question: %v", err))
		return nil
	}

	if q.IsPaidOnly {
		display.Warn(fmt.Sprintf("Question #%s is a LeetCode Premium question.", num))
		return nil
	}

	switch q.Status {
	case "ac":
		display.Success("You have solved this problem ✔")
	case "notac":
		display.Warn("You have attempted this problem but not solved it yet")
	default:
		if a.cfg.Session != "" {
			display.Info("You have not attempted this problem yet")
		}
	}

	display.PrintQuestion(q, a.client)

	if hasFlag(args, "--code") || hasFlag(args, "-c") {
		display.PrintCodeSnippet(a.cfg.Language, q.CodeSnippets)
	}

	fmt.Printf("  %s %s\n\n",
		display.Dim+"URL:"+display.Reset,
		display.BrightCyan+fmt.Sprintf("https://leetcode.com/problems/%s/", q.TitleSlug)+display.Reset)

	// Drop into interactive session for this question
	a.questionSession(q)
	return nil
}

// questionSession is an interactive REPL for a loaded question.
func (a *App) questionSession(q *api.Question) {
	reader := bufio.NewReader(os.Stdin)
	num := q.QuestionFrontendId

	fmt.Printf("  %s\n",
		display.BrightYellow+"┌─ Session: "+q.QuestionFrontendId+". "+q.Title+" ──────────────────────────────────────────┐"+display.Reset)
	fmt.Printf("  %s\n",
		display.BrightYellow+"│  code · test · submit · show · q=quit                                        │"+display.Reset)
	fmt.Printf("  %s\n\n",
		display.BrightYellow+"└───────────────────────────────────────────────────────────────────────────────┘"+display.Reset)

	for {
		fmt.Printf("  %s %s %s ",
			display.Dim+"["+num+"]"+display.Reset,
			display.BrightCyan+"lc"+display.Reset,
			display.BrightWhite+"→"+display.Reset,
		)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := parts[0]
		rest := parts[1:]

		switch cmd {
		case "q", "quit", "exit":
			fmt.Println()
			return

		case "show", "view":
			display.PrintQuestion(q, a.client)

		case "code":
			lang := a.cfg.Language
			if len(rest) > 0 {
				lang = rest[0]
			}
			a.generateCode(q, lang)

		case "test":
			a.testQuestion(q, rest)

		case "submit":
			a.submitQuestion(q, rest)

		case "lang":
			if len(rest) > 0 {
				a.cfg.Language = rest[0]
				config.Save(a.cfg)
				display.Success(fmt.Sprintf("Language set to: %s", display.BrightCyan+rest[0]+display.Reset))
			} else {
				display.Info(fmt.Sprintf("Current language: %s", display.BrightCyan+a.cfg.Language+display.Reset))
			}

		case "url":
			fmt.Printf("\n  %s\n\n", display.BrightCyan+fmt.Sprintf("https://leetcode.com/problems/%s/", q.TitleSlug)+display.Reset)

		case "help", "?":
			fmt.Println()
			cmds := [][]string{
				{"show", "Re-display the question"},
				{"code [lang]", "Generate solution file"},
				{"test [file]", "Run against example cases"},
				{"submit [file]", "Submit to LeetCode"},
				{"lang [l]", "View/set language"},
				{"url", "Print problem URL"},
				{"q / quit", "Exit session"},
			}
			for _, c := range cmds {
				fmt.Printf("    %-20s %s\n", display.BrightCyan+c[0]+display.Reset, display.Dim+c[1]+display.Reset)
			}
			fmt.Println()

		default:
			display.Warn(fmt.Sprintf("Unknown command: %s  (type 'help' for list)", cmd))
		}
	}
}

// ─── Code ─────────────────────────────────────────────────────────────────────

func (a *App) cmdCode(args []string) error {
	if len(args) == 0 {
		display.Fail("Usage: lc code <question-number> [language]")
		return nil
	}
	num := args[0]
	display.Spinner(fmt.Sprintf("Fetching starter code for #%s", num))
	q, err := a.client.GetQuestionByNumber(num)
	if err != nil {
		display.Fail(fmt.Sprintf("Failed to fetch question: %v", err))
		return nil
	}
	lang := a.cfg.Language
	for _, arg := range args[1:] {
		if !strings.HasPrefix(arg, "-") {
			lang = arg
		}
	}
	a.generateCode(q, lang)
	return nil
}

func (a *App) generateCode(q *api.Question, lang string) {
	var snippet *api.Snippet
	for i, s := range q.CodeSnippets {
		if s.LangSlug == lang || s.Lang == lang {
			snippet = &q.CodeSnippets[i]
			break
		}
	}
	if snippet == nil {
		display.Warn(fmt.Sprintf("No starter code for language: %s", lang))
		fmt.Println("  Available languages:")
		for _, s := range q.CodeSnippets {
			fmt.Printf("    • %s (%s)\n", s.Lang, s.LangSlug)
		}
		return
	}
	wrappedCode := codegen.Wrap(q, *snippet, q.ExampleTestcases)
	path, err := storage.SaveSolution(a.cfg, q.QuestionFrontendId, q.TitleSlug, lang, wrappedCode)
	if err != nil {
		display.Fail(fmt.Sprintf("Failed to save solution file: %v", err))
		return
	}
	display.Success("Solution file created:")
	fmt.Printf("\n    %s\n\n", display.BrightCyan+path+display.Reset)
	fmt.Println(display.BrightWhite + "  How to run locally:" + display.Reset)
	switch lang {
	case "cpp":
		fmt.Printf("    %s\n", display.Dim+"g++ -std=c++17 -o sol "+path+" && ./sol"+display.Reset)
	case "golang", "go":
		fmt.Printf("    %s\n", display.Dim+"go run "+path+display.Reset)
	case "python3", "python":
		fmt.Printf("    %s\n", display.Dim+"python3 "+path+display.Reset)
	case "java":
		fmt.Printf("    %s\n", display.Dim+"javac "+path+" && java -cp $(dirname "+path+") Main"+display.Reset)
	case "rust":
		fmt.Printf("    %s\n", display.Dim+"rustc "+path+" && ./solution"+display.Reset)
	case "javascript":
		fmt.Printf("    %s\n", display.Dim+"node "+path+display.Reset)
	}
	fmt.Println()
	if a.cfg.Editor != "" {
		display.Info(fmt.Sprintf("Opening in %s...", a.cfg.Editor))
		fmt.Printf("  %s\n\n", display.Dim+"$ "+a.cfg.Editor+" "+path+display.Reset)
	}
}

// ─── Lang ─────────────────────────────────────────────────────────────────────

func (a *App) cmdLang(args []string) error {
	if len(args) == 0 {
		display.Info(fmt.Sprintf("Current language: %s", display.BrightCyan+a.cfg.Language+display.Reset))
		fmt.Println()
		langs := [][]string{
			{"cpp", "C++ (default)"},
			{"golang", "Go"},
			{"python3", "Python 3"},
			{"javascript", "JavaScript"},
			{"typescript", "TypeScript"},
			{"java", "Java"},
			{"rust", "Rust"},
			{"c", "C"},
		}
		for _, l := range langs {
			marker := "  "
			if l[0] == a.cfg.Language {
				marker = display.BrightGreen + "▶ " + display.Reset
			}
			fmt.Printf("    %s%s %s\n", marker, display.BrightCyan+l[0]+display.Reset, display.Dim+"("+l[1]+")"+display.Reset)
		}
		fmt.Println()
		return nil
	}

	a.cfg.Language = args[0]
	config.Save(a.cfg)
	display.Success(fmt.Sprintf("Default language set to: %s", display.BrightCyan+args[0]+display.Reset))
	return nil
}

// ─── Test ─────────────────────────────────────────────────────────────────────

func (a *App) cmdTest(args []string) error {
	if len(args) == 0 {
		display.Fail("Usage: lc test <question-number> [solution-file] [--input \"...\"]")
		return nil
	}
	num := args[0]
	display.Spinner(fmt.Sprintf("Fetching question #%s", num))
	q, err := a.client.GetQuestionByNumber(num)
	if err != nil {
		display.Fail(fmt.Sprintf("Failed to fetch question: %v", err))
		return nil
	}
	a.testQuestion(q, args[1:])
	return nil
}

func (a *App) testQuestion(q *api.Question, args []string) {
	code, filePath, err := a.resolveCode(args, q, a.cfg.Language)
	if err != nil {
		display.Fail(err.Error())
		return
	}
	testInput := q.ExampleTestcases
	if testInput == "" {
		testInput = q.SampleTestCase
	}
	if customInput := flagValue(args, "--input"); customInput != "" {
		testInput = customInput
	}
	display.Header(fmt.Sprintf("Testing #%s — %s", q.QuestionFrontendId, q.Title))
	fmt.Printf("  %-16s %s\n", display.Dim+"File:"+display.Reset, display.BrightCyan+filePath+display.Reset)
	fmt.Printf("  %-16s %s\n", display.Dim+"Language:"+display.Reset, display.BrightCyan+a.cfg.Language+display.Reset)
	fmt.Printf("  %-16s\n%s\n\n",
		display.Dim+"Test Input:"+display.Reset,
		indentLines(testInput, "    ", display.Yellow, display.Reset),
	)
	display.Spinner("Submitting test run to LeetCode judge")
	checkID, err := a.client.TestCode(q.TitleSlug, a.cfg.Language, code, testInput)
	if err != nil {
		display.Fail(fmt.Sprintf("Test failed: %v", err))
		return
	}
	display.Spinner("Waiting for judge result")
	cr, err := a.client.PollResult(checkID, 30*time.Second)
	if err != nil {
		display.Fail(fmt.Sprintf("Polling failed: %v", err))
		return
	}
	printTestResult(cr)
}

// ─── Submit ───────────────────────────────────────────────────────────────────

func (a *App) cmdSubmit(args []string) error {
	if len(args) == 0 {
		display.Fail("Usage: lc submit <question-number> [solution-file]")
		return nil
	}
	num := args[0]
	display.Spinner(fmt.Sprintf("Fetching question #%s", num))
	q, err := a.client.GetQuestionByNumber(num)
	if err != nil {
		display.Fail(fmt.Sprintf("Failed to fetch question: %v", err))
		return nil
	}
	a.submitQuestion(q, args[1:])
	return nil
}

func (a *App) submitQuestion(q *api.Question, args []string) {
	code, filePath, err := a.resolveCode(args, q, a.cfg.Language)
	if err != nil {
		display.Fail(err.Error())
		return
	}
	stripped := api.StripSubmitCode(code)
	display.Header(fmt.Sprintf("Submitting #%s — %s", q.QuestionFrontendId, q.Title))
	fmt.Printf("  %-16s %s\n", display.Dim+"File:"+display.Reset, display.BrightCyan+filePath+display.Reset)
	fmt.Printf("  %-16s %s\n", display.Dim+"Language:"+display.Reset, display.BrightCyan+a.cfg.Language+display.Reset)
	if strings.Contains(code, "// --- SUBMIT START ---") {
		lineCount := len(strings.Split(stripped, "\n"))
		fmt.Printf("  %-16s %s\n", display.Dim+"Submitting:"+display.Reset,
			display.Dim+fmt.Sprintf("%d lines (harness stripped automatically)", lineCount)+display.Reset)
	}
	fmt.Println()
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("  Submit to LeetCode? %s ", display.BrightYellow+"[y/N]"+display.Reset)
	ans, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(ans)) != "y" {
		display.Info("Submission cancelled.")
		return
	}
	fmt.Println()
	display.Spinner("Submitting solution to LeetCode")
	checkID, err := a.client.SubmitCode(q.TitleSlug, a.cfg.Language, code)
	if err != nil {
		display.Fail(fmt.Sprintf("Submit failed: %v", err))
		return
	}
	display.Spinner("Waiting for judge result")
	cr, err := a.client.PollResult(checkID, 60*time.Second)
	if err != nil {
		display.Fail(fmt.Sprintf("Polling failed: %v", err))
		return
	}
	printSubmitResult(cr)
}

// ─── Result Printers ──────────────────────────────────────────────────────────

func printTestResult(cr *api.CheckResponse) {
	fmt.Println()
	display.Divider()

	if cr.CompileError != "" {
		display.Fail("Compile Error")
		fmt.Printf("\n  %s\n", display.BrightRed+cr.CompileError+display.Reset)
		if cr.FullCompileError != "" && cr.FullCompileError != cr.CompileError {
			fmt.Printf("\n  %s\n", display.Dim+cr.FullCompileError+display.Reset)
		}
		display.Divider()
		return
	}

	if !cr.RunSuccess {
		display.Fail("Runtime Error")
		if cr.RuntimeError != "" {
			fmt.Printf("\n  %s\n", display.BrightRed+cr.RuntimeError+display.Reset)
		}
		if cr.LastTestcase != "" {
			fmt.Printf("\n  %s\n    %s\n", display.Dim+"Failed on:"+display.Reset, display.Yellow+cr.LastTestcase+display.Reset)
		}
		display.Divider()
		return
	}

	total := cr.TotalTestcases
	correct := cr.TotalCorrect

	if cr.CorrectAnswer {
		display.Success(fmt.Sprintf("All test cases passed! (%d/%d)", correct, total))
	} else {
		display.Fail(fmt.Sprintf("Some test cases failed (%d/%d passed)", correct, total))
	}

	fmt.Println()
	if cr.CompareResult != "" {
		for i, r := range strings.Split(cr.CompareResult, "") {
			if r == "1" {
				fmt.Printf("  %s  Test case %-3d  %s\n", display.BrightGreen+"✔"+display.Reset, i+1, display.BrightGreen+"PASSED"+display.Reset)
			} else {
				fmt.Printf("  %s  Test case %-3d  %s\n", display.BrightRed+"✘"+display.Reset, i+1, display.BrightRed+"FAILED"+display.Reset)
			}
		}
		fmt.Println()
	}

	if len(cr.CodeOutput) > 0 && !cr.CorrectAnswer {
		display.PrintStat("Your output", strings.Join(cr.CodeOutput, ", "), display.BrightRed)
	}
	if cr.ExpectedOutput != "" && !cr.CorrectAnswer {
		display.PrintStat("Expected output", cr.ExpectedOutput, display.BrightGreen)
	}
	display.Divider()
}

func printSubmitResult(cr *api.CheckResponse) {
	fmt.Println()
	display.Divider()

	// Handle non-SUCCESS states explicitly
	switch cr.StatusMsg {
	case "Accepted":
		printAccepted(cr)
	case "Wrong Answer":
		display.Fail(fmt.Sprintf("Wrong Answer — %d/%d test cases passed", cr.TotalCorrect, cr.TotalTestcases))
		fmt.Println()
		if len(cr.CodeOutput) > 0 {
			display.PrintStat("Your output", strings.Join(cr.CodeOutput, ", "), display.BrightRed)
		}
		if cr.ExpectedOutput != "" {
			display.PrintStat("Expected", cr.ExpectedOutput, display.BrightGreen)
		}
		if cr.LastTestcase != "" {
			display.PrintStat("Failed on input", cr.LastTestcase, display.Yellow)
		}
	case "Time Limit Exceeded":
		display.Fail("Time Limit Exceeded")
		if cr.LastTestcase != "" {
			fmt.Printf("\n  %s\n    %s\n", display.Dim+"Failed on:"+display.Reset, display.Yellow+cr.LastTestcase+display.Reset)
		}
	case "Memory Limit Exceeded":
		display.Fail("Memory Limit Exceeded")
	case "Runtime Error":
		display.Fail("Runtime Error")
		if cr.RuntimeError != "" {
			fmt.Printf("\n  %s\n", display.BrightRed+cr.RuntimeError+display.Reset)
		}
		if cr.LastTestcase != "" {
			fmt.Printf("\n  %s\n    %s\n", display.Dim+"Failed on:"+display.Reset, display.Yellow+cr.LastTestcase+display.Reset)
		}
	case "Compile Error":
		display.Fail("Compile Error")
		if cr.CompileError != "" {
			fmt.Printf("\n  %s\n", display.BrightRed+cr.CompileError+display.Reset)
		}
	default:
		// Fallback: infer from status_code
		// 10 = Accepted, 11 = Wrong Answer, 12 = MLE, 13 = OLE, 14 = TLE, 15 = Runtime Error, 20 = Compile Error
		if cr.TotalCorrect > 0 && cr.TotalCorrect == cr.TotalTestcases {
			printAccepted(cr)
		} else {
			display.Fail(fmt.Sprintf("Submission failed: %s (code %d)", cr.StatusMsg, cr.StatusCode))
			if cr.LastTestcase != "" {
				fmt.Printf("\n  %s\n    %s\n", display.Dim+"Failed on:"+display.Reset, display.Yellow+cr.LastTestcase+display.Reset)
			}
		}
	}

	display.Divider()
}

func printAccepted(cr *api.CheckResponse) {
	fmt.Println()
	fmt.Println("  " + display.BgGreen + display.BrightWhite + "  ✔  ACCEPTED  " + display.Reset)
	fmt.Println()
	if cr.StatusRuntime != "" {
		display.PrintStat("Runtime", cr.StatusRuntime, display.BrightGreen)
		if cr.RuntimePercentile > 0 {
			display.PrintStat("Beats", fmt.Sprintf("%.1f%% of submissions", cr.RuntimePercentile), display.BrightGreen)
		}
	}
	if cr.StatusMemory != "" {
		display.PrintStat("Memory", cr.StatusMemory, display.BrightCyan)
		if cr.MemoryPercentile > 0 {
			display.PrintStat("Beats", fmt.Sprintf("%.1f%% in memory", cr.MemoryPercentile), display.BrightCyan)
		}
	}
	display.PrintStat("Test cases", fmt.Sprintf("%d/%d", cr.TotalCorrect, cr.TotalTestcases), display.BrightGreen)
	fmt.Println()
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (a *App) resolveCode(args []string, q *api.Question, lang string) (string, string, error) {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") && (strings.Contains(arg, ".") || strings.Contains(arg, "/") || strings.Contains(arg, "\\")) {
			code, err := storage.ReadFile(arg)
			if err != nil {
				return "", "", err
			}
			return code, arg, nil
		}
	}

	if storage.SolutionExists(a.cfg, q.QuestionFrontendId, q.TitleSlug, lang) {
		path, _ := storage.SolutionPath(a.cfg, q.QuestionFrontendId, q.TitleSlug, lang)
		code, err := storage.LoadSolution(a.cfg, q.QuestionFrontendId, q.TitleSlug, lang)
		if err != nil {
			return "", "", err
		}
		return code, path, nil
	}

	return "", "", fmt.Errorf(
		"no solution file found for #%s.\n  Run `lc code %s` to generate one, or pass a file path",
		q.QuestionFrontendId, q.QuestionFrontendId,
	)
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func flagValue(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, flag+"=") {
			return strings.TrimPrefix(a, flag+"=")
		}
	}
	return ""
}

func indentLines(s, indent, colorOn, colorOff string) string {
	lines := strings.Split(s, "\n")
	result := make([]string, 0, len(lines))
	for _, l := range lines {
		result = append(result, indent+colorOn+l+colorOff)
	}
	return strings.Join(result, "\n")
}

// ─── Profile ──────────────────────────────────────────────────────────────────

func (a *App) cmdProfile(args []string) error {
	username := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		username = args[0]
	}

	if username == "" {
		display.Spinner("Detecting logged-in username")
		u, err := a.client.GetCurrentUsername()
		if err != nil || u == "" {
			display.Fail("Could not detect username. Please run: lc profile <username>")
			fmt.Printf("\n  %s\n", display.Dim+"Example: lc profile lee215"+display.Reset)
			return nil
		}
		username = u
	}

	display.Spinner(fmt.Sprintf("Loading profile for @%s", username))
	profile, err := a.client.GetUserProfile(username)
	if err != nil {
		display.Fail(fmt.Sprintf("Failed to load profile: %v", err))
		return nil
	}

	display.Spinner("Loading submission calendar")
	cal, err := a.client.GetSubmissionCalendar(username)
	if err != nil {
		cal = nil // heatmap is optional — don't fail
	}

	display.PrintProfile(profile, cal)
	return nil
}

// ─── Browse ───────────────────────────────────────────────────────────────────

func (a *App) cmdBrowse(args []string) error {
	display.BrowseProblems(a.client)
	return nil
}
