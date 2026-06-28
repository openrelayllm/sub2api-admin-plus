package channels

import "strings"

type Context struct {
	Host         string
	Values       []string
	HeaderValues []string
}

type Detector interface {
	Detect(Context) []string
}

func NewContext(host string, headerSets ...map[string]string) Context {
	values := []string{strings.ToLower(strings.TrimSpace(host))}
	headerValues := []string{}
	for _, headers := range headerSets {
		for key, value := range headers {
			key = strings.ToLower(strings.TrimSpace(key))
			value = strings.ToLower(strings.TrimSpace(value))
			if key == "" && value == "" {
				continue
			}
			if key != "" {
				headerValues = append(headerValues, key)
			}
			if key != "" && value != "" {
				headerValues = append(headerValues, key+":"+value)
			}
		}
	}
	values = append(values, headerValues...)
	return Context{
		Host:         values[0],
		Values:       values,
		HeaderValues: headerValues,
	}
}

func (ctx Context) WithValues(values ...string) Context {
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		ctx.Values = append(ctx.Values, value)
	}
	return ctx
}

func SignalIfContains(ctx Context, signal string, needles ...string) []string {
	if ContainsAny(ctx, needles...) {
		return []string{signal}
	}
	return nil
}

func SignalIfHeaderContains(ctx Context, signal string, needles ...string) []string {
	if ContainsAnyHeader(ctx, needles...) {
		return []string{signal}
	}
	return nil
}

func ContainsAny(ctx Context, needles ...string) bool {
	for _, value := range ctx.Values {
		for _, needle := range needles {
			if strings.Contains(value, needle) {
				return true
			}
		}
	}
	return false
}

func ContainsAnyHeader(ctx Context, needles ...string) bool {
	for _, value := range ctx.HeaderValues {
		for _, needle := range needles {
			if strings.Contains(value, needle) {
				return true
			}
		}
	}
	return false
}

func ContainsSignal(signals []string, target string) bool {
	for _, signal := range signals {
		if signal == target {
			return true
		}
	}
	return false
}
