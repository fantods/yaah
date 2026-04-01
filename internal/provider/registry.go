package provider

import "sync"

type StreamFn func(model Model, ctx Context, opts *StreamOptions) *AssistantMessageEventStream

type Provider struct {
	API          string
	Stream       StreamFn
	StreamSimple StreamFn
}

var providers sync.Map

func Register(p Provider) {
	providers.Store(p.API, p)
}

func Lookup(api string) (Provider, bool) {
	v, ok := providers.Load(api)
	if !ok {
		return Provider{}, false
	}
	return v.(Provider), true
}

func ResetRegistry() {
	providers = sync.Map{}
}
