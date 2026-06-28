package purity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeProvider_ProtocolAliases(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty_defaults_openai", in: "", want: ProviderOpenAI},
		{name: "openai", in: "openai", want: ProviderOpenAI},
		{name: "openai_compatible_dash", in: "OpenAI-Compatible", want: ProviderOpenAI},
		{name: "openai_compatible_space", in: "openai compatible", want: ProviderOpenAI},
		{name: "anthropic", in: "anthropic", want: ProviderAnthropic},
		{name: "claude", in: "Claude", want: ProviderAnthropic},
		{name: "claude_compatible_dash", in: "claude-compatible", want: ProviderAnthropic},
		{name: "anthropic_compatible_space", in: "Anthropic Compatible", want: ProviderAnthropic},
		{name: "gemini", in: "gemini", want: ProviderGemini},
		{name: "gemini_compatible_dash", in: "Gemini-Compatible", want: ProviderGemini},
		{name: "google_ai_studio_space", in: "Google AI Studio", want: ProviderGemini},
		{name: "qwen_remains_unsupported", in: "qwen", want: "qwen"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, normalizeProvider(tc.in))
		})
	}
}
