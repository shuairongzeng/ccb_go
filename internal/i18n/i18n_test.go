package i18n

import (
	"os"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	// Clear all lang env vars
	for _, v := range []string{"CCB_LANG", "LANG", "LC_ALL", "LC_MESSAGES", "LANGUAGE"} {
		os.Unsetenv(v)
	}

	// Default should be English
	if got := DetectLanguage(); got != LangEN {
		t.Errorf("DetectLanguage() default = %q, want %q", got, LangEN)
	}

	// Chinese
	os.Setenv("CCB_LANG", "zh_CN.UTF-8")
	if got := DetectLanguage(); got != LangZH {
		t.Errorf("DetectLanguage() zh = %q, want %q", got, LangZH)
	}
	os.Unsetenv("CCB_LANG")

	// Japanese via LANG
	os.Setenv("LANG", "ja_JP.UTF-8")
	if got := DetectLanguage(); got != LangJA {
		t.Errorf("DetectLanguage() ja = %q, want %q", got, LangJA)
	}
	os.Unsetenv("LANG")
}

func TestGet(t *testing.T) {
	os.Unsetenv("CCB_LANG")
	os.Unsetenv("LANG")
	os.Unsetenv("LC_ALL")

	msgs := Get()
	if msgs == nil {
		t.Fatal("Get() returned nil")
	}
	if msgs.ErrTimeout == "" {
		t.Error("ErrTimeout is empty")
	}
}

func TestGetLang(t *testing.T) {
	en := GetLang(LangEN)
	zh := GetLang(LangZH)
	ja := GetLang(LangJA)

	if en.ErrTimeout == zh.ErrTimeout {
		t.Error("EN and ZH should have different ErrTimeout")
	}
	if en.ErrTimeout == ja.ErrTimeout {
		t.Error("EN and JA should have different ErrTimeout")
	}

	// Unknown language falls back to English
	unknown := GetLang("xx")
	if unknown.ErrTimeout != en.ErrTimeout {
		t.Error("Unknown language should fall back to English")
	}
}

func TestAllMessageKeysPopulated(t *testing.T) {
	for _, lang := range []string{LangEN, LangZH, LangJA} {
		msgs := GetLang(lang)

		checks := map[string]string{
			"ErrTimeout":       msgs.ErrTimeout,
			"ErrNoReply":       msgs.ErrNoReply,
			"ErrDaemonDown":    msgs.ErrDaemonDown,
			"ErrSessionNotSet": msgs.ErrSessionNotSet,
			"ErrLockFailed":    msgs.ErrLockFailed,
			"ErrPaneDead":      msgs.ErrPaneDead,
			"ErrSendFailed":    msgs.ErrSendFailed,
			"ErrNoBackend":     msgs.ErrNoBackend,
			"ErrNoSession":     msgs.ErrNoSession,
			"ErrInvalidToken":  msgs.ErrInvalidToken,
			"ErrUnknownMethod": msgs.ErrUnknownMethod,
			"DaemonStarting":   msgs.DaemonStarting,
			"DaemonStarted":    msgs.DaemonStarted,
			"DaemonStopping":   msgs.DaemonStopping,
			"DaemonStopped":    msgs.DaemonStopped,
			"DaemonAlready":    msgs.DaemonAlready,
			"DaemonNotFound":   msgs.DaemonNotFound,
			"ProviderPinging":  msgs.ProviderPinging,
			"ProviderOnline":   msgs.ProviderOnline,
			"ProviderOffline":  msgs.ProviderOffline,
			"AskSending":       msgs.AskSending,
			"AskWaiting":       msgs.AskWaiting,
			"AskReceived":      msgs.AskReceived,
			"TermDetecting":    msgs.TermDetecting,
			"SessionResolving": msgs.SessionResolving,
			"SessionFound":     msgs.SessionFound,
			"SessionNotFound":  msgs.SessionNotFound,
			"PaneCreating":     msgs.PaneCreating,
			"PaneCreated":      msgs.PaneCreated,
			"CommAnchorFound":  msgs.CommAnchorFound,
			"CommDoneFound":    msgs.CommDoneFound,
			"DebugLogPath":     msgs.DebugLogPath,
			"DebugReqID":       msgs.DebugReqID,
		}

		for key, val := range checks {
			if val == "" {
				t.Errorf("lang=%s, key=%s is empty", lang, key)
			}
		}
	}
}
