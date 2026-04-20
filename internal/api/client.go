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
	req.Header.Set("Origin", BaseURL)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("x-csrftoken", c.csrf)
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
			Date     string   `json:"date"`
			Link     string   `json:"link"`
			Question Question `json:"question"`
		} `json:"activeDailyCodingChallengeQuestion"`
	} `json:"data"`
}

func (c *Client) GetQuestionOfDay() (*Question, string, error) {
	query := `query questionOfToday {
		activeDailyCodingChallengeQuestion {
			date link
			question {
				questionFrontendId title titleSlug difficulty content
				topicTags { name } stats hints sampleTestCase exampleTestcases
				codeSnippets { lang langSlug code } metaData
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
	MetaData           string    `json:"metaData"`
	Status             string    `json:"status"` // "ac", "notac", null
}

type Tag struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Snippet struct {
	Lang     string `json:"lang"`
	LangSlug string `json:"langSlug"`
	Code     string `json:"code"`
}

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

// HasTopicTag returns true if question has a tag matching any of the given slugs.
func (q *Question) HasTopicTag(slugs ...string) bool {
	for _, t := range q.TopicTags {
		for _, s := range slugs {
			if strings.EqualFold(t.Slug, s) || strings.EqualFold(t.Name, s) {
				return true
			}
		}
	}
	return false
}

// ─── GetQuestionByNumber ──────────────────────────────────────────────────────

func (c *Client) GetQuestionByNumber(num string) (*Question, error) {
	listQuery := `query problemsetQuestionList($skip: Int, $limit: Int, $filters: QuestionListFilterInput) {
		problemsetQuestionList: questionList(
			categorySlug: "" limit: $limit skip: $skip filters: $filters
		) { data { questionFrontendId titleSlug title difficulty isPaidOnly } }
	}`
	limit := 50
	skip := 0
	var slug string
	for {
		vars := map[string]interface{}{"skip": skip, "limit": limit, "filters": map[string]interface{}{}}
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
			categorySlug: "" limit: $limit skip: $skip filters: $filters
		) { data { questionFrontendId titleSlug } }
	}`
	for skip := 0; skip <= 3500; skip += 500 {
		vars := map[string]interface{}{"skip": skip, "limit": 500, "filters": map[string]interface{}{}}
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
			questionFrontendId title titleSlug difficulty content isPaidOnly
			topicTags { name slug } stats hints sampleTestCase exampleTestcases
			metaData status codeSnippets { lang langSlug code }
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

type CheckResponse struct {
	State          string   `json:"state"`
	StatusMsg      string   `json:"status_msg"`
	StatusCode     int      `json:"status_code"`
	RunSuccess     bool     `json:"run_success"`
	CorrectAnswer  bool     `json:"correct_answer"`
	TotalCorrect   int      `json:"total_correct"`
	TotalTestcases int      `json:"total_testcases"`
	CompareResult  string   `json:"compare_result"`
	CodeOutput     []string `json:"code_output"`
	StdOutputList  []string `json:"std_output_list"`
	ExpectedOutput string   `json:"expected_output"`
	StatusRuntime     string  `json:"status_runtime"`
	StatusMemory      string  `json:"status_memory"`
	RuntimePercentile float64 `json:"runtime_percentile"`
	MemoryPercentile  float64 `json:"memory_percentile"`
	CompileError      string  `json:"compile_error"`
	FullCompileError  string  `json:"full_compile_error"`
	RuntimeError      string  `json:"runtime_error"`
	LastTestcase      string  `json:"last_testcase"`
}

func StripSubmitCode(code string) string {
	const startMarker = "// --- SUBMIT START ---"
	const endMarker = "// --- SUBMIT END ---"
	startIdx := strings.Index(code, startMarker)
	endIdx := strings.Index(code, endMarker)
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return code
	}
	return strings.TrimSpace(code[startIdx+len(startMarker) : endIdx])
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
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

func (c *Client) TestCode(slug, lang, code, testInput string) (string, error) {
	if c.session == "" {
		return "", fmt.Errorf("authentication required — run: lc auth")
	}
	q, err := c.GetQuestionBySlug(slug)
	if err != nil {
		return "", err
	}
	payload := map[string]interface{}{
		"lang": lang, "question_id": q.QuestionFrontendId,
		"typed_code": StripSubmitCode(code), "data_input": testInput,
	}
	b, err := c.doSubmitRequest("POST",
		fmt.Sprintf("%s/problems/%s/interpret_solution/", BaseURL, slug),
		fmt.Sprintf("%s/problems/%s/", BaseURL, slug), payload)
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
	payload := map[string]interface{}{
		"lang": lang, "question_id": q.QuestionFrontendId,
		"typed_code": StripSubmitCode(code),
	}
	b, err := c.doSubmitRequest("POST",
		fmt.Sprintf("%s/problems/%s/submit/", BaseURL, slug),
		fmt.Sprintf("%s/problems/%s/", BaseURL, slug), payload)
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
	req.Header.Set("User-Agent", "Mozilla/5.0")
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
	for time.Now().Before(deadline) {
		cr, err := c.CheckResult(checkID)
		if err != nil {
			return nil, err
		}
		if cr.State == "SUCCESS" {
			return cr, nil
		}
		time.Sleep(1 * time.Second)
	}
	return nil, fmt.Errorf("timed out waiting for result after %s", maxWait)
}

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

// ─── Profile ──────────────────────────────────────────────────────────────────

type UserProfile struct {
	MatchedUser *struct {
		Username string `json:"username"`
		Profile  struct {
			RealName    string `json:"realName"`
			AboutMe     string `json:"aboutMe"`
			CountryName string `json:"countryName"`
			Company     string `json:"company"`
			School      string `json:"school"`
			Ranking     int    `json:"ranking"`
			Reputation  int    `json:"reputation"`
		} `json:"profile"`
		SubmitStats struct {
			AcSubmissionNum []struct {
				Difficulty  string `json:"difficulty"`
				Count       int    `json:"count"`
				Submissions int    `json:"submissions"`
			} `json:"acSubmissionNum"`
			TotalSubmissionNum []struct {
				Difficulty string `json:"difficulty"`
				Count      int    `json:"count"`
			} `json:"totalSubmissionNum"`
		} `json:"submitStats"`
		Badges []struct {
			Name string `json:"name"`
		} `json:"badges"`
		ContestBadge *struct {
			Name    string `json:"name"`
			Expired bool   `json:"expired"`
		} `json:"contestBadge"`
		LanguageProblemCount []struct {
			LanguageName string `json:"languageName"`
			ProblemsSolved int  `json:"problemsSolved"`
		} `json:"languageProblemCount"`
	} `json:"matchedUser"`
	AllQuestionsCount []struct {
		Difficulty string `json:"difficulty"`
		Count      int    `json:"count"`
	} `json:"allQuestionsCount"`
	UserContestRanking *struct {
		AttendedContestsCount int     `json:"attendedContestsCount"`
		Rating                float64 `json:"rating"`
		GlobalRanking         int     `json:"globalRanking"`
		TotalParticipants     int     `json:"totalParticipants"`
		TopPercentage         float64 `json:"topPercentage"`
	} `json:"userContestRanking"`
}

type SubmissionCalendar struct {
	// map of unix_timestamp_string -> count
	Data map[string]int
}

func (c *Client) GetUserProfile(username string) (*UserProfile, error) {
	query := `query userProfile($username: String!) {
		allQuestionsCount { difficulty count }
		matchedUser(username: $username) {
			username
			profile { realName aboutMe countryName company school ranking reputation }
			submitStats {
				acSubmissionNum { difficulty count submissions }
				totalSubmissionNum { difficulty count }
			}
			badges { name }
			contestBadge { name expired }
			languageProblemCount { languageName problemsSolved }
		}
		userContestRanking(username: $username) {
			attendedContestsCount rating globalRanking totalParticipants topPercentage
		}
	}`
	var resp struct {
		Data UserProfile `json:"data"`
	}
	if err := c.doGraphQL(query, map[string]interface{}{"username": username}, &resp); err != nil {
		return nil, err
	}
	if resp.Data.MatchedUser == nil {
		return nil, fmt.Errorf("user %q not found", username)
	}
	return &resp.Data, nil
}

func (c *Client) GetSubmissionCalendar(username string) (map[string]int, error) {
	query := `query userProfileCalendar($username: String!, $year: Int) {
		matchedUser(username: $username) {
			userCalendar(year: $year) {
				submissionCalendar
				totalActiveDays
				streak
			}
		}
	}`

	merged := map[string]int{}
	now := time.Now()

	// Fetch both this year and last year to cover full 52-week window
	for _, year := range []int{now.Year() - 1, now.Year()} {
		var resp struct {
			Data struct {
				MatchedUser *struct {
					UserCalendar struct {
						SubmissionCalendar string `json:"submissionCalendar"`
					} `json:"userCalendar"`
				} `json:"matchedUser"`
			} `json:"data"`
		}
		if err := c.doGraphQL(query, map[string]interface{}{"username": username, "year": year}, &resp); err != nil {
			continue
		}
		if resp.Data.MatchedUser == nil {
			continue
		}
		cal := map[string]int{}
		if err := json.Unmarshal([]byte(resp.Data.MatchedUser.UserCalendar.SubmissionCalendar), &cal); err != nil {
			continue
		}
		for k, v := range cal {
			merged[k] += v
		}
	}

	if len(merged) == 0 {
		return nil, fmt.Errorf("no calendar data")
	}
	return merged, nil
}

// ─── Topic/Filter Browse ──────────────────────────────────────────────────────

type TopicTag struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ProblemItem struct {
	QuestionFrontendId string  `json:"questionFrontendId"`
	Title              string  `json:"title"`
	TitleSlug          string  `json:"titleSlug"`
	Difficulty         string  `json:"difficulty"`
	IsPaidOnly         bool    `json:"isPaidOnly"`
	TopicTags          []Tag   `json:"topicTags"`
	Status             string  `json:"status"`
	AcRate             float64 `json:"acRate"`
}

type ProblemListResult struct {
	Total    int           `json:"total"`
	Problems []ProblemItem `json:"data"`
}

func (c *Client) GetAllTopicTags() ([]TopicTag, error) {
	query := `query { topicTags { name slug } }`
	var resp struct {
		Data struct {
			TopicTags []TopicTag `json:"topicTags"`
		} `json:"data"`
	}
	if err := c.doGraphQL(query, nil, &resp); err != nil {
		return commonTopicTags(), nil
	}
	if len(resp.Data.TopicTags) == 0 {
		return commonTopicTags(), nil
	}
	return resp.Data.TopicTags, nil
}

func commonTopicTags() []TopicTag {
	return []TopicTag{
		{Name: "Array", Slug: "array"},
		{Name: "String", Slug: "string"},
		{Name: "Hash Table", Slug: "hash-table"},
		{Name: "Dynamic Programming", Slug: "dynamic-programming"},
		{Name: "Math", Slug: "math"},
		{Name: "Sorting", Slug: "sorting"},
		{Name: "Greedy", Slug: "greedy"},
		{Name: "Depth-First Search", Slug: "depth-first-search"},
		{Name: "Binary Search", Slug: "binary-search"},
		{Name: "Tree", Slug: "tree"},
		{Name: "Breadth-First Search", Slug: "breadth-first-search"},
		{Name: "Matrix", Slug: "matrix"},
		{Name: "Two Pointers", Slug: "two-pointers"},
		{Name: "Bit Manipulation", Slug: "bit-manipulation"},
		{Name: "Stack", Slug: "stack"},
		{Name: "Graph", Slug: "graph"},
		{Name: "Sliding Window", Slug: "sliding-window"},
		{Name: "Backtracking", Slug: "backtracking"},
		{Name: "Heap (Priority Queue)", Slug: "heap-priority-queue"},
		{Name: "Linked List", Slug: "linked-list"},
		{Name: "Prefix Sum", Slug: "prefix-sum"},
		{Name: "Simulation", Slug: "simulation"},
		{Name: "Counting", Slug: "counting"},
		{Name: "Union Find", Slug: "union-find"},
		{Name: "Recursion", Slug: "recursion"},
		{Name: "Trie", Slug: "trie"},
		{Name: "Divide and Conquer", Slug: "divide-and-conquer"},
		{Name: "Queue", Slug: "queue"},
		{Name: "Memoization", Slug: "memoization"},
		{Name: "Monotonic Stack", Slug: "monotonic-stack"},
		{Name: "Binary Search Tree", Slug: "binary-search-tree"},
		{Name: "Segment Tree", Slug: "segment-tree"},
		{Name: "Number Theory", Slug: "number-theory"},
		{Name: "Design", Slug: "design"},
		{Name: "Game Theory", Slug: "game-theory"},
		{Name: "Geometry", Slug: "geometry"},
	}
}

func (c *Client) GetProblemsByTopic(topicSlug, difficulty, status string, skip, limit int) (*ProblemListResult, error) {
	query := `query problemsetQuestionList($categorySlug: String, $limit: Int, $skip: Int, $filters: QuestionListFilterInput) {
		problemsetQuestionList: questionList(
			categorySlug: $categorySlug limit: $limit skip: $skip filters: $filters
		) {
			total: totalNum
			data {
				questionFrontendId title titleSlug difficulty
				isPaidOnly acRate status topicTags { name slug }
			}
		}
	}`
	filters := map[string]interface{}{}
	if topicSlug != "" {
		filters["tags"] = []string{topicSlug}
	}
	if difficulty != "" && difficulty != "ALL" {
		filters["difficulty"] = strings.ToUpper(difficulty)
	}
	if status == "ac" {
		filters["status"] = "AC"
	} else if status == "notac" {
		filters["status"] = "NOT_STARTED"
	}
	vars := map[string]interface{}{
		"categorySlug": "", "skip": skip, "limit": limit, "filters": filters,
	}
	var resp struct {
		Data struct {
			ProblemsetQuestionList ProblemListResult `json:"problemsetQuestionList"`
		} `json:"data"`
	}
	if err := c.doGraphQL(query, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.Data.ProblemsetQuestionList, nil
}

func (c *Client) GetCurrentUsername() (string, error) {
	query := `query { userStatus { username isSignedIn } }`
	var resp struct {
		Data struct {
			UserStatus struct {
				Username   string `json:"username"`
				IsSignedIn bool   `json:"isSignedIn"`
			} `json:"userStatus"`
		} `json:"data"`
	}
	if err := c.doGraphQL(query, nil, &resp); err != nil {
		return "", err
	}
	if !resp.Data.UserStatus.IsSignedIn {
		return "", fmt.Errorf("not signed in")
	}
	return resp.Data.UserStatus.Username, nil
}
