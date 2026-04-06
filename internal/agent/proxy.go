package agent

import (
	"fmt"

	"github.com/fantods/yaah/internal/logging"
	"github.com/fantods/yaah/internal/message"
	"github.com/fantods/yaah/internal/provider"
)

func StreamProxy(model provider.Model, ctx provider.Context, opts *provider.StreamOptions) *provider.AssistantMessageEventStream {
	p, ok := provider.Lookup(model.API)
	if !ok {
		err := fmt.Errorf("no provider registered for api: %s", model.API)
		logging.Debug("stream proxy: %v", err)
		stream := provider.NewEventStream(
			func(provider.AssistantMessageEvent) bool { return false },
			func(provider.AssistantMessageEvent) message.AssistantMessage {
				return message.AssistantMessage{}
			},
		)
		go func() {
			stream.Push(provider.EventError{Err: err})
			stream.End(nil)
		}()
		return stream
	}
	logging.Debug("stream proxy: delegating to provider for api=%s model=%s", model.API, model.ID)
	return p.Stream(model, ctx, opts)
}
