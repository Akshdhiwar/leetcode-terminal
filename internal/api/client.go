package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	BaseURL    = "https://leetcode.com"
	GraphQLURL = "https://leetcode.com/graphql"
)

type Client struct {
	http    *http.Client
	session string
	csrf    string
}

func NewClient(session, csrf string) *Client {
	return &Client{
		http:    &http.Client{Timeout: 30 * time.Second},
		session: session,
		csrf:    csrf,
	}
}

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (c *Client) doGraphQL(query string, vars map[string]interface{}, out interface{}) error {
	body, err := json.Marshal(graphQLRequest{Query: query, Variables: vars})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", GraphQLURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", BaseURL)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("x-csrftoken", c.csrf)
	req.Header.Set("Origin", BaseURL)

	if c.session != "" {
		req.AddCookie(&http.Cookie{Name: "LEETCODE_SESSION", Value: c.session})
		req.AddCookie(&http.Cookie{Name: "csrftoken", Value: c.csrf})
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d from LeetCode API: %s", resp.StatusCode, string(b))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, out)
}

// ─── Question of the Day ──────────────────────────────────────────────────────

type QODResponse struct {
	Data struct {
		ActiveDailyCodingChallengeQuestion struct {
			Date       string   `json:"date"`
			UserStatus string   `json:"userStatus"`
			Link       string   `json:"link"`
			Question   Question `json:"question"`
		} `json:"activeDailyCodingChallengeQuestion"`
	} `json:"data"`
}

func (c *Client) GetQuestionOfDay() (*Question, string, error) {
	query := `query questionOfToday {
		activeDailyCodingChallengeQuestion {
			date
			userStatus
			link
			question {
				questionFrontendId
				title
				titleSlug
				difficulty
				content
				topicTags { name }
				stats
				hints
				sampleTestCase
				exampleTestcases
				codeSnippets { lang langSlug code }
				metaData
			}
		}
	}`

	var resp QODResponse
	if err := c.doGraphQL(query, nil, &resp); err != nil {
		return nil, "", err
	}

	q := resp.Data.ActiveDailyCodingChallengeQuestion
	return &q.Question, q.Date, nil
}

// ─── Question Types ───────────────────────────────────────────────────────────

type QuestionListResponse struct {
	Data struct {
		ProblemsetQuestionList struct {
			Questions []Question `json:"data"`
		} `json:"problemsetQuestionList"`
	} `json:"data"`
}

type QuestionDetailResponse struct {
	Data struct {
		Question Question `json:"question"`
	} `json:"data"`
}

type Question struct {
	QuestionFrontendId string    `json:"questionFrontendId"`
	Title              string    `json:"title"`
	TitleSlug          string    `json:"titleSlug"`
	Difficulty         string    `json:"difficulty"`
	Content            string    `json:"content"`
	TopicTags          []Tag     `json:"topicTags"`
	Stats              string    `json:"stats"`
	Hints              []string  `json:"hints"`
	SampleTestCase     string    `json:"sampleTestCase"`
	ExampleTestcases   string    `json:"exampleTestcases"`
	CodeSnippets       []Snippet `json:"codeSnippets"`
	IsPaidOnly         bool      `json:"isPaidOnly"`
	MetaData           string    `json:"metaData"` // JSON string with function signature info
}

type Tag struct {
	Name string `json:"name"`
}

type Snippet struct {
	Lang     string `json:"lang"`
	LangSlug string `json:"langSlug"`
	Code     string `json:"code"`
}

// MetaDataParsed holds parsed function signature info
type MetaDataParsed struct {
	Name   string `json:"name"`
	Params []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"params"`
	Return struct {
		Type string `json:"type"`
	} `json:"return"`
}

func (q *Question) ParseMetaData() (*MetaDataParsed, error) {
	if q.MetaData == "" {
		return nil, fmt.Errorf("no metadata")
	}
	var m MetaDataParsed
	if err := json.Unmarshal([]byte(q.MetaData), &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ─── Fix: GetQuestionByNumber with proper pagination ─────────────────────────

func (c *Client) GetQuestionByNumber(num string) (*Question, error) {
	listQuery := `query problemsetQuestionList($skip: Int, $limit: Int, $filters: QuestionListFilterInput) {
		problemsetQuestionList: questionList(
			categorySlug: ""
			limit: $limit
			skip: $skip
			filters: $filters
		) {
			data {
				questionFrontendId
				titleSlug
				title
				difficulty
				isPaidOnly
			}
		}
	}`

	limit := 50
	skip := 0
	var slug string

	for {
		vars := map[string]interface{}{
			"skip":    skip,
			"limit":   limit,
			"filters": map[string]interface{}{},
		}

		var listResp QuestionListResponse
		if err := c.doGraphQL(listQuery, vars, &listResp); err != nil {
			return nil, err
		}

		questions := listResp.Data.ProblemsetQuestionList.Questions
		if len(questions) == 0 {
			break
		}

		for _, q := range questions {
			if q.QuestionFrontendId == num {
				slug = q.TitleSlug
				break
			}
		}

		if slug != "" {
			break
		}

		skip += limit
	}

	if slug == "" {
		// Fallback: try direct title slug guess (e.g. "two-sum" from "1")
		slug, _ = c.findSlugBySearch(num)
	}

	if slug == "" {
		return nil, fmt.Errorf("question #%s not found", num)
	}

	return c.GetQuestionBySlug(slug)
}

func (c *Client) findSlugBySearch(num string) (string, error) {
	query := `query problemsetQuestionList($skip: Int, $limit: Int, $filters: QuestionListFilterInput) {
		problemsetQuestionList: questionList(
			categorySlug: ""
			limit: $limit
			skip: $skip
			filters: $filters
		) {
			data {
				questionFrontendId
				titleSlug
			}
		}
	}`

	for skip := 0; skip <= 3500; skip += 500 {
		vars := map[string]interface{}{
			"skip":    skip,
			"limit":   500,
			"filters": map[string]interface{}{},
		}
		var resp QuestionListResponse
		if err := c.doGraphQL(query, vars, &resp); err != nil {
			return "", err
		}
		for _, q := range resp.Data.ProblemsetQuestionList.Questions {
			if q.QuestionFrontendId == num {
				return q.TitleSlug, nil
			}
		}
		if len(resp.Data.ProblemsetQuestionList.Questions) == 0 {
			break
		}
	}
	return "", fmt.Errorf("not found")
}

func (c *Client) GetQuestionBySlug(slug string) (*Question, error) {
	query := `query questionData($titleSlug: String!) {
		question(titleSlug: $titleSlug) {
			questionFrontendId
			title
			titleSlug
			difficulty
			content
			isPaidOnly
			topicTags { name }
			stats
			hints
			sampleTestCase
			exampleTestcases
			metaData
			codeSnippets { lang langSlug code }
		}
	}`

	var resp QuestionDetailResponse
	if err := c.doGraphQL(query, map[string]interface{}{"titleSlug": slug}, &resp); err != nil {
		return nil, err
	}

	return &resp.Data.Question, nil
}

// ─── Submit / Test ────────────────────────────────────────────────────────────

type InterpretResponse struct {
	InterpretId string `json:"interpret_id"`
}

type SubmitResponse struct {
	SubmissionId int `json:"submission_id"`
}

// CheckResponse covers both test-run and submit result responses.
// LeetCode uses the same check endpoint for both but with slightly
// different fields populated.
type CheckResponse struct {
	State   string `json:"state"`
	// "Accepted", "Wrong Answer", "Time Limit Exceeded",
	// "Runtime Error", "Compile Error", "Memory Limit Exceeded"
	StatusMsg  string `json:"status_msg"`
	StatusCode int    `json:"status_code"`

	RunSuccess bool `json:"run_success"`

	// Test run specific
	CorrectAnswer  bool     `json:"correct_answer"`
	TotalCorrect   int      `json:"total_correct"`
	TotalTestcases int      `json:"total_testcases"`
	CompareResult  string   `json:"compare_result"`
	CodeOutput     []string `json:"code_output"`
	StdOutputList  []string `json:"std_output_list"`
	ExpectedOutput string   `json:"expected_output"`
	TaskFinishTime int64    `json:"task_finish_time"`

	// Submit specific
	StatusRuntime     string  `json:"status_runtime"`
	StatusMemory      string  `json:"status_memory"`
	RuntimePercentile float64 `json:"runtime_percentile"`
	MemoryPercentile  float64 `json:"memory_percentile"`

	// Errors
	CompileError     string `json:"compile_error"`
	FullCompileError string `json:"full_compile_error"`
	RuntimeError     string `json:"runtime_error"`
	LastTestcase     string `json:"last_testcase"`
}

func (c *Client) doSubmitRequest(method, url, referer string, payload interface{}) ([]byte, error) {
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", referer)
	req.Header.Set("x-csrftoken", c.csrf)
	req.Header.Set("Origin", BaseURL)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.AddCookie(&http.Cookie{Name: "LEETCODE_SESSION", Value: c.session})
	req.AddCookie(&http.Cookie{Name: "csrftoken", Value: c.csrf})

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	return b, nil
}

// StripSubmitCode removes local-only sections before sending to LeetCode.
// It strips everything outside the // --- SUBMIT START --- / END markers if present,
// otherwise returns the code as-is.
func StripSubmitCode(code string) string {
	const startMarker = "// --- SUBMIT START ---"
	const endMarker = "// --- SUBMIT END ---"

	startIdx := strings.Index(code, startMarker)
	endIdx := strings.Index(code, endMarker)

	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		// No markers — return as-is
		return code
	}

	// Extract only what's between the markers
	inner := code[startIdx+len(startMarker) : endIdx]
	return strings.TrimSpace(inner)
}

func (c *Client) TestCode(slug, lang, code, testInput string) (string, error) {
	if c.session == "" {
		return "", fmt.Errorf("authentication required — run: lc auth")
	}

	q, err := c.GetQuestionBySlug(slug)
	if err != nil {
		return "", err
	}

	// Strip local harness before sending
	cleanCode := StripSubmitCode(code)

	payload := map[string]interface{}{
		"lang":        lang,
		"question_id": q.QuestionFrontendId,
		"typed_code":  cleanCode,
		"data_input":  testInput,
	}

	referer := fmt.Sprintf("%s/problems/%s/", BaseURL, slug)
	url := fmt.Sprintf("%s/problems/%s/interpret_solution/", BaseURL, slug)

	b, err := c.doSubmitRequest("POST", url, referer, payload)
	if err != nil {
		return "", err
	}

	var ir InterpretResponse
	json.Unmarshal(b, &ir)
	if ir.InterpretId == "" {
		return "", fmt.Errorf("empty interpret_id — check your session cookie")
	}
	return ir.InterpretId, nil
}

func (c *Client) SubmitCode(slug, lang, code string) (string, error) {
	if c.session == "" {
		return "", fmt.Errorf("authentication required — run: lc auth")
	}

	q, err := c.GetQuestionBySlug(slug)
	if err != nil {
		return "", err
	}

	// Always strip the local harness before submitting
	cleanCode := StripSubmitCode(code)

	payload := map[string]interface{}{
		"lang":        lang,
		"question_id": q.QuestionFrontendId,
		"typed_code":  cleanCode,
	}

	referer := fmt.Sprintf("%s/problems/%s/", BaseURL, slug)
	url := fmt.Sprintf("%s/problems/%s/submit/", BaseURL, slug)

	b, err := c.doSubmitRequest("POST", url, referer, payload)
	if err != nil {
		return "", err
	}

	var sr SubmitResponse
	json.Unmarshal(b, &sr)
	if sr.SubmissionId == 0 {
		return "", fmt.Errorf("empty submission_id — check your session cookie")
	}
	return fmt.Sprintf("%d", sr.SubmissionId), nil
}

func (c *Client) CheckResult(checkID string) (*CheckResponse, error) {
	url := fmt.Sprintf("%s/submissions/detail/%s/check/", BaseURL, checkID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	req.Header.Set("Referer", fmt.Sprintf("%s/submissions/", BaseURL))
	req.Header.Set("x-csrftoken", c.csrf)
	if c.session != "" {
		req.AddCookie(&http.Cookie{Name: "LEETCODE_SESSION", Value: c.session})
		req.AddCookie(&http.Cookie{Name: "csrftoken", Value: c.csrf})
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	var cr CheckResponse
	json.Unmarshal(b, &cr)
	return &cr, nil
}

func (c *Client) PollResult(checkID string, maxWait time.Duration) (*CheckResponse, error) {
	deadline := time.Now().Add(maxWait)
	interval := 1 * time.Second

	for time.Now().Before(deadline) {
		cr, err := c.CheckResult(checkID)
		if err != nil {
			return nil, err
		}

		// LeetCode returns state = "PENDING" | "STARTED" | "SUCCESS"
		// and status_code reflects the actual verdict only when state == "SUCCESS"
		if cr.State == "SUCCESS" {
			return cr, nil
		}

		time.Sleep(interval)
	}
	return nil, fmt.Errorf("timed out waiting for result after %s", maxWait)
}

// FetchImage downloads an image URL and returns raw bytes
func (c *Client) FetchImage(url string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", BaseURL)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	data, err := io.ReadAll(resp.Body)
	return data, contentType, err
}
