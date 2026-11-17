package groundwork

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/influxdata/telegraf"
)

// Targets for rule processing
const (
	ruleTargetResource = "resource"
	ruleTargetGroup    = "group"
	ruleTargetService  = "service"
	ruleTargetStatus   = "status"
	ruleTargetMessage  = "message"
	ruleTargetSkip     = "skip"
)

var ruleTargets = []string{
	ruleTargetResource,
	ruleTargetGroup,
	ruleTargetService,
	ruleTargetStatus,
	ruleTargetMessage,
	ruleTargetSkip,
}

// rule describes how to build one of the GroundWork values from a tag or a
// field of the metric. Rules are evaluated in the order they are configured
// and the first rule that applies wins.
type rule struct {
	Target   string `toml:"target"`
	Tag      string `toml:"tag"`
	Field    string `toml:"field"`
	Pattern  string `toml:"pattern"`
	Template string `toml:"template"`

	pattern  *regexp.Regexp
	template *template.Template
}

// ruleContext carries the values made available to the rule template. The
// embedded metric provides access to the measurement name, tags and fields,
// e.g. "{{ .Name }}", "{{ .Tag \"cluster\" }}", "{{ .Field \"load\" }}".
type ruleContext struct {
	telegraf.TemplateMetric
	// Value is the raw value of the configured tag or field. It is empty if no source is configured.
	Value string
	// Match is the full regular expression match. It is empty if no pattern is configured.
	Match string
	// Groups contains the named capture groups of the regular expression.
	Groups map[string]string
}

// validateRules checks the rules and compiles the regular expressions and templates.
func validateRules(rules []rule) error {
	for i := range rules {
		r := &rules[i]

		target := strings.ToLower(strings.TrimSpace(r.Target))
		valid := false
		for _, t := range ruleTargets {
			if target == t {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("rule %d: invalid target %q, must be one of %s", i+1, r.Target, strings.Join(ruleTargets, ", "))
		}
		r.Target = target

		if r.Tag != "" && r.Field != "" {
			return fmt.Errorf("rule %d: only one of \"tag\" and \"field\" may be set", i+1)
		}
		if r.Tag == "" && r.Field == "" && r.Pattern != "" {
			return fmt.Errorf("rule %d: \"pattern\" requires a \"tag\" or \"field\" to match against", i+1)
		}
		if r.Tag == "" && r.Field == "" && r.Template == "" {
			return fmt.Errorf("rule %d: at least one of \"tag\", \"field\" or \"template\" must be set", i+1)
		}

		if r.Pattern != "" {
			pattern, err := regexp.Compile(r.Pattern)
			if err != nil {
				return fmt.Errorf("rule %d: invalid pattern: %w", i+1, err)
			}
			r.pattern = pattern
		}

		if r.Template != "" {
			tmpl, err := template.New(fmt.Sprintf("rule-%d", i+1)).Funcs(sprig.TxtFuncMap()).Parse(r.Template)
			if err != nil {
				return fmt.Errorf("rule %d: invalid template: %w", i+1, err)
			}
			r.template = tmpl
		}
	}

	return nil
}

// ruleSourceKeys returns the set of tags and fields used as input for a
// rule. Keys of skip rules are not reported, since skipped metrics never
// reach the payload.
func ruleSourceKeys(rules []rule) (tags, fields map[string]bool) {
	tags, fields = make(map[string]bool), make(map[string]bool)
	for i := range rules {
		if rules[i].Target == ruleTargetSkip {
			continue
		}
		if rules[i].Tag != "" {
			tags[rules[i].Tag] = true
		}
		if rules[i].Field != "" {
			fields[rules[i].Field] = true
		}
	}
	return tags, fields
}

// evalRule evaluates a single rule against the metric. It reports whether the
// rule applies and, if so, the resulting value. The rule applies when the
// configured source (if any) exists, the pattern (if any) matches and the
// resulting value is not empty.
func (g *Groundwork) evalRule(r *rule, m telegraf.Metric) (string, bool) {
	src, hasSource := ruleSource(r, m)
	if !hasSource {
		return "", false
	}

	match := ""
	groups := map[string]string{}
	if r.pattern != nil {
		locs := r.pattern.FindStringSubmatchIndex(src)
		if locs == nil {
			return "", false
		}
		match = src[locs[0]:locs[1]]
		for i, name := range r.pattern.SubexpNames() {
			if i == 0 || name == "" || locs[2*i] < 0 {
				continue
			}
			groups[name] = src[locs[2*i]:locs[2*i+1]]
		}
	}

	out := src
	switch {
	case r.template != nil:
		var sb strings.Builder
		tm, ok := m.(telegraf.TemplateMetric)
		if !ok {
			g.Log.Warnf("rule with target %q: metric of type %T does not implement telegraf.TemplateMetric", r.Target, m)
			return "", false
		}
		data := ruleContext{
			TemplateMetric: tm,
			Value:          src,
			Match:          match,
			Groups:         groups,
		}
		if err := r.template.Execute(&sb, data); err != nil {
			g.Log.Warnf("rule with target %q: failed to execute template: %v", r.Target, err)
			return "", false
		}
		out = sb.String()
	case r.pattern != nil:
		out = match
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return "", false
	}

	return out, true
}

// ruleValue returns the value of the first matching rule for the given target.
func (g *Groundwork) ruleValue(m telegraf.Metric, target string) (string, bool) {
	for i := range g.Rules {
		if g.Rules[i].Target != target {
			continue
		}
		if value, ok := g.evalRule(&g.Rules[i], m); ok {
			return value, true
		}
	}
	return "", false
}

// ruleStatusValue returns the first valid status produced by the configured
// status rules. Rules producing an invalid status are logged and skipped.
func (g *Groundwork) ruleStatusValue(m telegraf.Metric) (string, bool) {
	for i := range g.Rules {
		if g.Rules[i].Target != ruleTargetStatus {
			continue
		}
		value, ok := g.evalRule(&g.Rules[i], m)
		if !ok {
			continue
		}
		if !validStatus(value) {
			g.Log.Warnf("rule %d: %q is not a valid status, ignoring the rule result", i+1, value)
			continue
		}
		return value, true
	}
	return "", false
}

// ruleSource resolves the input value for the rule, based on the configured tag or field.
func ruleSource(r *rule, m telegraf.Metric) (string, bool) {
	switch {
	case r.Tag != "":
		v, ok := m.GetTag(r.Tag)
		return v, ok
	case r.Field != "":
		v, ok := m.GetField(r.Field)
		if !ok {
			return "", false
		}
		switch v := v.(type) {
		case string:
			return v, true
		case []byte:
			return string(v), true
		default:
			return fmt.Sprintf("%v", v), true
		}
	default:
		// No source configured, the rule only has a template that works on the metric itself.
		return "", true
	}
}
