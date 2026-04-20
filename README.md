# leetcode-cli

A fast, beautiful LeetCode CLI written in Go. No browser needed — solve problems, run tests, and submit solutions right from your terminal.

```
  _              _    ____          _
 | |    ___  ___| |_ / ___|___   __| | ___
 | |   / _ \/ _ \ __| |   / _ \ / _  |/ _ \
 | |__|  __/  __/ |_| |__| (_) | (_| |  __/
 |_____\___|\___|\__|\____\___/ \__,_|\___|  CLI
```

---

## Features

- **View any question** by number with syntax-highlighted content
- **Question of the Day** — see today's daily challenge instantly
- **Generate starter code** in any language, saved to a local file
- **Run tests** against example test cases with pass/fail per case
- **Submit solutions** and see accepted / wrong answer / runtime error
- **Hints** on demand
- **Multi-language** — Go, Python, JavaScript, Java, C++, Rust, and more
- Direct GitHub release binaries for Linux, macOS, and Windows
- Zero dependencies — single static binary

---

## Installation

### Option 1: Download a prebuilt binary (recommended)

Grab the latest release assets directly from GitHub:

- Linux amd64: https://github.com/Akshdhiwar/leetcode-terminal/releases/download/latest/linux-amd64/lc
- Linux arm64: https://github.com/Akshdhiwar/leetcode-terminal/releases/download/latest/linux-arm64/lc
- macOS amd64: https://github.com/Akshdhiwar/leetcode-terminal/releases/download/latest/darwin-amd64/lc
- macOS arm64: https://github.com/Akshdhiwar/leetcode-terminal/releases/download/latest/darwin-arm64/lc
- Windows amd64: https://github.com/Akshdhiwar/leetcode-terminal/releases/download/latest/windows-amd64/lc.exe
- Checksums: https://github.com/Akshdhiwar/leetcode-terminal/releases/download/latest/checksums.txt

Install and name the binary `lc` so you can run commands from any terminal:

```bash
# Linux amd64
curl -L -o lc https://github.com/Akshdhiwar/leetcode-terminal/releases/download/latest/linux-amd64/lc
chmod +x lc
sudo mv lc /usr/local/bin/lc

# macOS arm64
curl -L -o lc https://github.com/Akshdhiwar/leetcode-terminal/releases/download/latest/darwin-arm64/lc
chmod +x lc
sudo mv lc /usr/local/bin/lc

# Windows amd64
curl -L -o lc.exe https://github.com/Akshdhiwar/leetcode-terminal/releases/download/latest/windows-amd64/lc.exe
# Move lc.exe to a folder in your PATH, for example C:\Windows\System32 or a custom bin folder.
```

If the binary is not already in your PATH, add its folder to your environment variables:

- Linux/macOS:
  ```bash
  export PATH="$HOME/bin:$PATH"
  ```
  Add that line to `~/.bashrc`, `~/.zshrc`, or your shell profile.

- Windows:
  1. Open System Properties → Advanced → Environment Variables.
  2. Edit `Path` under your user variables.
  3. Add the folder containing `lc.exe`.
  4. Restart your terminal.

Verify installation:

```bash
lc --help
which lc
```

After install, run commands like:

```bash
lc show 33
lc test 33
lc submit 33
```

### Option 2: Build from source

```bash
git clone https://github.com/Akshdhiwar/leetcode-terminal.git
cd leetcode-terminal
go build -o lc .
```

---

## Authentication

Test and submit require your LeetCode session cookie.

```bash
lc auth
```

Follow the prompt:
1. Log in at https://leetcode.com in your browser
2. Open **DevTools → Application → Cookies → leetcode.com**
3. Copy `LEETCODE_SESSION` and `csrftoken` values
4. Paste them when prompted

Credentials are saved to `~/.leetcode-cli/config.json` (chmod 600).

> **Viewing questions does NOT require authentication.**

---

## Usage

```
lc [command] [flags]
```

### Commands

| Command | Description |
|---|---|
| `lc auth` | Save LeetCode session for test/submit |
| `lc today` | View the Question of the Day |
| `lc show <number>` | View a question by number |
| `lc code <number>` | Generate a starter code file |
| `lc test <number>` | Test solution against example cases |
| `lc submit <number>` | Submit solution to LeetCode |
| `lc hints <number>` | Show hints for a question |
| `lc lang [language]` | View or set default language |

---

## Workflow Example

```bash
# 1. See today's challenge
lc today

# 2. View question #1 (Two Sum)
lc show 1

# 3. Generate a Go solution file
lc code 1
# → Creates ~/.leetcode-cli/solutions/1-two-sum.go

# 4. Edit the file with your solution
vim ~/.leetcode-cli/solutions/1-two-sum.go

# 5. Test against example cases
lc test 1
# ✔ Test case 1   PASSED
# ✘ Test case 2   FAILED
#   Your output:  [1,0]
#   Expected:     [0,1]

# 6. Fix and test again
lc test 1

# 7. Submit when all pass
lc submit 1
#   ✔ ACCEPTED
#   Runtime:  2 ms    Beats 97.3% of submissions
#   Memory:   3.2 MB  Beats 88.1%

# Use a custom file
lc test 1 ./mysolution.go
lc submit 1 ./mysolution.go

# Test with custom input
lc test 1 --input "[3,2,4]\n7"
```

---

## Language Support

```bash
lc lang              # show current language
lc lang python3      # switch to Python 3
lc lang golang       # switch to Go (default)
lc lang javascript   # switch to JavaScript
lc lang java         # switch to Java
lc lang cpp          # switch to C++
lc lang rust         # switch to Rust
```

Language is saved globally — no need to specify per command.

---

## Flags

| Flag | Works with | Description |
|---|---|---|
| `--hints` | `show`, `today` | Show hints inline |
| `--code`, `-c` | `show`, `today` | Print starter code |
| `--input "..."` | `test` | Custom test input |

---

## File Locations

| Path | Contents |
|---|---|
| `~/.leetcode-cli/config.json` | Auth + language preferences |
| `~/.leetcode-cli/solutions/` | Generated solution files |

---

## Test Result Output

```
──────────────────────────────────────────────────────────────
✔ Test case 1    PASSED
✔ Test case 2    PASSED
✘ Test case 3    FAILED

  Your output:   [0]
  Expected:      [0,1]
──────────────────────────────────────────────────────────────
```

## Submit Result Output

```
──────────────────────────────────────────────────────────────
  ✔ ACCEPTED

  Runtime:       2 ms
  Beats:         97.30% of submissions
  Memory:        3.2 MB
  Beats:         88.10% in memory
  Test cases:    58/58
──────────────────────────────────────────────────────────────
```

---

## Building for All Platforms

```bash
make build-all
```

Or manually:

```bash
GOOS=linux   GOARCH=amd64 go build -o dist/lc-linux-amd64 .
GOOS=darwin  GOARCH=arm64 go build -o dist/lc-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o dist/lc-windows-amd64.exe .
```

---

## License

MIT
