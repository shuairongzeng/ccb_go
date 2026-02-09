package protocol

// ProviderDaemonSpec defines daemon-side configuration for a provider.
type ProviderDaemonSpec struct {
	DaemonKey      string
	ProtocolPrefix string
	StateFileName  string
	LogFileName    string
	IdleTimeoutEnv string
	LockName       string
}

// ProviderClientSpec defines client-side configuration for a provider.
type ProviderClientSpec struct {
	ProtocolPrefix      string
	EnabledEnv          string
	AutostartEnvPrimary string
	AutostartEnvLegacy  string
	StateFileEnv        string
	SessionFilename     string
	DaemonBinName       string
	DaemonModule        string
}

// Provider daemon specs
var (
	CaskdSpec = ProviderDaemonSpec{
		DaemonKey:      "caskd",
		ProtocolPrefix: "cask",
		StateFileName:  "caskd.json",
		LogFileName:    "caskd.log",
		IdleTimeoutEnv: "CCB_CASKD_IDLE_TIMEOUT_S",
		LockName:       "caskd",
	}

	GaskdSpec = ProviderDaemonSpec{
		DaemonKey:      "gaskd",
		ProtocolPrefix: "gask",
		StateFileName:  "gaskd.json",
		LogFileName:    "gaskd.log",
		IdleTimeoutEnv: "CCB_GASKD_IDLE_TIMEOUT_S",
		LockName:       "gaskd",
	}

	OaskdSpec = ProviderDaemonSpec{
		DaemonKey:      "oaskd",
		ProtocolPrefix: "oask",
		StateFileName:  "oaskd.json",
		LogFileName:    "oaskd.log",
		IdleTimeoutEnv: "CCB_OASKD_IDLE_TIMEOUT_S",
		LockName:       "oaskd",
	}

	LaskdSpec = ProviderDaemonSpec{
		DaemonKey:      "laskd",
		ProtocolPrefix: "lask",
		StateFileName:  "laskd.json",
		LogFileName:    "laskd.log",
		IdleTimeoutEnv: "CCB_LASKD_IDLE_TIMEOUT_S",
		LockName:       "laskd",
	}

	DaskdSpec = ProviderDaemonSpec{
		DaemonKey:      "daskd",
		ProtocolPrefix: "dask",
		StateFileName:  "daskd.json",
		LogFileName:    "daskd.log",
		IdleTimeoutEnv: "CCB_DASKD_IDLE_TIMEOUT_S",
		LockName:       "daskd",
	}
)

// Provider client specs
var (
	CaskClientSpec = ProviderClientSpec{
		ProtocolPrefix:      "cask",
		EnabledEnv:          "CCB_CASKD",
		AutostartEnvPrimary: "CCB_CASKD_AUTOSTART",
		AutostartEnvLegacy:  "CCB_AUTO_CASKD",
		StateFileEnv:        "CCB_CASKD_STATE_FILE",
		SessionFilename:     ".codex-session",
		DaemonBinName:       "askd",
		DaemonModule:        "askd.daemon",
	}

	GaskClientSpec = ProviderClientSpec{
		ProtocolPrefix:      "gask",
		EnabledEnv:          "CCB_GASKD",
		AutostartEnvPrimary: "CCB_GASKD_AUTOSTART",
		AutostartEnvLegacy:  "CCB_AUTO_GASKD",
		StateFileEnv:        "CCB_GASKD_STATE_FILE",
		SessionFilename:     ".gemini-session",
		DaemonBinName:       "askd",
		DaemonModule:        "askd.daemon",
	}

	OaskClientSpec = ProviderClientSpec{
		ProtocolPrefix:      "oask",
		EnabledEnv:          "CCB_OASKD",
		AutostartEnvPrimary: "CCB_OASKD_AUTOSTART",
		AutostartEnvLegacy:  "CCB_AUTO_OASKD",
		StateFileEnv:        "CCB_OASKD_STATE_FILE",
		SessionFilename:     ".opencode-session",
		DaemonBinName:       "askd",
		DaemonModule:        "askd.daemon",
	}

	LaskClientSpec = ProviderClientSpec{
		ProtocolPrefix:      "lask",
		EnabledEnv:          "CCB_LASKD",
		AutostartEnvPrimary: "CCB_LASKD_AUTOSTART",
		AutostartEnvLegacy:  "CCB_AUTO_LASKD",
		StateFileEnv:        "CCB_LASKD_STATE_FILE",
		SessionFilename:     ".claude-session",
		DaemonBinName:       "askd",
		DaemonModule:        "askd.daemon",
	}

	DaskClientSpec = ProviderClientSpec{
		ProtocolPrefix:      "dask",
		EnabledEnv:          "CCB_DASKD",
		AutostartEnvPrimary: "CCB_DASKD_AUTOSTART",
		AutostartEnvLegacy:  "CCB_AUTO_DASKD",
		StateFileEnv:        "CCB_DASKD_STATE_FILE",
		SessionFilename:     ".droid-session",
		DaemonBinName:       "askd",
		DaemonModule:        "askd.daemon",
	}
)

// AllDaemonSpecs returns all provider daemon specs.
func AllDaemonSpecs() []ProviderDaemonSpec {
	return []ProviderDaemonSpec{CaskdSpec, GaskdSpec, OaskdSpec, LaskdSpec, DaskdSpec}
}

// AllClientSpecs returns all provider client specs.
func AllClientSpecs() []ProviderClientSpec {
	return []ProviderClientSpec{CaskClientSpec, GaskClientSpec, OaskClientSpec, LaskClientSpec, DaskClientSpec}
}

// DaemonSpecByKey returns the daemon spec for a given key (e.g., "caskd").
func DaemonSpecByKey(key string) *ProviderDaemonSpec {
	for _, s := range AllDaemonSpecs() {
		if s.DaemonKey == key {
			return &s
		}
	}
	return nil
}

// ClientSpecByPrefix returns the client spec for a given protocol prefix (e.g., "cask").
func ClientSpecByPrefix(prefix string) *ProviderClientSpec {
	for _, s := range AllClientSpecs() {
		if s.ProtocolPrefix == prefix {
			return &s
		}
	}
	return nil
}

// ProviderNameMap maps user-facing provider names to protocol prefixes.
var ProviderNameMap = map[string]string{
	"codex":    "cask",
	"gemini":   "gask",
	"opencode": "oask",
	"claude":   "lask",
	"droid":    "dask",
}

// PrefixToProviderName maps protocol prefixes to user-facing provider names.
var PrefixToProviderName = map[string]string{
	"cask": "codex",
	"gask": "gemini",
	"oask": "opencode",
	"lask": "claude",
	"dask": "droid",
}
