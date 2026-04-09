// Package codegen wraps LeetCode snippets in a locally runnable harness.
//
// The generated file has this structure:
//
//	<local imports / boilerplate>
//	// --- SUBMIT START ---
//	<exact LeetCode snippet>
//	// --- SUBMIT END ---
//	<local test harness / main function>
//
// When submitting, api.StripSubmitCode() extracts only the inner section.
// When running locally, the full file compiles and runs.
package codegen

import (
	"fmt"
	"strings"

	"github.com/user/leetcode-cli/internal/api"
)

// Wrap generates a locally-runnable file for the given question and snippet.
func Wrap(q *api.Question, snippet api.Snippet, testcases string) string {
	switch snippet.LangSlug {
	case "cpp":
		return wrapCPP(snippet.Code, testcases, q)
	case "golang", "go":
		return wrapGo(snippet.Code, testcases, q)
	case "python3", "python":
		return wrapPython(snippet.Code, testcases)
	case "java":
		return wrapJava(snippet.Code, testcases, q)
	case "javascript":
		return wrapJS(snippet.Code, testcases)
	case "typescript":
		return wrapTS(snippet.Code, testcases)
	case "rust":
		return wrapRust(snippet.Code, testcases)
	default:
		// Generic: just wrap with markers and a comment
		return fmt.Sprintf("// Run locally with your language toolchain\n// Remove the harness section before submitting\n\n// --- SUBMIT START ---\n%s\n// --- SUBMIT END ---\n", snippet.Code)
	}
}

// ─── C++ ──────────────────────────────────────────────────────────────────────

func wrapCPP(code, testcases string, q *api.Question) string {
	examples := formatTestCasesAsComments(testcases)
	return fmt.Sprintf(`#include <bits/stdc++.h>
using namespace std;

// ╔══════════════════════════════════════════════════════╗
// ║  LeetCode #%s — %s
// ║  Difficulty: %s
// ║
// ║  HOW TO USE:
// ║  • Run locally:  g++ -std=c++17 -o sol solution.cpp && ./sol
// ║  • Submit:       lc submit %s
// ║    (the harness below is auto-stripped before submit)
// ╚══════════════════════════════════════════════════════╝

// ─── Example Test Cases ───────────────────────────────
%s
// ─────────────────────────────────────────────────────

// --- SUBMIT START ---
%s
// --- SUBMIT END ---

// ─── Local Test Harness ───────────────────────────────
// Edit the test cases in main() to test your solution.
// This section is NOT submitted to LeetCode.

int main() {
    Solution sol;

    // TODO: Add your test cases here
    // Example (adjust types to match the problem):
    //
    // auto result = sol.twoSum({2, 7, 11, 15}, 9);
    // vector<int> expected = {0, 1};
    // if (result == expected) {
    //     cout << "✔ Test passed" << endl;
    // } else {
    //     cout << "✘ Test failed" << endl;
    //     cout << "  Got:      ";
    //     for (auto v : result) cout << v << " ";
    //     cout << endl;
    //     cout << "  Expected: ";
    //     for (auto v : expected) cout << v << " ";
    //     cout << endl;
    // }

    cout << "Edit main() to add your test cases." << endl;
    return 0;
}
`,
		q.QuestionFrontendId, q.Title, q.Difficulty,
		q.QuestionFrontendId,
		examples, code)
}

// ─── Go ───────────────────────────────────────────────────────────────────────

func wrapGo(code, testcases string, q *api.Question) string {
	examples := formatTestCasesAsComments(testcases)
	return fmt.Sprintf(`package main

import "fmt"

// ╔══════════════════════════════════════════════════════╗
// ║  LeetCode #%s — %s
// ║  Difficulty: %s
// ║
// ║  HOW TO USE:
// ║  • Run locally:  go run solution.go
// ║  • Submit:       lc submit %s
// ╚══════════════════════════════════════════════════════╝

// ─── Example Test Cases ───────────────────────────────
%s
// ─────────────────────────────────────────────────────

// --- SUBMIT START ---
%s
// --- SUBMIT END ---

// ─── Local Test Harness ───────────────────────────────

func main() {
    // TODO: add test cases here
    // Example (adjust types to match the problem):
    //
    // result := twoSum([]int{2, 7, 11, 15}, 9)
    // expected := []int{0, 1}
    // if reflect.DeepEqual(result, expected) {
    //     fmt.Println("✔ Test passed")
    // } else {
    //     fmt.Printf("✘ Got %%v, want %%v\n", result, expected)
    // }

    fmt.Println("Edit main() to add your test cases.")
}
`,
		q.QuestionFrontendId, q.Title, q.Difficulty,
		q.QuestionFrontendId,
		examples, code)
}

// ─── Python ───────────────────────────────────────────────────────────────────

func wrapPython(code, testcases string) string {
	examples := formatTestCasesAsPythonComments(testcases)
	return fmt.Sprintf(`#!/usr/bin/env python3
# HOW TO USE:
#   Run locally:  python3 solution.py
#   Submit:       lc submit <number>

# ─── Example Test Cases ───────────────────────────────
%s
# ─────────────────────────────────────────────────────

# --- SUBMIT START ---
%s
# --- SUBMIT END ---

# ─── Local Test Harness ───────────────────────────────
if __name__ == "__main__":
    sol = Solution()

    # TODO: Add your test cases here
    # Example:
    # result = sol.twoSum([2, 7, 11, 15], 9)
    # expected = [0, 1]
    # if result == expected:
    #     print("✔ Test passed")
    # else:
    #     print(f"✘ Got {result}, want {expected}")

    print("Edit __main__ to add your test cases.")
`,
		examples, code)
}

// ─── Java ─────────────────────────────────────────────────────────────────────

func wrapJava(code, testcases string, q *api.Question) string {
	examples := formatTestCasesAsComments(testcases)

	// LeetCode Java snippets are just the class body — wrap in a runnable class
	return fmt.Sprintf(`import java.util.*;

// ╔══════════════════════════════════════════════════════╗
// ║  LeetCode #%s — %s
// ║  Difficulty: %s
// ║
// ║  HOW TO USE:
// ║  • Run locally:  javac Solution.java && java Solution
// ║  • Submit:       lc submit %s
// ╚══════════════════════════════════════════════════════╝

// ─── Example Test Cases ───────────────────────────────
%s
// ─────────────────────────────────────────────────────

// --- SUBMIT START ---
%s
// --- SUBMIT END ---

// ─── Local Test Harness ───────────────────────────────
// This wraps Solution in a runnable class for local testing.
class Main {
    public static void main(String[] args) {
        Solution sol = new Solution();

        // TODO: Add your test cases here
        // Example:
        // int[] result = sol.twoSum(new int[]{2, 7, 11, 15}, 9);
        // int[] expected = {0, 1};
        // if (Arrays.equals(result, expected)) {
        //     System.out.println("✔ Test passed");
        // } else {
        //     System.out.println("✘ Got " + Arrays.toString(result));
        // }

        System.out.println("Edit main() to add your test cases.");
    }
}
`,
		q.QuestionFrontendId, q.Title, q.Difficulty,
		q.QuestionFrontendId,
		examples, code)
}

// ─── JavaScript ───────────────────────────────────────────────────────────────

func wrapJS(code, testcases string) string {
	examples := formatTestCasesAsComments(testcases)
	return fmt.Sprintf(`// HOW TO USE:
//   Run locally:  node solution.js
//   Submit:       lc submit <number>

// ─── Example Test Cases ───────────────────────────────
%s
// ─────────────────────────────────────────────────────

// --- SUBMIT START ---
%s
// --- SUBMIT END ---

// ─── Local Test Harness ───────────────────────────────
(function () {
    // TODO: Add your test cases here
    // Example:
    // const result = twoSum([2, 7, 11, 15], 9);
    // const expected = JSON.stringify([0, 1]);
    // if (JSON.stringify(result) === expected) {
    //     console.log("✔ Test passed");
    // } else {
    //     console.log("✘ Got", result, "want", expected);
    // }

    console.log("Edit the harness to add your test cases.");
})();
`,
		examples, code)
}

// ─── TypeScript ───────────────────────────────────────────────────────────────

func wrapTS(code, testcases string) string {
	examples := formatTestCasesAsComments(testcases)
	return fmt.Sprintf(`// HOW TO USE:
//   Run locally:  ts-node solution.ts   (or: npx ts-node solution.ts)
//   Submit:       lc submit <number>

// ─── Example Test Cases ───────────────────────────────
%s
// ─────────────────────────────────────────────────────

// --- SUBMIT START ---
%s
// --- SUBMIT END ---

// ─── Local Test Harness ───────────────────────────────
(function () {
    // TODO: Add your test cases here

    console.log("Edit the harness to add your test cases.");
})();
`,
		examples, code)
}

// ─── Rust ─────────────────────────────────────────────────────────────────────

func wrapRust(code, testcases string) string {
	examples := formatTestCasesAsComments(testcases)
	return fmt.Sprintf(`// HOW TO USE:
//   Run locally:  rustc solution.rs && ./solution
//   Submit:       lc submit <number>

// ─── Example Test Cases ───────────────────────────────
%s
// ─────────────────────────────────────────────────────

// --- SUBMIT START ---
%s
// --- SUBMIT END ---

// ─── Local Test Harness ───────────────────────────────
fn main() {
    // TODO: Add your test cases here
    // Example:
    // let result = Solution::two_sum(vec![2, 7, 11, 15], 9);
    // assert_eq!(result, vec![0, 1], "Test failed");
    // println!("✔ Test passed");

    println!("Edit main() to add your test cases.");
}
`,
		examples, code)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func formatTestCasesAsComments(testcases string) string {
	if testcases == "" {
		return "// (no example test cases available)"
	}
	lines := strings.Split(strings.TrimSpace(testcases), "\n")
	out := make([]string, 0, len(lines)+2)
	out = append(out, "// Input:")
	for _, l := range lines {
		out = append(out, "//   "+l)
	}
	return strings.Join(out, "\n")
}

func formatTestCasesAsPythonComments(testcases string) string {
	if testcases == "" {
		return "# (no example test cases available)"
	}
	lines := strings.Split(strings.TrimSpace(testcases), "\n")
	out := make([]string, 0, len(lines)+2)
	out = append(out, "# Input:")
	for _, l := range lines {
		out = append(out, "#   "+l)
	}
	return strings.Join(out, "\n")
}
