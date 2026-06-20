package domain

import (
	"strings"
	"time"
)

type SupplierKeyStatus string

const (
	SupplierKeyStatusProvisioning         SupplierKeyStatus = "provisioning"
	SupplierKeyStatusBound                SupplierKeyStatus = "bound"
	SupplierKeyStatusManualSecretRequired SupplierKeyStatus = "manual_secret_required"
	SupplierKeyStatusFailed               SupplierKeyStatus = "failed"
	SupplierKeyStatusDisabled             SupplierKeyStatus = "disabled"
)

type SupplierKey struct {
	ID                     int64             `json:"id"`
	SupplierID             int64             `json:"supplier_id"`
	SupplierGroupID        int64             `json:"supplier_group_id"`
	ExternalGroupID        string            `json:"external_group_id"`
	ExternalKeyID          string            `json:"external_key_id"`
	Name                   string            `json:"name"`
	KeyFingerprint         string            `json:"key_fingerprint"`
	KeyLast4               string            `json:"key_last4"`
	Status                 SupplierKeyStatus `json:"status"`
	ProviderFamily         string            `json:"provider_family"`
	LocalSub2APIAccountID  int64             `json:"local_sub2api_account_id,omitempty"`
	LocalAccountName       string            `json:"local_account_name,omitempty"`
	LocalAccountPlatform   string            `json:"local_account_platform,omitempty"`
	LocalAccountType       string            `json:"local_account_type,omitempty"`
	ProvisionRequest       map[string]any    `json:"provision_request,omitempty"`
	ProvisionResponse      map[string]any    `json:"provision_response,omitempty"`
	ErrorCode              string            `json:"error_code,omitempty"`
	ErrorMessage           string            `json:"error_message,omitempty"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
}

func (s SupplierKeyStatus) Valid() bool {
	switch s {
	case SupplierKeyStatusProvisioning, SupplierKeyStatusBound, SupplierKeyStatusManualSecretRequired, SupplierKeyStatusFailed, SupplierKeyStatusDisabled:
		return true
	default:
		return false
	}
}

func NormalizeSupplierKeyStatus(value string) SupplierKeyStatus {
	return SupplierKeyStatus(strings.ToLower(strings.TrimSpace(value)))
}
