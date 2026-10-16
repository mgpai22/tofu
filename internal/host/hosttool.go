package host

import (
	"context"
	"encoding/json"
	"sync"

	"tofu/internal/llm"
	"tofu/internal/session"
	"tofu/internal/turn"
)

// Host tools are owned by the stdio RPC client. The model only sees the
// registered definitions; every call blocks the turn on the client's answer.

const ToolCallMethod = "tofu/callTool"

type HostToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ToolsRegisterParams struct {
	Tools  []HostToolDef `json:"tools"`
	System string        `json:"system"`
}

type ToolCallRequest struct {
	Identity
	Call string          `json:"call"`
	Tool string          `json:"tool"`
	Args json.RawMessage `json:"args"`
}

type ToolCallAnswer struct {
	Allowed bool   `json:"allowed"`
	Result  string `json:"result"`
}

var hostRegistryMu sync.Mutex
var hostRegistryTools []HostToolDef
var hostRegistrySystem string
var hostRegistrySet bool

func RegisterHostTools(tools []HostToolDef, system string) {
	hostRegistryMu.Lock()
	defer hostRegistryMu.Unlock()
	hostRegistryTools = tools
	hostRegistrySystem = system
	hostRegistrySet = true
}

func HostToolsRegistered() ([]HostToolDef, string, bool) {
	hostRegistryMu.Lock()
	defer hostRegistryMu.Unlock()
	return hostRegistryTools, hostRegistrySystem, hostRegistrySet
}

var hostWaiters sync.Map // call ID -> chan ToolCallAnswer

type HostCaller func(ctx context.Context, tool string, args json.RawMessage) (ToolCallAnswer, error)

func WaitHostTool(ctx context.Context, emit func(Event), tool string, args json.RawMessage) (ToolCallAnswer, error) {
	id := session.NewEventID()
	wait := make(chan ToolCallAnswer, 1)
	hostWaiters.Store(id, wait)
	defer hostWaiters.Delete(id)
	emit(Event{Kind: EventHostTool, ID: id, Tool: tool, Args: args})
	select {
	case answered := <-wait:
		return answered, nil
	case <-ctx.Done():
		return ToolCallAnswer{}, ctx.Err()
	}
}

func HasHostToolWaiter(id string) bool {
	_, found := hostWaiters.Load(id)
	return found
}

func AnswerHostTool(id string, answer ToolCallAnswer) bool {
	waited, found := hostWaiters.Load(id)
	if !found {
		return false
	}
	hostWaiters.Delete(id)
	waited.(chan ToolCallAnswer) <- answer
	return true
}

type hostTool struct {
	def    HostToolDef
	caller HostCaller
}

func MakeHostTools(caller HostCaller) []turn.Tool {
	defs, _, set := HostToolsRegistered()
	if !set {
		return nil
	}
	tools := make([]turn.Tool, 0, len(defs))
	for _, def := range defs {
		tools = append(tools, hostTool{def: def, caller: caller})
	}
	return tools
}

func (t hostTool) Name() string { return t.def.Name }

func (t hostTool) Definition() llm.Tool {
	return llm.Tool{Name: t.def.Name, Description: t.def.Description, Parameters: t.def.Parameters}
}

func (t hostTool) Run(ctx context.Context, args json.RawMessage) (turn.Result, error) {
	answered, err := t.caller(ctx, t.def.Name, args)
	if err != nil {
		return turn.Result{}, err
	}
	if !answered.Allowed {
		denied := "DENIED by host policy: " + t.def.Name + " is not allowed"
		return turn.Result{Content: denied, FailureText: denied, Outcome: turn.ResultFailed}, nil
	}
	return turn.Result{Content: answered.Result, Outcome: turn.ResultSucceeded}, nil
}
