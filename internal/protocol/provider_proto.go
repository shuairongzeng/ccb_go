package protocol

import (
	"fmt"
	"strings"
)

// ProviderProto defines provider-specific prompt wrapping and response extraction.
type ProviderProto struct {
	Name          string
	WrapPrompt    func(message string, reqID string) string
	ExtractReply  func(text string, reqID string) string
	IsDone        func(text string, reqID string) bool
}

// --- Codex (cask) protocol ---

func wrapCodexPrompt(message string, reqID string) string {
	return WrapCodexPrompt(message, reqID)
}

func extractCodexReply(text string, reqID string) string {
	return StripDoneText(text, reqID)
}

func isCodexDone(text string, reqID string) bool {
	return IsDoneText(text, reqID)
}

// --- Gemini (gask) protocol ---

func wrapGeminiPrompt(message string, reqID string) string {
	message = strings.TrimRight(message, "\n\r\t ")
	return fmt.Sprintf(
		"%s %s\n\n%s\n\nIMPORTANT:\n- Reply normally.\n- Reply normally, in English.\n- End your reply with this exact final line (verbatim, on its own line):\n%s %s\n",
		ReqIDPrefix, reqID,
		message,
		DonePrefix, reqID,
	)
}

func extractGeminiReply(text string, reqID string) string {
	return StripDoneText(text, reqID)
}

func isGeminiDone(text string, reqID string) bool {
	return IsDoneText(text, reqID)
}

// --- OpenCode (oask) protocol ---

func wrapOpenCodePrompt(message string, reqID string) string {
	message = strings.TrimRight(message, "\n\r\t ")
	return fmt.Sprintf(
		"%s %s\n\n%s\n\nIMPORTANT:\n- Reply normally.\n- Reply normally, in English.\n- End your reply with this exact final line (verbatim, on its own line):\n%s %s\n",
		ReqIDPrefix, reqID,
		message,
		DonePrefix, reqID,
	)
}

func extractOpenCodeReply(text string, reqID string) string {
	return StripDoneText(text, reqID)
}

func isOpenCodeDone(text string, reqID string) bool {
	return IsDoneText(text, reqID)
}

// --- Claude (lask) protocol ---

func wrapClaudePrompt(message string, reqID string) string {
	message = strings.TrimRight(message, "\n\r\t ")
	return fmt.Sprintf(
		"%s %s\n\n%s\n\nIMPORTANT:\n- Reply normally.\n- Reply normally, in English.\n- End your reply with this exact final line (verbatim, on its own line):\n%s %s\n",
		ReqIDPrefix, reqID,
		message,
		DonePrefix, reqID,
	)
}

func extractClaudeReply(text string, reqID string) string {
	return StripDoneText(text, reqID)
}

func isClaudeDone(text string, reqID string) bool {
	return IsDoneText(text, reqID)
}

// --- Droid (dask) protocol ---

func wrapDroidPrompt(message string, reqID string) string {
	message = strings.TrimRight(message, "\n\r\t ")
	return fmt.Sprintf(
		"%s %s\n\n%s\n\nIMPORTANT:\n- Reply normally.\n- Reply normally, in English.\n- End your reply with this exact final line (verbatim, on its own line):\n%s %s\n",
		ReqIDPrefix, reqID,
		message,
		DonePrefix, reqID,
	)
}

func extractDroidReply(text string, reqID string) string {
	return StripDoneText(text, reqID)
}

func isDroidDone(text string, reqID string) bool {
	return IsDoneText(text, reqID)
}

// --- Provider protocol registry ---

var (
	CodexProto = &ProviderProto{
		Name:         "codex",
		WrapPrompt:   wrapCodexPrompt,
		ExtractReply: extractCodexReply,
		IsDone:       isCodexDone,
	}

	GeminiProto = &ProviderProto{
		Name:         "gemini",
		WrapPrompt:   wrapGeminiPrompt,
		ExtractReply: extractGeminiReply,
		IsDone:       isGeminiDone,
	}

	OpenCodeProto = &ProviderProto{
		Name:         "opencode",
		WrapPrompt:   wrapOpenCodePrompt,
		ExtractReply: extractOpenCodeReply,
		IsDone:       isOpenCodeDone,
	}

	ClaudeProto = &ProviderProto{
		Name:         "claude",
		WrapPrompt:   wrapClaudePrompt,
		ExtractReply: extractClaudeReply,
		IsDone:       isClaudeDone,
	}

	DroidProto = &ProviderProto{
		Name:         "droid",
		WrapPrompt:   wrapDroidPrompt,
		ExtractReply: extractDroidReply,
		IsDone:       isDroidDone,
	}
)

// ProtoByName returns the ProviderProto for a given provider name.
func ProtoByName(name string) *ProviderProto {
	switch strings.ToLower(name) {
	case "codex", "cask":
		return CodexProto
	case "gemini", "gask":
		return GeminiProto
	case "opencode", "oask":
		return OpenCodeProto
	case "claude", "lask":
		return ClaudeProto
	case "droid", "dask":
		return DroidProto
	}
	return nil
}
