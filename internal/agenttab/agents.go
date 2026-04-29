package agenttab

import (
	"fmt"
	"strings"
)

func isAgentSpec(fc FileConfig, raw string) bool {
	name, _, _ := strings.Cut(raw, "/")
	_, ok := fc.Agents[name]
	return ok
}

func parseAgentSpec(fc FileConfig, raw string) (agentSpec, error) {
	name, model, hasModel := strings.Cut(raw, "/")
	name = strings.TrimSpace(name)
	model = strings.TrimSpace(model)
	if name == "" {
		return agentSpec{}, fmt.Errorf("invalid agent spec %q", raw)
	}
	def, ok := fc.Agents[name]
	if !ok || def.Command == "" {
		return agentSpec{}, fmt.Errorf("agent %q is not configured", name)
	}
	if hasModel && model == "" {
		return agentSpec{}, fmt.Errorf("agent %q has empty model in %q", name, raw)
	}
	if hasModel && def.ModelArg == "" {
		return agentSpec{}, fmt.Errorf("agent %q does not support model selection; set model_arg in config", name)
	}
	label := name
	if model != "" {
		label = name + "/" + model
	}
	return agentSpec{Name: name, Model: model, Label: label}, nil
}
