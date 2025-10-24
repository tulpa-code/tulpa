package prompt

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/tulpa-code/tulpa/internal/config"
	"github.com/tulpa-code/tulpa/internal/llm/tools"
)

func CoderPrompt(_ string, contextFiles ...string) string {
	cfg := config.Get()
	var cwd string
	if cfg == nil {
		cwd = "."
	} else {
		cwd = cfg.WorkingDir()
	}
	basePrompt := string(defaultCoderPrompt)
	contextContent := getContextFromPaths(cwd, contextFiles)
	if contextContent != "" {
		return fmt.Sprintf("%s\n\n# Project-Specific Context\n Make sure to follow the instructions in the context below\n%s", basePrompt, contextContent)
	}
	return basePrompt
}

//go:embed anthropic.md
var defaultCoderPrompt []byte

func getEnvironmentInfo() string {
	cfg := config.Get()
	if cfg == nil {
		return "Environment information unavailable - no config loaded"
	}
	cwd := cfg.WorkingDir()
	isGit := isGitRepo(cwd)
	platform := runtime.GOOS
	date := time.Now().Format("1/2/2006")
	output, _, _ := tools.ListDirectoryTree(cwd, tools.LSParams{})
	return fmt.Sprintf(`Here is useful information about the environment you are running in:
<env>
Working directory: %s
Is directory a git repo: %s
Platform: %s
Today's date: %s
</env>
<project>
%s
</project>
		`, cwd, boolToYesNo(isGit), platform, date, output)
}

func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func lspInformation() string {
	cfg := config.Get()
	if cfg == nil {
		return ""
	}
	hasLSP := false
	for _, v := range cfg.LSP {
		if !v.Disabled {
			hasLSP = true
			break
		}
	}
	if !hasLSP {
		return ""
	}
	return `# LSP Information
Tools that support it will also include useful diagnostics such as linting and typechecking.
- These diagnostics will be automatically enabled when you run the tool, and will be displayed in the output at the bottom within the <file_diagnostics></file_diagnostics> and <project_diagnostics></project_diagnostics> tags.
- Take necessary actions to fix the issues.
- You should ignore diagnostics of files that you did not change or are not related or caused by your changes unless the user explicitly asks you to fix them.
`
}

func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
