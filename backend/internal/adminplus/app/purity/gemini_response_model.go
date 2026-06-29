package purity

import (
	"strings"

	"github.com/tidwall/gjson"
)

func geminiResponseModelFromProbe(probe httpProbe, fallbackModel string) (string, string, string, string) {
	bodyModel := strings.TrimSpace(gjson.GetBytes(probe.Body, "modelVersion").String())
	if bodyModel != "" {
		return bodyModel, "body.modelVersion", bodyModel, ""
	}
	fallbackModel = strings.TrimPrefix(strings.TrimSpace(fallbackModel), "models/")
	if fallbackModel != "" {
		return fallbackModel, "request.model", "", ""
	}
	return "", "", "", ""
}
