// Package codegen generates locally-runnable solution files.
//
// It detects the data structures used in the question (Tree, LinkedList, Graph,
// Matrix, etc.) by examining topic tags and the function signature metadata,
// then injects the appropriate helper definitions and test harness so you can
// run and debug locally with zero boilerplate.
//
// File structure:
//
//	<helper definitions for detected data structures>
//	// --- SUBMIT START ---
//	<exact LeetCode snippet>
//	// --- SUBMIT END ---
//	<pre-filled local test harness / main()>
//
// Only the SUBMIT START..END block is sent to LeetCode on submit.
package codegen

import (
	"fmt"
	"strings"

	"github.com/user/leetcode-cli/internal/api"
)

// Wrap produces a complete locally-runnable file for the given snippet.
func Wrap(q *api.Question, snippet api.Snippet, testcases string) string {
	ds := detectDataStructures(q)
	tc := formatTestCases(testcases, snippet.LangSlug)

	switch snippet.LangSlug {
	case "cpp":
		return wrapCPP(q, snippet.Code, tc, ds)
	case "golang", "go":
		return wrapGo(q, snippet.Code, tc, ds)
	case "python3", "python":
		return wrapPython(q, snippet.Code, tc, ds)
	case "java":
		return wrapJava(q, snippet.Code, tc, ds)
	case "javascript":
		return wrapJS(q, snippet.Code, tc)
	case "rust":
		return wrapRust(q, snippet.Code, tc)
	default:
		return fmt.Sprintf("// Run with your toolchain\n\n// --- SUBMIT START ---\n%s\n// --- SUBMIT END ---\n", snippet.Code)
	}
}

// ─── Data Structure Detection ─────────────────────────────────────────────────

type dsFlags struct {
	tree       bool
	linkedList bool
	graph      bool
	matrix     bool
	trie       bool
	heap       bool
}

func detectDataStructures(q *api.Question) dsFlags {
	var f dsFlags
	tagSlugs := map[string]bool{}
	for _, t := range q.TopicTags {
		tagSlugs[strings.ToLower(t.Slug)] = true
		tagSlugs[strings.ToLower(t.Name)] = true
	}

	f.tree = tagSlugs["tree"] || tagSlugs["binary-tree"] || tagSlugs["binary-search-tree"] ||
		tagSlugs["n-ary-tree"] || strings.Contains(strings.ToLower(q.Title), "tree") ||
		strings.Contains(strings.ToLower(q.Content), "treenode") ||
		strings.Contains(strings.ToLower(snippet(q)), "treenode")

	f.linkedList = tagSlugs["linked-list"] ||
		strings.Contains(strings.ToLower(q.Content), "listnode") ||
		strings.Contains(strings.ToLower(snippet(q)), "listnode")

	f.graph = tagSlugs["graph"] || tagSlugs["depth-first-search"] ||
		tagSlugs["breadth-first-search"] || tagSlugs["union-find"]

	f.matrix = tagSlugs["matrix"] ||
		strings.Contains(strings.ToLower(q.Content), "grid") ||
		strings.Contains(strings.ToLower(q.Content), "matrix")

	f.trie = tagSlugs["trie"]
	f.heap = tagSlugs["heap-priority-queue"] || tagSlugs["heap"]

	return f
}

func snippet(q *api.Question) string {
	for _, s := range q.CodeSnippets {
		return s.Code
	}
	return ""
}

// ─── C++ ──────────────────────────────────────────────────────────────────────

func wrapCPP(q *api.Question, code, tc string, ds dsFlags) string {
	var sb strings.Builder

	sb.WriteString(`#include <bits/stdc++.h>
using namespace std;
`)

	// ── Struct helpers ────────────────────────────────────────────────────────
	if ds.tree {
		sb.WriteString(`
// ── TreeNode ──────────────────────────────────────────────────
struct TreeNode {
    int val;
    TreeNode *left, *right;
    TreeNode() : val(0), left(nullptr), right(nullptr) {}
    TreeNode(int x) : val(x), left(nullptr), right(nullptr) {}
    TreeNode(int x, TreeNode *l, TreeNode *r) : val(x), left(l), right(r) {}
};

// Build tree from level-order array: {1,2,3,null,4} → tree
// Use INT_MIN to represent null nodes.
TreeNode* buildTree(vector<int> vals) {
    if (vals.empty() || vals[0] == INT_MIN) return nullptr;
    TreeNode* root = new TreeNode(vals[0]);
    queue<TreeNode*> q;
    q.push(root);
    int i = 1;
    while (!q.empty() && i < (int)vals.size()) {
        TreeNode* node = q.front(); q.pop();
        if (i < (int)vals.size() && vals[i] != INT_MIN) {
            node->left = new TreeNode(vals[i]); q.push(node->left);
        }
        i++;
        if (i < (int)vals.size() && vals[i] != INT_MIN) {
            node->right = new TreeNode(vals[i]); q.push(node->right);
        }
        i++;
    }
    return root;
}

// Serialize tree to level-order for easy printing
vector<string> treeToVec(TreeNode* root) {
    vector<string> res;
    if (!root) return res;
    queue<TreeNode*> q;
    q.push(root);
    while (!q.empty()) {
        TreeNode* n = q.front(); q.pop();
        if (n) { res.push_back(to_string(n->val)); q.push(n->left); q.push(n->right); }
        else res.push_back("null");
    }
    // trim trailing nulls
    while (!res.empty() && res.back() == "null") res.pop_back();
    return res;
}

void printTree(TreeNode* root) {
    auto v = treeToVec(root);
    cout << "[";
    for (int i = 0; i < (int)v.size(); i++) { if(i) cout << ","; cout << v[i]; }
    cout << "]" << endl;
}
`)
	}

	if ds.linkedList {
		sb.WriteString(`
// ── ListNode ───────────────────────────────────────────────────
struct ListNode {
    int val;
    ListNode *next;
    ListNode() : val(0), next(nullptr) {}
    ListNode(int x) : val(x), next(nullptr) {}
    ListNode(int x, ListNode *next) : val(x), next(next) {}
};

// Build linked list from vector: {1,2,3} → 1->2->3
ListNode* buildList(vector<int> vals) {
    ListNode dummy(0);
    ListNode* cur = &dummy;
    for (int v : vals) { cur->next = new ListNode(v); cur = cur->next; }
    return dummy.next;
}

// Build cycle list: vals with pos=-1 means no cycle
ListNode* buildCycleList(vector<int> vals, int pos) {
    ListNode dummy(0);
    ListNode* cur = &dummy;
    ListNode* cycleNode = nullptr;
    int i = 0;
    for (int v : vals) {
        cur->next = new ListNode(v); cur = cur->next;
        if (i == pos) cycleNode = cur;
        i++;
    }
    if (cycleNode) cur->next = cycleNode;
    return dummy.next;
}

void printList(ListNode* head) {
    cout << "[";
    set<ListNode*> seen;
    bool first = true;
    while (head && seen.find(head) == seen.end()) {
        if (!first) cout << ",";
        cout << head->val;
        seen.insert(head);
        head = head->next;
        first = false;
    }
    if (head) cout << "... (cycle)";
    cout << "]" << endl;
}
`)
	}

	// ── Banner ────────────────────────────────────────────────────────────────
	sb.WriteString(fmt.Sprintf(`
// ╔══════════════════════════════════════════════════════╗
// ║  LeetCode #%s — %s
// ║  Difficulty : %s
// ║
// ║  Run locally : g++ -std=c++17 -o sol solution.cpp && ./sol
// ║  Submit      : lc submit %s  (harness auto-stripped)
// ╚══════════════════════════════════════════════════════╝
%s
`,
		q.QuestionFrontendId, q.Title, q.Difficulty,
		q.QuestionFrontendId, tc,
	))

	// ── Solution ──────────────────────────────────────────────────────────────
	sb.WriteString("// --- SUBMIT START ---\n")
	sb.WriteString(code)
	sb.WriteString("\n// --- SUBMIT END ---\n")

	// ── Test Harness ──────────────────────────────────────────────────────────
	sb.WriteString(`
// ═══════════════════════════════════════════════════════
// LOCAL TEST HARNESS — not submitted to LeetCode
// ═══════════════════════════════════════════════════════
int main() {
    Solution sol;
`)
	if ds.tree {
		sb.WriteString(`
    // Example: build a tree from level-order {4,2,7,1,3,6,9}
    // INT_MIN = null node
    TreeNode* root = buildTree({4, 2, 7, 1, 3, 6, 9});

    // TODO: call your solution and print result
    // Example for invertTree:
    // TreeNode* result = sol.invertTree(root);
    // printTree(result);
`)
	}
	if ds.linkedList {
		sb.WriteString(`
    // Example: build linked list 1->2->3->4->5
    ListNode* head = buildList({1, 2, 3, 4, 5});

    // TODO: call your solution and print result
    // Example for reverseList:
    // ListNode* result = sol.reverseList(head);
    // printList(result);
`)
	}
	if ds.matrix {
		sb.WriteString(`
    // Example: build a 3x3 grid
    vector<vector<int>> grid = {
        {1, 1, 0},
        {1, 0, 0},
        {0, 0, 1}
    };
    // TODO: call your solution
    // cout << sol.numIslands(grid) << endl;
`)
	}
	if ds.graph {
		sb.WriteString(`
    // Example: adjacency list graph with 5 nodes, 0-indexed
    int n = 5;
    vector<vector<int>> edges = {{0,1},{1,2},{2,3},{3,4}};
    // TODO: call your solution
    // cout << sol.method(n, edges) << endl;
`)
	}
	if !ds.tree && !ds.linkedList && !ds.matrix && !ds.graph {
		sb.WriteString(`
    // TODO: add test cases. Examples:
    // cout << sol.methodName(arg1, arg2) << endl;
    // assert(sol.methodName(arg1) == expected);
`)
	}
	sb.WriteString(`
    cout << "Done." << endl;
    return 0;
}
`)
	return sb.String()
}

// ─── Go ───────────────────────────────────────────────────────────────────────

func wrapGo(q *api.Question, code, tc string, ds dsFlags) string {
	var sb strings.Builder

	sb.WriteString("package main\n\nimport (\n\t\"fmt\"\n)\n")

	if ds.tree {
		sb.WriteString(`
// ── TreeNode ──────────────────────────────────────────────────
type TreeNode struct {
    Val   int
    Left  *TreeNode
    Right *TreeNode
}

// BuildTree builds a tree from level-order slice. Use -1 for null nodes.
func BuildTree(vals []int) *TreeNode {
    if len(vals) == 0 || vals[0] == -1 { return nil }
    root := &TreeNode{Val: vals[0]}
    queue := []*TreeNode{root}
    i := 1
    for len(queue) > 0 && i < len(vals) {
        node := queue[0]; queue = queue[1:]
        if i < len(vals) && vals[i] != -1 {
            node.Left = &TreeNode{Val: vals[i]}; queue = append(queue, node.Left)
        }
        i++
        if i < len(vals) && vals[i] != -1 {
            node.Right = &TreeNode{Val: vals[i]}; queue = append(queue, node.Right)
        }
        i++
    }
    return root
}

func PrintTree(root *TreeNode) {
    if root == nil { fmt.Println("[]"); return }
    res := []string{}
    queue := []*TreeNode{root}
    for len(queue) > 0 {
        n := queue[0]; queue = queue[1:]
        if n != nil { res = append(res, fmt.Sprintf("%d", n.Val)); queue = append(queue, n.Left, n.Right) } else { res = append(res, "null") }
    }
    for len(res) > 0 && res[len(res)-1] == "null" { res = res[:len(res)-1] }
    fmt.Printf("[%s]\n", joinStr(res, ","))
}

func joinStr(s []string, sep string) string {
    r := ""; for i, v := range s { if i > 0 { r += sep }; r += v }; return r
}
`)
	}

	if ds.linkedList {
		sb.WriteString(`
// ── ListNode ───────────────────────────────────────────────────
type ListNode struct {
    Val  int
    Next *ListNode
}

// BuildList builds a linked list from a slice.
func BuildList(vals []int) *ListNode {
    dummy := &ListNode{}; cur := dummy
    for _, v := range vals { cur.Next = &ListNode{Val: v}; cur = cur.Next }
    return dummy.Next
}

func PrintList(head *ListNode) {
    seen := map[*ListNode]bool{}
    fmt.Print("[")
    first := true
    for head != nil && !seen[head] {
        if !first { fmt.Print(",") }
        fmt.Print(head.Val); seen[head] = true; head = head.Next; first = false
    }
    if head != nil { fmt.Print("...(cycle)") }
    fmt.Println("]")
}
`)
	}

	sb.WriteString(fmt.Sprintf(`
// ╔══════════════════════════════════════════════════════╗
// ║  LeetCode #%s — %s
// ║  Difficulty : %s
// ║  Run        : go run solution.go
// ║  Submit     : lc submit %s
// ╚══════════════════════════════════════════════════════╝
%s

// --- SUBMIT START ---
%s
// --- SUBMIT END ---

func main() {
`,
		q.QuestionFrontendId, q.Title, q.Difficulty,
		q.QuestionFrontendId, tc, code,
	))

	if ds.tree {
		sb.WriteString("\troot := BuildTree([]int{4, 2, 7, 1, 3, 6, 9}) // -1 = null\n")
		sb.WriteString("\t_ = root\n")
		sb.WriteString("\t// result := invertTree(root)\n\t// PrintTree(result)\n")
	}
	if ds.linkedList {
		sb.WriteString("\thead := BuildList([]int{1, 2, 3, 4, 5})\n")
		sb.WriteString("\t_ = head\n")
		sb.WriteString("\t// result := reverseList(head)\n\t// PrintList(result)\n")
	}
	if ds.matrix {
		sb.WriteString("\tgrid := [][]int{{1, 1, 0}, {1, 0, 0}, {0, 0, 1}}\n")
		sb.WriteString("\t_ = grid\n\t// fmt.Println(numIslands(grid))\n")
	}
	if !ds.tree && !ds.linkedList && !ds.matrix {
		sb.WriteString("\t// TODO: add test cases\n\t// fmt.Println(twoSum([]int{2,7,11,15}, 9))\n")
	}
	sb.WriteString("\tfmt.Println(\"Done.\")\n}\n")
	return sb.String()
}

// ─── Python ───────────────────────────────────────────────────────────────────

func wrapPython(q *api.Question, code, tc string, ds dsFlags) string {
	var sb strings.Builder

	sb.WriteString("#!/usr/bin/env python3\n")
	sb.WriteString(fmt.Sprintf("# LeetCode #%s — %s  [%s]\n", q.QuestionFrontendId, q.Title, q.Difficulty))
	sb.WriteString(fmt.Sprintf("# Run: python3 solution.py  |  Submit: lc submit %s\n\n", q.QuestionFrontendId))

	if ds.tree || ds.linkedList {
		sb.WriteString("from collections import deque\nfrom typing import Optional, List\n\n")
	} else {
		sb.WriteString("from typing import List, Optional\n\n")
	}

	if ds.tree {
		sb.WriteString(`class TreeNode:
    def __init__(self, val=0, left=None, right=None):
        self.val = val; self.left = left; self.right = right

def build_tree(vals: List) -> Optional['TreeNode']:
    """Build from level-order list. Use None for null nodes."""
    if not vals or vals[0] is None: return None
    root = TreeNode(vals[0])
    q = deque([root]); i = 1
    while q and i < len(vals):
        node = q.popleft()
        if i < len(vals) and vals[i] is not None:
            node.left = TreeNode(vals[i]); q.append(node.left)
        i += 1
        if i < len(vals) and vals[i] is not None:
            node.right = TreeNode(vals[i]); q.append(node.right)
        i += 1
    return root

def print_tree(root) -> None:
    if not root: print("[]"); return
    res, q = [], deque([root])
    while q:
        n = q.popleft()
        if n: res.append(str(n.val)); q.append(n.left); q.append(n.right)
        else: res.append("null")
    while res and res[-1] == "null": res.pop()
    print("[" + ",".join(res) + "]")

`)
	}

	if ds.linkedList {
		sb.WriteString(`class ListNode:
    def __init__(self, val=0, next=None):
        self.val = val; self.next = next

def build_list(vals: List[int]) -> Optional['ListNode']:
    dummy = ListNode(); cur = dummy
    for v in vals: cur.next = ListNode(v); cur = cur.next
    return dummy.next

def print_list(head) -> None:
    seen, vals = set(), []
    while head and id(head) not in seen:
        vals.append(str(head.val)); seen.add(id(head)); head = head.next
    if head: vals.append("...(cycle)")
    print("[" + " -> ".join(vals) + "]")

`)
	}

	sb.WriteString(tc + "\n\n")
	sb.WriteString("# --- SUBMIT START ---\n")
	sb.WriteString(code)
	sb.WriteString("\n# --- SUBMIT END ---\n\n")
	sb.WriteString("if __name__ == \"__main__\":\n    sol = Solution()\n\n")

	if ds.tree {
		sb.WriteString("    # root = build_tree([4, 2, 7, 1, 3, 6, 9])  # None = null\n")
		sb.WriteString("    # print_tree(sol.invertTree(root))\n\n")
	}
	if ds.linkedList {
		sb.WriteString("    # head = build_list([1, 2, 3, 4, 5])\n")
		sb.WriteString("    # print_list(sol.reverseList(head))\n\n")
	}
	if ds.matrix {
		sb.WriteString("    # grid = [[1,1,0],[1,0,0],[0,0,1]]\n")
		sb.WriteString("    # print(sol.numIslands(grid))\n\n")
	}
	if !ds.tree && !ds.linkedList && !ds.matrix {
		sb.WriteString("    # print(sol.methodName(arg1, arg2))\n\n")
	}
	sb.WriteString("    print('Done.')\n")
	return sb.String()
}

// ─── Java ─────────────────────────────────────────────────────────────────────

func wrapJava(q *api.Question, code, tc string, ds dsFlags) string {
	var sb strings.Builder
	sb.WriteString("import java.util.*;\n\n")

	if ds.tree {
		sb.WriteString(`class TreeNode {
    int val; TreeNode left, right;
    TreeNode() {}
    TreeNode(int val) { this.val = val; }
    TreeNode(int val, TreeNode left, TreeNode right) { this.val=val; this.left=left; this.right=right; }

    static TreeNode build(Integer[] vals) {
        if (vals == null || vals.length == 0 || vals[0] == null) return null;
        TreeNode root = new TreeNode(vals[0]);
        Queue<TreeNode> q = new LinkedList<>(); q.add(root);
        int i = 1;
        while (!q.isEmpty() && i < vals.length) {
            TreeNode n = q.poll();
            if (i < vals.length && vals[i] != null) { n.left = new TreeNode(vals[i]); q.add(n.left); } i++;
            if (i < vals.length && vals[i] != null) { n.right = new TreeNode(vals[i]); q.add(n.right); } i++;
        }
        return root;
    }
    static void print(TreeNode root) {
        List<String> res = new ArrayList<>();
        Queue<TreeNode> q = new LinkedList<>(); q.add(root);
        while (!q.isEmpty()) {
            TreeNode n = q.poll();
            if (n != null) { res.add(String.valueOf(n.val)); q.add(n.left); q.add(n.right); }
            else res.add("null");
        }
        while (!res.isEmpty() && res.get(res.size()-1).equals("null")) res.remove(res.size()-1);
        System.out.println(res);
    }
}

`)
	}

	if ds.linkedList {
		sb.WriteString(`class ListNode {
    int val; ListNode next;
    ListNode() {} ListNode(int val) { this.val = val; }
    ListNode(int val, ListNode next) { this.val = val; this.next = next; }

    static ListNode build(int[] vals) {
        ListNode dummy = new ListNode(), cur = dummy;
        for (int v : vals) { cur.next = new ListNode(v); cur = cur.next; }
        return dummy.next;
    }
    static void print(ListNode head) {
        StringBuilder sb = new StringBuilder("[");
        Set<ListNode> seen = new HashSet<>();
        while (head != null && !seen.contains(head)) {
            sb.append(head.val).append(","); seen.add(head); head = head.next;
        }
        if (sb.length() > 1) sb.deleteCharAt(sb.length()-1);
        sb.append("]"); System.out.println(sb);
    }
}

`)
	}

	sb.WriteString(fmt.Sprintf("// LeetCode #%s — %s [%s]\n// Run: javac solution.java && java Main\n// Submit: lc submit %s\n\n",
		q.QuestionFrontendId, q.Title, q.Difficulty, q.QuestionFrontendId))
	sb.WriteString(tc + "\n\n")
	sb.WriteString("// --- SUBMIT START ---\n")
	sb.WriteString(code)
	sb.WriteString("\n// --- SUBMIT END ---\n\n")
	sb.WriteString("class Main {\n    public static void main(String[] args) {\n        Solution sol = new Solution();\n")

	if ds.tree {
		sb.WriteString("        TreeNode root = TreeNode.build(new Integer[]{4,2,7,1,3,6,9});\n")
		sb.WriteString("        // TreeNode.print(sol.invertTree(root));\n")
	}
	if ds.linkedList {
		sb.WriteString("        ListNode head = ListNode.build(new int[]{1,2,3,4,5});\n")
		sb.WriteString("        // ListNode.print(sol.reverseList(head));\n")
	}
	if !ds.tree && !ds.linkedList {
		sb.WriteString("        // System.out.println(sol.methodName(arg1, arg2));\n")
	}
	sb.WriteString("        System.out.println(\"Done.\");\n    }\n}\n")
	return sb.String()
}

// ─── JavaScript ───────────────────────────────────────────────────────────────

func wrapJS(q *api.Question, code, tc string) string {
	return fmt.Sprintf(`// LeetCode #%s — %s [%s]
// Run: node solution.js  |  Submit: lc submit %s

%s

// --- SUBMIT START ---
%s
// --- SUBMIT END ---

// ── Test ──────────────────────────────────────────────────────
(function() {
    // TODO: add test cases
    // console.log(twoSum([2,7,11,15], 9));
    console.log('Done.');
})();
`, q.QuestionFrontendId, q.Title, q.Difficulty, q.QuestionFrontendId, tc, code)
}

// ─── Rust ─────────────────────────────────────────────────────────────────────

func wrapRust(q *api.Question, code, tc string) string {
	return fmt.Sprintf(`// LeetCode #%s — %s [%s]
// Run: rustc solution.rs && ./solution  |  Submit: lc submit %s

%s

// --- SUBMIT START ---
%s
// --- SUBMIT END ---

fn main() {
    // TODO: add test cases
    // let result = Solution::method_name(arg);
    // assert_eq!(result, expected);
    println!("Done.");
}
`, q.QuestionFrontendId, q.Title, q.Difficulty, q.QuestionFrontendId, tc, code)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func formatTestCases(testcases, lang string) string {
	if testcases == "" {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(testcases), "\n")
	prefix := "//"
	if lang == "python3" || lang == "python" {
		prefix = "#"
	}
	var out []string
	out = append(out, prefix+" ── Example test cases ───────────────────────────")
	for _, l := range lines {
		out = append(out, prefix+"  "+l)
	}
	return strings.Join(out, "\n")
}
