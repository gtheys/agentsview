package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInferPiSkillNameReadSKILLMD(t *testing.T) {
	tests := []struct {
		name       string
		toolName   string
		inputJSON  string
		sessionCwd string
		want       string
	}{
		{
			name:      "absolute SKILL.md read yields parent dir",
			toolName:  "read",
			inputJSON: `{"path":"/skills/engineering/implement-plan/SKILL.md"}`,
			want:      "implement-plan",
		},
		{
			name:      "file_path key also matches",
			toolName:  "read",
			inputJSON: `{"file_path":"/skills/review/SKILL.md"}`,
			want:      "review",
		},
		{
			name:      "tilde path resolves to parent dir",
			toolName:  "read",
			inputJSON: `{"path":"~/.pi/agent/skills/create-plan/SKILL.md"}`,
			want:      "create-plan",
		},
		{
			name:       "relative path resolves against session cwd",
			toolName:   "read",
			inputJSON:  `{"path":"skills/engineering/create-plan/SKILL.md"}`,
			sessionCwd: "/worktrees/demo",
			want:       "create-plan",
		},
		{
			name:      "plain source file read is not a skill",
			toolName:  "read",
			inputJSON: `{"path":"/repo/auth.go"}`,
			want:      "",
		},
		{
			name:      "glob path is not a skill",
			toolName:  "read",
			inputJSON: `{"path":"/skills/*/SKILL.md"}`,
			want:      "",
		},
		{
			name:      "non-read tools stay unattributed",
			toolName:  "edit",
			inputJSON: `{"path":"/skills/foo/SKILL.md"}`,
			want:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferPiSkillName(tt.toolName, tt.inputJSON, tt.sessionCwd)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInferPiSkillNameReadFrontmatter(t *testing.T) {
	skillDir := filepath.Join(t.TempDir(), "deploy-docs")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	skillFile := filepath.Join(skillDir, "SKILL.md")
	require.NoError(t, os.WriteFile(skillFile, []byte(
		"---\nname: deploy-docs\ndescription: Deployment guides\n---\n\nbody\n",
	), 0o644))

	got := inferPiSkillName(
		"read", `{"path":"/opt/skills/deploy-docs/SKILL.md"}`, "",
	)
	assert.Equal(t, "deploy-docs", got)
}

func TestInferPiSkillNameSkillURI(t *testing.T) {
	tests := []struct {
		name      string
		toolName  string
		inputJSON string
		want      string
	}{
		{
			name:      "read with skill path URI",
			toolName:  "read",
			inputJSON: `{"path":"skill://create-plan/SKILL.md"}`,
			want:      "create-plan",
		},
		{
			name:      "leading slashes are trimmed",
			toolName:  "read",
			inputJSON: `{"path":"skill:///review-pr"}`,
			want:      "review-pr",
		},
		{
			name:      "bare name is attributed",
			toolName:  "read",
			inputJSON: `{"path":"skill://coding-standards"}`,
			want:      "coding-standards",
		},
		{
			name:      "URI in any string field is attributed",
			toolName:  "view",
			inputJSON: `{"url":"skill://deploy-docs"}`,
			want:      "deploy-docs",
		},
		{
			name:      "query suffix is stripped",
			toolName:  "read",
			inputJSON: `{"path":"skill://create-plan?section=1"}`,
			want:      "create-plan",
		},
		{
			name:      "empty name falls through",
			toolName:  "read",
			inputJSON: `{"path":"skill:///"}`,
			want:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferPiSkillName(tt.toolName, tt.inputJSON, "")
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInferPiSkillNameShellCommand(t *testing.T) {
	tests := []struct {
		name       string
		toolName   string
		inputJSON  string
		sessionCwd string
		want       string
	}{
		{
			name:       "cat of a SKILL.md",
			toolName:   "bash",
			inputJSON:  `{"command":"cat skills/review/SKILL.md"}`,
			sessionCwd: "/worktrees/demo",
			want:       "review",
		},
		{
			name:       "grep of a SKILL.md",
			toolName:   "bash",
			inputJSON:  `{"command":"grep -n frontmatter /skills/review/SKILL.md"}`,
			sessionCwd: "",
			want:       "review",
		},
		{
			name:       "sed without -i is a read",
			toolName:   "bash",
			inputJSON:  `{"command":"sed -n 1,10p /skills/review/SKILL.md"}`,
			sessionCwd: "",
			want:       "review",
		},
		{
			name:       "relative path resolves against session cwd",
			toolName:   "bash",
			inputJSON:  `{"command":"cat ./skills/create-plan/SKILL.md"}`,
			sessionCwd: "/worktrees/demo",
			want:       "create-plan",
		},
		{
			name:       "unrelated command is untouched",
			toolName:   "bash",
			inputJSON:  `{"command":"go test ./..."}`,
			sessionCwd: "/worktrees/demo",
			want:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferPiSkillName(tt.toolName, tt.inputJSON, tt.sessionCwd)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPiProviderAttributesSkillNames(t *testing.T) {
	fixturePath := createTestFile(
		t, "pi-session.jsonl",
		loadFixture(t, "pi/session.jsonl"),
	)
	sess, msgs, err := parsePiTestSession(t, fixturePath, "", "local")
	require.NoError(t, err)

	// The fixture header carries cwd /Users/alice/code/my-project and
	// entry-2 holds one read tool call with file_path auth.go. No
	// SKILL.md is involved, so no skill name may be attributed.
	assert.Equal(t, "", msgs[1].ToolCalls[0].SkillName)
	assert.NotEmpty(t, sess.Cwd)
}
