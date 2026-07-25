package signature

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

const (
	ClassificationIdentified = "identified"
	ClassificationLikely     = "likely"
	ClassificationUnknown    = "unknown"
)

//go:embed families.json
var defaultFamiliesJSON []byte

type Family struct {
	ID                      string   `json:"id"`
	Channel                 string   `json:"channel"`
	ModelFamilies           []string `json:"model_families"`
	RequiredTopLevelFields  []int    `json:"required_top_level_fields"`
	RequiredEnvelopeFields  []int    `json:"required_envelope_fields"`
	RequiredMetadataFields  []int    `json:"required_metadata_fields"`
	ForbiddenMetadataFields []int    `json:"forbidden_metadata_fields"`
	MinSamples              int      `json:"min_samples"`
	SampleCount             int      `json:"sample_count"`
	SourceType              string   `json:"source_type"`
	ValidFrom               string   `json:"valid_from"`
	Limitations             []string `json:"limitations"`
	RiskCodes               []string `json:"risk_codes"`
}

type Registry struct {
	Version  string   `json:"version"`
	Families []Family `json:"families"`
}

type Classification struct {
	Channel     string
	Status      string
	Confidence  float64
	FamilyID    string
	SampleCount int
	SourceType  string
	Limitations []string
	RiskCodes   []string
	Fingerprint Fingerprint
}

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     Registry
	defaultRegistryErr  error
)

func LoadRegistry(data []byte) (Registry, error) {
	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return Registry{}, fmt.Errorf("decode signature registry: %w", err)
	}
	if strings.TrimSpace(registry.Version) == "" || len(registry.Families) == 0 {
		return Registry{}, fmt.Errorf("signature registry is empty")
	}
	for _, family := range registry.Families {
		if strings.TrimSpace(family.ID) == "" || strings.TrimSpace(family.Channel) == "" || family.MinSamples <= 0 {
			return Registry{}, fmt.Errorf("invalid signature family")
		}
	}
	return registry, nil
}

func DefaultRegistry() (Registry, error) {
	defaultRegistryOnce.Do(func() {
		defaultRegistry, defaultRegistryErr = LoadRegistry(defaultFamiliesJSON)
	})
	return defaultRegistry, defaultRegistryErr
}

func (registry Registry) Classify(fingerprint Fingerprint, model string) Classification {
	matches := make([]Family, 0, 2)
	for _, family := range registry.Families {
		if familyMatches(family, fingerprint, model) {
			matches = append(matches, family)
		}
	}
	if len(matches) != 1 {
		return Classification{
			Status:      ClassificationUnknown,
			Confidence:  0,
			Fingerprint: fingerprint,
			Limitations: []string{"signature_structure_not_in_calibrated_registry"},
		}
	}
	family := matches[0]
	status := ClassificationLikely
	confidence := 0.78
	if family.SampleCount >= family.MinSamples {
		status = ClassificationIdentified
		confidence = 0.96
	}
	return Classification{
		Channel:     family.Channel,
		Status:      status,
		Confidence:  confidence,
		FamilyID:    family.ID,
		SampleCount: family.SampleCount,
		SourceType:  family.SourceType,
		Limitations: append([]string(nil), family.Limitations...),
		RiskCodes:   append([]string(nil), family.RiskCodes...),
		Fingerprint: fingerprint,
	}
}

func familyMatches(family Family, fingerprint Fingerprint, model string) bool {
	if !modelMatches(family.ModelFamilies, model) {
		return false
	}
	return containsAll(fingerprint.TopLevelFields, family.RequiredTopLevelFields) &&
		containsAll(fingerprint.EnvelopeFields, family.RequiredEnvelopeFields) &&
		containsAll(fingerprint.MetadataFields, family.RequiredMetadataFields) &&
		containsNone(fingerprint.MetadataFields, family.ForbiddenMetadataFields)
}

func modelMatches(families []string, model string) bool {
	if len(families) == 0 {
		return true
	}
	model = strings.ToLower(strings.TrimSpace(model))
	for _, family := range families {
		if strings.Contains(model, strings.ToLower(strings.TrimSpace(family))) {
			return true
		}
	}
	return false
}

func containsAll(actual []int, required []int) bool {
	set := make(map[int]struct{}, len(actual))
	for _, value := range actual {
		set[value] = struct{}{}
	}
	for _, value := range required {
		if _, ok := set[value]; !ok {
			return false
		}
	}
	return true
}

func containsNone(actual []int, forbidden []int) bool {
	set := make(map[int]struct{}, len(actual))
	for _, value := range actual {
		set[value] = struct{}{}
	}
	for _, value := range forbidden {
		if _, ok := set[value]; ok {
			return false
		}
	}
	return true
}
