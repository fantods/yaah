package message

type StopReason string

const (
	StopReasonStop    StopReason = "stop"
	StopReasonLength  StopReason = "length"
	StopReasonToolUse StopReason = "toolUse"
	StopReasonAborted StopReason = "aborted"
	StopReasonError   StopReason = "error"
)

type Cost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	Total      float64 `json:"total"`
}

type Usage struct {
	Input       int64 `json:"input"`
	Output      int64 `json:"output"`
	CacheRead   int64 `json:"cacheRead"`
	CacheWrite  int64 `json:"cacheWrite"`
	TotalTokens int64 `json:"totalTokens"`
	Cost        Cost  `json:"cost"`
}

func (u Usage) Add(other Usage) Usage {
	return Usage{
		Input:       u.Input + other.Input,
		Output:      u.Output + other.Output,
		CacheRead:   u.CacheRead + other.CacheRead,
		CacheWrite:  u.CacheWrite + other.CacheWrite,
		TotalTokens: u.TotalTokens + other.TotalTokens,
		Cost: Cost{
			Input:      u.Cost.Input + other.Cost.Input,
			Output:     u.Cost.Output + other.Cost.Output,
			CacheRead:  u.Cost.CacheRead + other.Cost.CacheRead,
			CacheWrite: u.Cost.CacheWrite + other.Cost.CacheWrite,
			Total:      u.Cost.Total + other.Cost.Total,
		},
	}
}
