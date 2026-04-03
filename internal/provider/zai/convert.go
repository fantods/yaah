package zai

import (
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
	openaiprovider "github.com/fantods/yaah/internal/provider/openai"
	"github.com/openai/openai-go"
)

func ConvertMessages(msgs []message.Message) []openai.ChatCompletionMessageParamUnion {
	return openaiprovider.ConvertMessages(msgs)
}

func ConvertTools(tools []provider.Tool) []openai.ChatCompletionToolParam {
	return openaiprovider.ConvertTools(tools)
}
