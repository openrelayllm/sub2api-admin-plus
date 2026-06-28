package purity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPurityArchitecture_ServiceFileStaysAsThinRouter(t *testing.T) {
	contentBytes, err := os.ReadFile("service.go")
	require.NoError(t, err)
	content := string(contentBytes)

	require.LessOrEqual(t, strings.Count(content, "\n")+1, 180)
	forbidden := []string{
		"probeResponses",
		"probeClaude",
		"runTokenAudit",
		"runClaudeTokenAudit",
		"wrapperFingerprintSignals",
		"buildResponses",
		"buildClaude",
		"responsesToolProbePayload",
		"claudeMessagesProbePayload",
		"gjson",
	}
	for _, token := range forbidden {
		require.NotContains(t, content, token)
	}
	require.Contains(t, content, "runOpenAICheck")
	require.Contains(t, content, "runClaudeCheck")
}

func TestPurityArchitecture_MainstreamChannelsHaveOwnDetectorFolder(t *testing.T) {
	requiredChannels := []string{
		"openai",
		"claude",
		"gemini",
		"antigravity",
		"bedrock",
		"cliproxyapi",
		"newapi",
		"sub2api",
		"qwen",
		"glm",
		"doubao",
		"minimax",
		"hunyuan",
		"kimi",
		"mimo",
		"xai",
		"deepseek",
	}

	for _, channel := range requiredChannels {
		t.Run(channel, func(t *testing.T) {
			info, err := os.Stat(filepath.Join("channels", channel))
			require.NoError(t, err)
			require.True(t, info.IsDir())

			detectorPath := filepath.Join("channels", channel, "detector.go")
			detectorBytes, err := os.ReadFile(detectorPath)
			require.NoError(t, err)
			require.Contains(t, string(detectorBytes), "func (Detector) Detect")
		})
	}
}
