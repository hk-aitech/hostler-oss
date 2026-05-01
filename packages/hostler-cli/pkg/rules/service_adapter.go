// Package rules — ServiceAdapter.
//
// The five cmd-facing functions (List / Get / LoadConfig / LoadConfigBytes /
// ResolveEffective) of pkg/rules are passed through verbatim to satisfy the
// app.RuleRunnerService contract. All return values are typed as `any` so
// that internal/app keeps its prohibition on importing pkg/rules.
//
// cmd code asserts the concrete type from `any`
// (e.g. `allRules := raw.([]rules.Rule)`).
//
// The pre-existing RulesRegistryAdapter is a separate adapter that exposes
// the domain-neutral RuleDescriptor of ports.RuleRunner. This ServiceAdapter
// is a convenience pass-through for direct cmd invocation, so the two have
// distinct purposes.
package rules

// ServiceAdapter is a pass-through adapter for app.RuleRunnerService.
type ServiceAdapter struct{}

// NewServiceAdapter constructs a ServiceAdapter that uses the global
// registry / loader.
func NewServiceAdapter() *ServiceAdapter { return &ServiceAdapter{} }

// List passes through to rules.List. Returns []rules.Rule wrapped as any.
func (ServiceAdapter) List() any { return List() }

// Get passes through to rules.Get. Returns rules.Rule wrapped as any.
func (ServiceAdapter) Get(ruleID string) (any, bool) {
	r, ok := Get(ruleID)
	if !ok {
		return nil, false
	}
	return r, true
}

// LoadConfig passes through to rules.LoadConfig.
// Returns *rules.EngineConfig wrapped as any.
func (ServiceAdapter) LoadConfig(projectRoot string) (any, error) {
	return LoadConfig(projectRoot)
}

// LoadConfigBytes passes through to rules.LoadConfigBytes.
// Returns *rules.EngineConfig wrapped as any.
func (ServiceAdapter) LoadConfigBytes(data []byte) (any, error) {
	return LoadConfigBytes(data)
}

// ResolveEffective passes through to rules.ResolveEffective.
// cfg is the concrete *EngineConfig type — received as any and asserted
// internally. Returns map[string]*EffectiveRule wrapped as any.
func (ServiceAdapter) ResolveEffective(cfg any, projectRoot string) (any, error) {
	c, _ := cfg.(*EngineConfig)
	return ResolveEffective(c, projectRoot)
}
