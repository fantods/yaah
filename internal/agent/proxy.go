package agent

import (
	"fmt"

	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

func StreamProxy(model provider.Model, ctx provider.Context, opts *provider.StreamOptions) *provider.AssistantMessageEventStream {
	p, ok := provider.Lookup(model.API)
	if !ok {
		stream := provider.NewEventStream(
			func(provider.AssistantMessageEvent) bool { return false },
			func(provider.AssistantMessageEvent) message.AssistantMessage {
				return message.AssistantMessage{}
			},
		)
		go func() {
			stream.End(nil)
		}()
		_ = fmt.Errorf("no provider registered for api: %s", model.API)
		return stream
	}
	return p.Stream(model, ctx, opts)
}
