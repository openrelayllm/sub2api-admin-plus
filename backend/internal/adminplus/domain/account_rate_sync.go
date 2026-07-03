package domain

import "time"

type AccountRateSyncStatus string

const (
	AccountRateSyncStatusMatched   AccountRateSyncStatus = "matched"
	AccountRateSyncStatusRenamed   AccountRateSyncStatus = "renamed"
	AccountRateSyncStatusNotFound  AccountRateSyncStatus = "not_found"
	AccountRateSyncStatusAmbiguous AccountRateSyncStatus = "ambiguous"
	AccountRateSyncStatusFailed    AccountRateSyncStatus = "failed"
)

type AccountRateSyncHistory struct {
	ID                      int64                 `json:"id"`
	LocalSub2APIAccountID   int64                 `json:"local_sub2api_account_id"`
	LocalAccountName        string                `json:"local_account_name"`
	LocalAccountPlatform    string                `json:"local_account_platform"`
	KeyFingerprint          string                `json:"-"`
	KeyLast4                string                `json:"key_last4,omitempty"`
	SupplierID              int64                 `json:"supplier_id,omitempty"`
	SupplierName            string                `json:"supplier_name,omitempty"`
	SupplierType            string                `json:"supplier_type,omitempty"`
	SupplierGroupID         int64                 `json:"supplier_group_id,omitempty"`
	SupplierGroupName       string                `json:"supplier_group_name,omitempty"`
	SupplierKeyID           int64                 `json:"supplier_key_id,omitempty"`
	MatchSource             string                `json:"match_source,omitempty"`
	EffectiveRateMultiplier float64               `json:"effective_rate_multiplier"`
	TargetAccountName       string                `json:"target_account_name,omitempty"`
	Status                  AccountRateSyncStatus `json:"status"`
	ErrorCode               string                `json:"error_code,omitempty"`
	ErrorMessage            string                `json:"error_message,omitempty"`
	Renamed                 bool                  `json:"renamed"`
	OldAccountName          string                `json:"old_account_name,omitempty"`
	NewAccountName          string                `json:"new_account_name,omitempty"`
	SyncedAt                time.Time             `json:"synced_at"`
	CreatedAt               time.Time             `json:"created_at"`
}
