package domain

import "time"

type SupplierChannelProbeStatus string

const (
	SupplierChannelProbeStatusUntested          SupplierChannelProbeStatus = "untested"
	SupplierChannelProbeStatusAvailable         SupplierChannelProbeStatus = "available"
	SupplierChannelProbeStatusSlowFirstToken    SupplierChannelProbeStatus = "slow_first_token"
	SupplierChannelProbeStatusSlowTotal         SupplierChannelProbeStatus = "slow_total"
	SupplierChannelProbeStatusRequestError      SupplierChannelProbeStatus = "request_error"
	SupplierChannelProbeStatusRemoteUnavailable SupplierChannelProbeStatus = "remote_unavailable"
	SupplierChannelProbeStatusNoLocalAccount    SupplierChannelProbeStatus = "no_local_account"
	SupplierChannelProbeStatusProbeFailed       SupplierChannelProbeStatus = "probe_failed"
)

type SupplierChannelCheckSnapshot struct {
	ID                      int64                      `json:"id"`
	SupplierID              int64                      `json:"supplier_id"`
	SupplierGroupID         int64                      `json:"supplier_group_id"`
	SupplierKeyID           int64                      `json:"supplier_key_id,omitempty"`
	SupplierAccountID       int64                      `json:"supplier_account_id,omitempty"`
	LocalSub2APIAccountID   int64                      `json:"local_sub2api_account_id,omitempty"`
	ExternalGroupID         string                     `json:"external_group_id,omitempty"`
	GroupName               string                     `json:"group_name"`
	ProviderFamily          string                     `json:"provider_family"`
	ChannelMonitorID        int64                      `json:"channel_monitor_id,omitempty"`
	ChannelName             string                     `json:"channel_name,omitempty"`
	ChannelProvider         string                     `json:"channel_provider,omitempty"`
	PrimaryModel            string                     `json:"primary_model,omitempty"`
	RemoteStatus            string                     `json:"remote_status"`
	ProbeModel              string                     `json:"probe_model"`
	ProbeStatus             SupplierChannelProbeStatus `json:"probe_status"`
	Recommended             bool                       `json:"recommended"`
	EffectiveRateMultiplier float64                    `json:"effective_rate_multiplier"`
	FirstTokenMS            int64                      `json:"first_token_ms"`
	DurationMS              int64                      `json:"duration_ms"`
	StatusCode              int                        `json:"status_code"`
	ErrorClass              string                     `json:"error_class,omitempty"`
	ErrorMessage            string                     `json:"error_message,omitempty"`
	LocalAccountSchedulable bool                       `json:"local_account_schedulable"`
	CapturedAt              time.Time                  `json:"captured_at"`
	CreatedAt               time.Time                  `json:"created_at"`
}
