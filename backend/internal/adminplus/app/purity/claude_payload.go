package purity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

func claudeToolProbePayload(model string, probeCtx claudeProbeContext) []byte {
	body, _ := json.Marshal(map[string]any{
		"model":       firstNonEmptyString(model, defaultClaudeModel),
		"max_tokens":  512,
		"temperature": 1,
		"system":      claudeSystemBlocks("Call the requested tool exactly once."),
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					claudeTextBlock("Call the probe_ping tool with ok=true. Do not answer in text.", true),
				},
			},
		},
		"tools": []map[string]any{
			{
				"name":        "probe_ping",
				"description": "Capability probe. Call to acknowledge.",
				"input_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ok": map[string]any{"type": "boolean"},
					},
					"required": []string{"ok"},
				},
			},
		},
		"tool_choice": map[string]any{"type": "tool", "name": "probe_ping"},
		"metadata":    probeCtx.metadata(),
	})
	return body
}

func claudeStreamProbePayload(model string, probeCtx claudeProbeContext) []byte {
	body, _ := json.Marshal(map[string]any{
		"model":       firstNonEmptyString(model, defaultClaudeModel),
		"max_tokens":  32,
		"temperature": 1,
		"stream":      true,
		"system":      claudeSystemBlocks("Return concise responses."),
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					claudeTextBlock("Return exactly: ok", true),
				},
			},
		},
		"metadata": probeCtx.metadata(),
	})
	return body
}

func claudeMultimodalProbePayload(model string, probeCtx claudeProbeContext) []byte {
	body, _ := json.Marshal(map[string]any{
		"model":       firstNonEmptyString(model, defaultClaudeModel),
		"max_tokens":  32,
		"temperature": 1,
		"system":      claudeSystemBlocks("Return concise responses."),
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": "Read the attached image and return exactly: ok"},
					{
						"type": "image",
						"source": map[string]any{
							"type":       "base64",
							"media_type": "image/png",
							"data":       probePNGBase64,
						},
					},
				},
			},
		},
		"metadata": probeCtx.metadata(),
	})
	return body
}

func claudeAuditProbePayload(model string, round int, auditNonce string, probeCtx claudeProbeContext, history []claudeAuditTurn) []byte {
	system := claudeAuditSystemBlocks(round, auditNonce)
	messages := claudeAuditMessages(round, auditNonce, history)
	body, _ := json.Marshal(map[string]any{
		"model":       firstNonEmptyString(model, defaultClaudeModel),
		"max_tokens":  claudeTokenAuditOutputBudget(round),
		"temperature": 1,
		"system":      system,
		"messages":    messages,
		"metadata":    probeCtx.metadata(),
	})
	return body
}

func claudeInvalidThinkingSignaturePayload(model string, probeCtx claudeProbeContext) []byte {
	body, _ := json.Marshal(map[string]any{
		"model":       firstNonEmptyString(model, defaultClaudeModel),
		"max_tokens":  32,
		"temperature": 1,
		"system":      claudeSystemBlocks("Validate historical thinking signatures."),
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					claudeTextBlock("Return exactly: warmup", false),
				},
			},
			{
				"role": "assistant",
				"content": []map[string]any{
					{"type": "thinking", "thinking": "invalid historical thought", "signature": "not-a-valid-claude-thinking-signature"},
					{"type": "text", "text": "warmup"},
				},
			},
			{
				"role": "user",
				"content": []map[string]any{
					claudeTextBlock("Return exactly: ok", false),
				},
			},
		},
		"metadata": probeCtx.metadata(),
	})
	return body
}

func claudeThinkingBudgetViolationPayload(model string, probeCtx claudeProbeContext) []byte {
	body, _ := json.Marshal(map[string]any{
		"model":       firstNonEmptyString(model, defaultClaudeModel),
		"max_tokens":  64,
		"temperature": 1,
		"thinking": map[string]any{
			"type":          "enabled",
			"budget_tokens": 64,
		},
		"system": claudeSystemBlocks("Validate thinking budget constraints."),
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					claudeTextBlock("Return exactly: ok", false),
				},
			},
		},
		"metadata": probeCtx.metadata(),
	})
	return body
}

func claudeCacheControlOverflowPayload(model string, probeCtx claudeProbeContext) []byte {
	systemBlocks := make([]map[string]any, 0, 6)
	for i := 0; i < 6; i++ {
		systemBlocks = append(systemBlocks, claudeTextBlock(fmt.Sprintf("Cache control overflow probe block %d.", i+1), true))
	}
	body, _ := json.Marshal(map[string]any{
		"model":       firstNonEmptyString(model, defaultClaudeModel),
		"max_tokens":  32,
		"temperature": 1,
		"system":      systemBlocks,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					claudeTextBlock("Return exactly: ok", false),
				},
			},
		},
		"metadata": probeCtx.metadata(),
	})
	return body
}

func claudeAuditSystemBlocks(round int, auditNonce string) []map[string]any {
	round = clampAuditRound(round)
	return []map[string]any{
		claudeTextBlock(claudeBillingAttributionText(fmt.Sprintf("Purity token audit round %02d", round)), false),
		claudeTextBlock(claudeCodeSystemPrompt, true),
		claudeTextBlock(auditStableCacheText(auditNonce), true),
	}
}

func claudeAuditMessages(round int, auditNonce string, history []claudeAuditTurn) []map[string]any {
	round = clampAuditRound(round)
	messages := make([]map[string]any, 0, len(history)*2+1)
	for _, turn := range history {
		userText := strings.TrimSpace(turn.UserText)
		if userText == "" {
			userText = claudeAuditUserText(turn.Round, auditNonce)
		}
		messages = append(messages, map[string]any{
			"role":    "user",
			"content": []map[string]any{claudeTextBlock(userText, false)},
		})
		assistantText := strings.TrimSpace(turn.AssistantText)
		if assistantText == "" {
			continue
		}
		messages = append(messages, map[string]any{
			"role":    "assistant",
			"content": []map[string]any{claudeTextBlock(assistantText, false)},
		})
	}
	messages = append(messages, map[string]any{
		"role":    "user",
		"content": []map[string]any{claudeTextBlock(claudeAuditUserText(round, auditNonce), true)},
	})
	return messages
}

func clampAuditRound(round int) int {
	if round < 1 {
		return 1
	}
	if round > tokenAuditSamples {
		return tokenAuditSamples
	}
	return round
}

func claudeSystemBlocks(firstUserText string) []map[string]any {
	return []map[string]any{
		claudeTextBlock(claudeBillingAttributionText(firstUserText), false),
		claudeTextBlock(claudeCodeSystemPrompt, true),
	}
}

func claudeAuditResponseInstruction(round int) string {
	return strings.Join([]string{
		"Purity token audit response target.",
		claudeTokenAuditRoundInstruction(round),
	}, "\n\n")
}

func claudeTokenAuditOutputBudget(round int) int {
	target := claudeTokenAuditOutputTarget(round)
	if target <= 0 {
		return tokenAuditOutputBudget(round)
	}
	return minInt(1800, target+160)
}

func claudeTokenAuditRoundInstruction(round int) string {
	target := claudeTokenAuditOutputTarget(round)
	return fmt.Sprintf("Round %02d: output exactly %d comma-separated items. Each item must be the single lowercase letter x. Use no spaces, no numbering, no markdown, and no text before or after the list.", clampAuditRound(round), target)
}

func claudeTokenAuditOutputTarget(round int) int {
	targets := []int{152, 162, 770, 336, 616, 668, 1413, 783, 128, 111, 387}
	round = clampAuditRound(round)
	return targets[round-1]
}

func claudeAuditUserText(round int, auditNonce string) string {
	round = clampAuditRound(round)
	return strings.Join([]string{
		fmt.Sprintf("proxyai.best Claude cache audit turn %02d. audit_nonce=%s", round, auditNonce),
		auditRoundCacheText(round),
		claudeAuditResponseInstruction(round),
	}, "\n\n")
}

func claudeTextBlock(text string, cache bool) map[string]any {
	block := map[string]any{"type": "text", "text": text}
	if cache {
		block["cache_control"] = map[string]any{"type": "ephemeral"}
	}
	return block
}

func claudeBillingAttributionText(firstUserText string) string {
	chars := []byte{'0', '0', '0'}
	for idx, pos := range []int{4, 7, 20} {
		if pos < len(firstUserText) {
			chars[idx] = firstUserText[pos]
		}
	}
	sum := sha256.Sum256([]byte(claudeFingerprintSalt + string(chars) + claudeCodeProbeVersion))
	fp := hex.EncodeToString(sum[:])[:3]
	return fmt.Sprintf("x-anthropic-billing-header: cc_version=%s.%s; cc_entrypoint=cli; cch=00000;", claudeCodeProbeVersion, fp)
}
