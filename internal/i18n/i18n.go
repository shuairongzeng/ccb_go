package i18n

import (
	"os"
	"strings"
)

// Supported languages
const (
	LangEN = "en"
	LangZH = "zh"
	LangJA = "ja"
)

// Messages holds all translatable strings.
type Messages struct {
	// Common errors
	ErrTimeout       string
	ErrNoReply       string
	ErrDaemonDown    string
	ErrSessionNotSet string
	ErrLockFailed    string
	ErrPaneDead      string
	ErrSendFailed    string
	ErrNoBackend     string
	ErrNoSession     string
	ErrInvalidToken  string
	ErrUnknownMethod string

	// Daemon lifecycle
	DaemonStarting string
	DaemonStarted  string
	DaemonStopping string
	DaemonStopped  string
	DaemonAlready  string
	DaemonNotFound string

	// Provider status
	ProviderPinging string
	ProviderOnline  string
	ProviderOffline string

	// Ask flow
	AskSending  string
	AskWaiting  string
	AskReceived string

	// Terminal detection
	TermDetecting    string
	TermTmuxFound    string
	TermWeztermFound string
	TermPSFound      string
	TermNoneFound    string

	// Session resolution
	SessionResolving  string
	SessionFound      string
	SessionNotFound   string
	SessionBinding    string
	SessionBound      string
	SessionExpired    string

	// Pane management
	PaneCreating   string
	PaneCreated    string
	PaneKilling    string
	PaneKilled     string
	PaneNotAlive   string
	PaneAlive      string

	// Communication
	CommAnchorFound   string
	CommDoneFound     string
	CommFallbackScan  string
	CommRebind        string
	CommPollStart     string
	CommPollTimeout   string

	// Debug/diagnostic
	DebugLogPath     string
	DebugReqID       string
	DebugSessionKey  string
	DebugAnchorMs    string
	DebugDoneMs      string
}

var translations = map[string]*Messages{
	LangEN: {
		ErrTimeout:       "Timeout waiting for reply",
		ErrNoReply:       "No reply received",
		ErrDaemonDown:    "Daemon is not running",
		ErrSessionNotSet: "Session not configured",
		ErrLockFailed:    "Failed to acquire lock",
		ErrPaneDead:      "Provider pane is no longer alive",
		ErrSendFailed:    "Failed to send message",
		ErrNoBackend:     "No terminal backend available",
		ErrNoSession:     "No session found for provider",
		ErrInvalidToken:  "Invalid authentication token",
		ErrUnknownMethod: "Unknown request method",

		DaemonStarting: "Starting daemon...",
		DaemonStarted:  "Daemon started",
		DaemonStopping: "Stopping daemon...",
		DaemonStopped:  "Daemon stopped",
		DaemonAlready:  "Daemon is already running",
		DaemonNotFound: "Daemon state file not found",

		ProviderPinging: "Pinging %s...",
		ProviderOnline:  "%s is online",
		ProviderOffline: "%s is offline",

		AskSending:  "Sending to %s...",
		AskWaiting:  "Waiting for %s reply...",
		AskReceived: "Reply received from %s",

		TermDetecting:    "Detecting terminal backend...",
		TermTmuxFound:    "Using tmux backend",
		TermWeztermFound: "Using WezTerm backend",
		TermPSFound:      "Using PowerShell backend",
		TermNoneFound:    "No terminal backend found",

		SessionResolving: "Resolving session for %s...",
		SessionFound:     "Session found: %s (source: %s)",
		SessionNotFound:  "No session found for %s",
		SessionBinding:   "Binding session %s...",
		SessionBound:     "Session bound: %s",
		SessionExpired:   "Session expired: %s",

		PaneCreating: "Creating pane for %s...",
		PaneCreated:  "Pane created: %s",
		PaneKilling:  "Killing pane %s...",
		PaneKilled:   "Pane killed: %s",
		PaneNotAlive: "Pane %s is not alive",
		PaneAlive:    "Pane %s is alive",

		CommAnchorFound:  "Anchor found (req_id: %s)",
		CommDoneFound:    "Done marker found (req_id: %s)",
		CommFallbackScan: "Using fallback scan for %s",
		CommRebind:       "Rebinding session for %s",
		CommPollStart:    "Starting poll for %s (interval: %dms)",
		CommPollTimeout:  "Poll timeout for %s after %ds",

		DebugLogPath:    "Log path: %s",
		DebugReqID:      "Request ID: %s",
		DebugSessionKey: "Session key: %s",
		DebugAnchorMs:   "Anchor detected at %dms",
		DebugDoneMs:     "Done detected at %dms",
	},
	LangZH: {
		ErrTimeout:       "等待回复超时",
		ErrNoReply:       "未收到回复",
		ErrDaemonDown:    "守护进程未运行",
		ErrSessionNotSet: "会话未配置",
		ErrLockFailed:    "获取锁失败",
		ErrPaneDead:      "提供者面板已不存在",
		ErrSendFailed:    "发送消息失败",
		ErrNoBackend:     "没有可用的终端后端",
		ErrNoSession:     "未找到提供者的会话",
		ErrInvalidToken:  "无效的认证令牌",
		ErrUnknownMethod: "未知的请求方法",

		DaemonStarting: "正在启动守护进程...",
		DaemonStarted:  "守护进程已启动",
		DaemonStopping: "正在停止守护进程...",
		DaemonStopped:  "守护进程已停止",
		DaemonAlready:  "守护进程已在运行",
		DaemonNotFound: "未找到守护进程状态文件",

		ProviderPinging: "正在 ping %s...",
		ProviderOnline:  "%s 在线",
		ProviderOffline: "%s 离线",

		AskSending:  "正在发送到 %s...",
		AskWaiting:  "正在等待 %s 回复...",
		AskReceived: "已收到 %s 的回复",

		TermDetecting:    "正在检测终端后端...",
		TermTmuxFound:    "使用 tmux 后端",
		TermWeztermFound: "使用 WezTerm 后端",
		TermPSFound:      "使用 PowerShell 后端",
		TermNoneFound:    "未找到终端后端",

		SessionResolving: "正在解析 %s 的会话...",
		SessionFound:     "找到会话: %s (来源: %s)",
		SessionNotFound:  "未找到 %s 的会话",
		SessionBinding:   "正在绑定会话 %s...",
		SessionBound:     "会话已绑定: %s",
		SessionExpired:   "会话已过期: %s",

		PaneCreating: "正在为 %s 创建面板...",
		PaneCreated:  "面板已创建: %s",
		PaneKilling:  "正在关闭面板 %s...",
		PaneKilled:   "面板已关闭: %s",
		PaneNotAlive: "面板 %s 已不存在",
		PaneAlive:    "面板 %s 存活",

		CommAnchorFound:  "找到锚点 (req_id: %s)",
		CommDoneFound:    "找到完成标记 (req_id: %s)",
		CommFallbackScan: "使用回退扫描: %s",
		CommRebind:       "正在重新绑定 %s 的会话",
		CommPollStart:    "开始轮询 %s (间隔: %dms)",
		CommPollTimeout:  "%s 轮询超时 (%ds)",

		DebugLogPath:    "日志路径: %s",
		DebugReqID:      "请求 ID: %s",
		DebugSessionKey: "会话键: %s",
		DebugAnchorMs:   "锚点检测于 %dms",
		DebugDoneMs:     "完成检测于 %dms",
	},
	LangJA: {
		ErrTimeout:       "応答待ちタイムアウト",
		ErrNoReply:       "応答なし",
		ErrDaemonDown:    "デーモンが実行されていません",
		ErrSessionNotSet: "セッションが設定されていません",
		ErrLockFailed:    "ロックの取得に失敗しました",
		ErrPaneDead:      "プロバイダーペインが存在しません",
		ErrSendFailed:    "メッセージの送信に失敗しました",
		ErrNoBackend:     "利用可能なターミナルバックエンドがありません",
		ErrNoSession:     "プロバイダーのセッションが見つかりません",
		ErrInvalidToken:  "無効な認証トークン",
		ErrUnknownMethod: "不明なリクエストメソッド",

		DaemonStarting: "デーモンを起動中...",
		DaemonStarted:  "デーモンが起動しました",
		DaemonStopping: "デーモンを停止中...",
		DaemonStopped:  "デーモンが停止しました",
		DaemonAlready:  "デーモンは既に実行中です",
		DaemonNotFound: "デーモン状態ファイルが見つかりません",

		ProviderPinging: "%s に ping 中...",
		ProviderOnline:  "%s はオンラインです",
		ProviderOffline: "%s はオフラインです",

		AskSending:  "%s に送信中...",
		AskWaiting:  "%s の応答を待機中...",
		AskReceived: "%s から応答を受信しました",

		TermDetecting:    "ターミナルバックエンドを検出中...",
		TermTmuxFound:    "tmux バックエンドを使用",
		TermWeztermFound: "WezTerm バックエンドを使用",
		TermPSFound:      "PowerShell バックエンドを使用",
		TermNoneFound:    "ターミナルバックエンドが見つかりません",

		SessionResolving: "%s のセッションを解決中...",
		SessionFound:     "セッション発見: %s (ソース: %s)",
		SessionNotFound:  "%s のセッションが見つかりません",
		SessionBinding:   "セッション %s をバインド中...",
		SessionBound:     "セッションバインド完了: %s",
		SessionExpired:   "セッション期限切れ: %s",

		PaneCreating: "%s のペインを作成中...",
		PaneCreated:  "ペイン作成完了: %s",
		PaneKilling:  "ペイン %s を終了中...",
		PaneKilled:   "ペイン終了完了: %s",
		PaneNotAlive: "ペイン %s は存在しません",
		PaneAlive:    "ペイン %s は存在します",

		CommAnchorFound:  "アンカー検出 (req_id: %s)",
		CommDoneFound:    "完了マーカー検出 (req_id: %s)",
		CommFallbackScan: "%s のフォールバックスキャンを使用",
		CommRebind:       "%s のセッションを再バインド",
		CommPollStart:    "%s のポーリング開始 (間隔: %dms)",
		CommPollTimeout:  "%s のポーリングタイムアウト (%ds)",

		DebugLogPath:    "ログパス: %s",
		DebugReqID:      "リクエストID: %s",
		DebugSessionKey: "セッションキー: %s",
		DebugAnchorMs:   "アンカー検出: %dms",
		DebugDoneMs:     "完了検出: %dms",
	},
}

// DetectLanguage detects the user's preferred language from environment variables.
func DetectLanguage() string {
	for _, envVar := range []string{"CCB_LANG", "LANG", "LC_ALL", "LC_MESSAGES", "LANGUAGE"} {
		val := strings.TrimSpace(os.Getenv(envVar))
		if val == "" {
			continue
		}
		lower := strings.ToLower(val)
		if strings.HasPrefix(lower, "zh") {
			return LangZH
		}
		if strings.HasPrefix(lower, "ja") {
			return LangJA
		}
		if strings.HasPrefix(lower, "en") {
			return LangEN
		}
	}
	return LangEN
}

// Get returns the Messages for the detected language.
func Get() *Messages {
	return GetLang(DetectLanguage())
}

// GetLang returns the Messages for a specific language.
func GetLang(lang string) *Messages {
	if m, ok := translations[lang]; ok {
		return m
	}
	return translations[LangEN]
}
