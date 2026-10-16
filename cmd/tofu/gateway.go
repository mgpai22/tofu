package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"tofu/internal/llm/models"
	"tofu/internal/llm/wire/anthropic"
	"tofu/internal/turn"
)

// A gateway is any Anthropic Messages endpoint reached with a plain API key,
// such as a local proxy. TOFU_GATEWAY_BASE names it without /v1; when it is
// set, every turn goes there and no subscription or catalog is read.
const (
	gatewayBaseVariable    = "TOFU_GATEWAY_BASE"
	gatewayKeyVariable     = "TOFU_GATEWAY_KEY"
	gatewayModelVariable   = "TOFU_GATEWAY_MODEL"
	gatewayHeadersVariable = "TOFU_GATEWAY_HEADERS"
)

func gatewayWire(opts runOpts) (appWire, error) {
	model := cmp.Or(opts.model, os.Getenv(gatewayModelVariable))
	key := os.Getenv(gatewayKeyVariable)
	if model == "" || key == "" {
		return appWire{}, errors.New(gatewayBaseVariable + " is set, so " + gatewayModelVariable + " and " + gatewayKeyVariable + " are needed too")
	}
	headers := map[string]string{}
	if raw := os.Getenv(gatewayHeadersVariable); raw != "" {
		if err := json.Unmarshal([]byte(raw), &headers); err != nil {
			return appWire{}, errors.New(gatewayHeadersVariable + " is not a JSON object of strings: " + err.Error())
		}
	}
	wire, err := anthropic.New(anthropic.Config{
		BaseURL:   os.Getenv(gatewayBaseVariable),
		Proxy:     true,
		Model:     model,
		Token:     func(context.Context, string) (string, error) { return key, nil },
		HTTP:      &http.Client{Transport: extraHeaders{base: http.DefaultTransport, headers: headers}},
		Transport: turnTransportConfig(),
	})
	if err != nil {
		return appWire{}, err
	}
	fixed := turn.Subscription{Wire: wire, Effort: opts.effort}
	return appWire{held: &accounts{modelID: model, now: time.Now, fixed: fixed}, spend: turn.SpendAPIKey, selected: models.Model{ID: model}}, nil
}

type extraHeaders struct {
	base    http.RoundTripper
	headers map[string]string
}

func (e extraHeaders) RoundTrip(request *http.Request) (*http.Response, error) {
	if len(e.headers) > 0 {
		request = request.Clone(request.Context())
		for name, value := range e.headers {
			request.Header.Set(name, value)
		}
	}
	return e.base.RoundTrip(request)
}
