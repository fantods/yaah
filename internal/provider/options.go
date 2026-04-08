package provider

type ThinkingLevel string

const (
	ThinkingLevelMinimal ThinkingLevel = "minimal"
	ThinkingLevelLow     ThinkingLevel = "low"
	ThinkingLevelMedium  ThinkingLevel = "medium"
	ThinkingLevelHigh    ThinkingLevel = "high"
	ThinkingLevelXHigh   ThinkingLevel = "xhigh"
)

type ThinkingBudgets struct {
	Minimal *int `json:"minimal,omitempty"`
	Low     *int `json:"low,omitempty"`
	Medium  *int `json:"medium,omitempty"`
	High    *int `json:"high,omitempty"`
}

type CacheRetention string

const (
	CacheRetentionNone  CacheRetention = "none"
	CacheRetentionShort CacheRetention = "short"
	CacheRetentionLong  CacheRetention = "long"
)

type Transport string

const (
	TransportSSE       Transport = "sse"
	TransportWebsocket Transport = "websocket"
	TransportAuto      Transport = "auto"
)

type StreamOptions struct {
	Temperature     *float64          `json:"temperature,omitempty"`
	MaxTokens       *int              `json:"maxTokens,omitempty"`
	APIKey          *string           `json:"apiKey,omitempty"`
	Transport       Transport         `json:"transport,omitempty"`
	CacheRetention  CacheRetention    `json:"cacheRetention,omitempty"`
	SessionID       string            `json:"sessionId,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	MaxRetryDelayMs *int              `json:"maxRetryDelayMs,omitempty"`
	Metadata        map[string]any    `json:"metadata,omitempty"`
	ThinkingEnabled bool              `json:"thinkingEnabled,omitempty"`
}

type SimpleStreamOptions struct {
	StreamOptions
	Reasoning       ThinkingLevel    `json:"reasoning,omitempty"`
	ThinkingBudgets *ThinkingBudgets `json:"thinkingBudgets,omitempty"`
}
