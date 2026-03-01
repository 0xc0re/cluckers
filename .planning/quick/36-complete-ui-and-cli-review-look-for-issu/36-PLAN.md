---
phase: 36-ui-cli-review
plan: 36
type: execute
wave: 1
depends_on: []
files_modified:
  - cmd/cluckers/main.go
  - internal/cli/launch.go
  - internal/cli/root.go
  - internal/launch/pipeline.go
  - internal/launch/prep.go
  - internal/ui/errors.go
  - internal/ui/prompt.go
autonomous: true
requirements: []
must_haves:
  truths:
    - "CLI errors are shown exactly once, with suggestions visible"
    - "Empty username or password rejected immediately with clear message"
    - "Command descriptions accurately reflect current platform support"
    - "No dead code referencing removed Wine-direct launch path"
  artifacts:
    - path: "cmd/cluckers/main.go"
      provides: "Formatted error output using ui.FormatError"
    - path: "internal/ui/prompt.go"
      provides: "Empty input validation for username and password"
    - path: "internal/ui/errors.go"
      provides: "Clean error utilities without orphaned Wine functions"
  key_links:
    - from: "cmd/cluckers/main.go"
      to: "internal/ui/errors.go"
      via: "ui.FormatError for error display"
      pattern: "ui\\.FormatError"
    - from: "internal/launch/pipeline.go"
      to: "cmd/cluckers/main.go"
      via: "error return without double-printing"
      pattern: "return err"
---

<objective>
Fix UI and CLI issues discovered during code review: double error printing,
lost UserError suggestions, empty input acceptance, stale command descriptions,
and dead code from the Wine-to-Proton migration.

Purpose: Users currently see errors printed twice on launch failures, never see
helpful suggestion text on non-pipeline commands, and can submit empty credentials
that produce confusing gateway errors.

Output: Cleaner error UX, accurate help text, no dead code.
</objective>

<execution_context>
@/home/cstory/.claude/get-shit-done/workflows/execute-plan.md
@/home/cstory/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@CLAUDE.md
@cmd/cluckers/main.go
@internal/cli/launch.go
@internal/cli/root.go
@internal/launch/pipeline.go
@internal/launch/prep.go
@internal/ui/errors.go
@internal/ui/prompt.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Fix error display -- eliminate double printing and surface suggestions</name>
  <files>
    cmd/cluckers/main.go
    internal/launch/pipeline.go
    internal/launch/prep.go
  </files>
  <action>
    **Problem 1: Double error printing.** In `pipeline.go:RunWithReporter()` line 86,
    `ui.Error(ui.FormatError(err, cfg.Verbose))` prints the formatted error to stdout.
    Then the error propagates back through launch.go to main.go, where
    `fmt.Fprintln(os.Stderr, err)` prints the bare Message again. Users see the error
    twice. Same issue in `prep.go:RunPrepWithReporter()` line 47.

    Fix: Remove the `ui.Error(ui.FormatError(...))` calls from both `RunWithReporter()`
    and `RunPrepWithReporter()`. The pipeline already calls `reporter.StepFailed()` which
    prints a red X with the step name. The error formatting should happen at the top level
    in main.go, not inside the pipeline.

    In `pipeline.go`, remove line 86: `ui.Error(ui.FormatError(err, cfg.Verbose))`
    In `prep.go`, remove line 47: `ui.Error(ui.FormatError(err, cfg.Verbose))`

    **Problem 2: UserError suggestions lost at top level.** In `main.go`, the error
    handler is `fmt.Fprintln(os.Stderr, err)` which only calls `err.Error()` (the
    Message field). Suggestion and Detail are never shown for any command.

    Fix: In `main.go`, replace the error handler:

    ```go
    if err := cli.Execute(); err != nil {
        // Use FormatError to show suggestions and details for UserErrors.
        // Verbose is not available here (config loads inside Cobra), so
        // check if -v was passed by looking at os.Args.
        verbose := false
        for _, arg := range os.Args {
            if arg == "-v" || arg == "--verbose" {
                verbose = true
                break
            }
        }
        fmt.Fprintln(os.Stderr, ui.FormatError(err, verbose))
        os.Exit(1)
    }
    ```

    This requires importing `"github.com/0xc0re/cluckers/internal/ui"` in main.go.

    Note: `ui.FormatError` already handles non-UserError cases (returns `err.Error()`),
    so this is safe for all error types.
  </action>
  <verify>
    go build ./cmd/cluckers && go vet ./... && GOOS=windows go vet ./...
  </verify>
  <done>
    Pipeline errors shown exactly once with suggestions visible. Non-pipeline command
    errors (login, update, logout) also show suggestions when present.
  </done>
</task>

<task type="auto">
  <name>Task 2: Validate inputs, fix descriptions, remove dead code</name>
  <files>
    internal/ui/prompt.go
    internal/ui/errors.go
    internal/cli/launch.go
    internal/cli/root.go
  </files>
  <action>
    **Fix 1: Empty username/password validation.** In `prompt.go`, after reading
    input in both `PromptUsername()` and `PromptPassword()`, validate that the
    result is not empty. If empty, return a clear error:

    In `PromptUsername()`, after `strings.TrimSpace(line)`, add:
    ```go
    username := strings.TrimSpace(line)
    if username == "" {
        return "", fmt.Errorf("username cannot be empty")
    }
    return username, nil
    ```

    In `PromptPassword()`, after `string(pw)`, add:
    ```go
    password := string(pw)
    if password == "" {
        return "", fmt.Errorf("password cannot be empty")
    }
    return password, nil
    ```

    **Fix 2: Stale command descriptions.** In `root.go`:
    - Change `Short` from `"Project Crown Linux Launcher"` to `"Project Crown Launcher"`
      (it supports Windows too).

    In `launch.go`:
    - Change `Long` from `"...then launches Realm Royale under Wine."` to
      `"...then launches Realm Royale."` (Wine is an implementation detail;
      on Windows there is no Wine; on Linux it uses Proton-GE not Wine directly).

    **Fix 3: Remove dead code.** In `launch.go`:
    - Remove the `isUserError` function (lines 27-34). It is called on line 18 but
      the result is unused -- both branches of the if/else return `err` unchanged.
    - Simplify the RunE to just:
      ```go
      RunE: func(cmd *cobra.Command, args []string) error {
          return launch.Run(cmd.Context(), Cfg)
      },
      ```
    - Remove the unused `"github.com/0xc0re/cluckers/internal/ui"` import from launch.go.

    In `errors.go`:
    - Remove `WineNotFoundError()` function (never called anywhere in the codebase).
    - Remove `detectDistro()` function (only called by WineNotFoundError).
    - Remove `wineInstallInstructions()` function (only called by WineNotFoundError).
    - Remove the unused `"bufio"` and `"os"` imports (check if FormatError still needs
      them -- it uses `strings` and `errors` only, so `bufio`, `fmt`, and `os` can go).
    - Keep `"fmt"` only if still used elsewhere in the file. Check: `FormatError` uses
      `strings.Builder`, not `fmt`. So `"fmt"` can also be removed. The remaining
      imports should be just `"errors"` and `"strings"`.
  </action>
  <verify>
    go build ./cmd/cluckers && go test ./... && go vet ./... && GOOS=windows go vet ./...
  </verify>
  <done>
    Empty username/password rejected with "cannot be empty" error before any API call.
    Root command Short says "Project Crown Launcher" (not "Linux Launcher").
    Launch command Long does not mention Wine.
    No dead code: isUserError, WineNotFoundError, detectDistro, wineInstallInstructions
    all removed. All tests pass, vet clean on both platforms.
  </done>
</task>

</tasks>

<verification>
1. `go build ./cmd/cluckers` succeeds
2. `go test ./...` all pass
3. `go vet ./...` clean
4. `GOOS=windows go vet ./...` clean
5. `grep -r "WineNotFoundError\|isUserError\|detectDistro\|wineInstallInstructions" internal/ cmd/` returns no results
6. `grep "under Wine" internal/cli/launch.go` returns no results
7. `grep "Linux Launcher" internal/cli/root.go` returns no results
</verification>

<success_criteria>
- Errors printed exactly once to terminal (no double-printing from pipeline + main)
- UserError suggestions visible on all commands (login, launch, update, logout, steam)
- Empty username or password rejected immediately with clear message
- All command descriptions accurate for cross-platform (no Wine, no Linux-only)
- Zero dead code from Wine migration
- All tests pass, vet clean on both Linux and Windows targets
</success_criteria>

<output>
After completion, create `.planning/quick/36-complete-ui-and-cli-review-look-for-issu/36-SUMMARY.md`
</output>
