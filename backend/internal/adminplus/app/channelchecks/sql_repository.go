package channelchecks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/lib/pq"
)

type SQLRepository struct {
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) ListCandidates(ctx context.Context, supplierID int64) ([]*Candidate, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			s.id,
			s.name,
			s.type,
			s.runtime_status,
			s.health_status,
			sg.id,
			sg.external_group_id,
			sg.name,
			sg.provider_family,
			sg.effective_rate_multiplier,
			COALESCE(sk.id, 0),
			COALESCE(asa.id, 0),
			COALESCE(a.id, 0),
			COALESCE(a.name, ''),
			COALESCE(a.platform, ''),
			COALESCE(a.type, ''),
			COALESCE(a.status, ''),
			COALESCE(a.schedulable, FALSE),
			COALESCE(array_agg(ag.group_id ORDER BY ag.priority, ag.group_id) FILTER (WHERE ag.group_id IS NOT NULL), ARRAY[]::BIGINT[])
		FROM admin_plus_supplier_groups sg
		INNER JOIN admin_plus_suppliers s ON s.id = sg.supplier_id
		LEFT JOIN admin_plus_supplier_keys sk
			ON sk.supplier_group_id = sg.id
			AND sk.supplier_id = sg.supplier_id
			AND sk.status = 'bound'
		LEFT JOIN admin_plus_supplier_accounts asa
			ON asa.supplier_id = sg.supplier_id
			AND asa.supplier_key_id = sk.id
		LEFT JOIN accounts a
			ON a.id = asa.local_sub2api_account_id
			AND a.deleted_at IS NULL
		LEFT JOIN account_groups ag ON ag.account_id = a.id
		WHERE sg.supplier_id = $1
			AND sg.status = 'active'
			AND sg.effective_rate_multiplier > 0
		GROUP BY s.id, s.name, s.type, s.runtime_status, s.health_status,
			sg.id, sg.external_group_id, sg.name, sg.provider_family, sg.effective_rate_multiplier,
			sk.id, asa.id, a.id, a.name, a.platform, a.type, a.status, a.schedulable
		ORDER BY sg.effective_rate_multiplier ASC, sg.id ASC
	`, supplierID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*Candidate, 0)
	for rows.Next() {
		var item Candidate
		var supplierType, runtimeStatus, healthStatus string
		var groupIDs pq.Int64Array
		if err := rows.Scan(
			&item.SupplierID,
			&item.SupplierName,
			&supplierType,
			&runtimeStatus,
			&healthStatus,
			&item.SupplierGroupID,
			&item.ExternalGroupID,
			&item.GroupName,
			&item.ProviderFamily,
			&item.EffectiveRateMultiplier,
			&item.SupplierKeyID,
			&item.SupplierAccountID,
			&item.LocalSub2APIAccountID,
			&item.LocalAccountName,
			&item.LocalAccountPlatform,
			&item.LocalAccountType,
			&item.LocalAccountStatus,
			&item.LocalAccountSchedulable,
			&groupIDs,
		); err != nil {
			return nil, err
		}
		item.SupplierType = adminplusdomain.SupplierType(supplierType)
		item.SupplierRuntimeStatus = adminplusdomain.SupplierRuntimeStatus(runtimeStatus)
		item.SupplierHealthStatus = adminplusdomain.SupplierHealthStatus(healthStatus)
		item.LocalAccountGroupIDs = append([]int64(nil), groupIDs...)
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *SQLRepository) CreateSnapshot(ctx context.Context, snapshot *adminplusdomain.SupplierChannelCheckSnapshot) (*adminplusdomain.SupplierChannelCheckSnapshot, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	if snapshot == nil {
		return nil, infraerrors.New(http.StatusBadRequest, "SUPPLIER_CHANNEL_CHECK_REQUIRED", "supplier channel check snapshot is required")
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_plus_supplier_channel_check_snapshots (
			supplier_id, supplier_group_id, supplier_key_id, supplier_account_id,
			local_sub2api_account_id, external_group_id, group_name, provider_family,
			channel_monitor_id, channel_name, channel_provider, primary_model,
			remote_status, probe_model, probe_status, recommended,
			effective_rate_multiplier, first_token_ms, duration_ms, status_code,
			error_class, error_message, local_account_schedulable, captured_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		RETURNING `+snapshotColumns()+`
	`,
		snapshot.SupplierID,
		snapshot.SupplierGroupID,
		snapshot.SupplierKeyID,
		snapshot.SupplierAccountID,
		snapshot.LocalSub2APIAccountID,
		snapshot.ExternalGroupID,
		snapshot.GroupName,
		snapshot.ProviderFamily,
		snapshot.ChannelMonitorID,
		snapshot.ChannelName,
		snapshot.ChannelProvider,
		snapshot.PrimaryModel,
		snapshot.RemoteStatus,
		snapshot.ProbeModel,
		string(snapshot.ProbeStatus),
		snapshot.Recommended,
		snapshot.EffectiveRateMultiplier,
		snapshot.FirstTokenMS,
		snapshot.DurationMS,
		snapshot.StatusCode,
		snapshot.ErrorClass,
		snapshot.ErrorMessage,
		snapshot.LocalAccountSchedulable,
		snapshot.CapturedAt,
	)
	return scanSnapshot(row)
}

func (r *SQLRepository) ListLatestSnapshots(ctx context.Context, supplierID int64, limit int) ([]*adminplusdomain.SupplierChannelCheckSnapshot, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+snapshotColumns()+`
		FROM admin_plus_supplier_channel_check_snapshots
		WHERE supplier_id = $1
		ORDER BY captured_at DESC, id DESC
		LIMIT $2
	`, supplierID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanSnapshots(rows)
}

func (r *SQLRepository) ListLatestSnapshotsBySupplierIDs(ctx context.Context, supplierIDs []int64) ([]*adminplusdomain.SupplierChannelCheckSnapshot, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	if len(supplierIDs) == 0 {
		return []*adminplusdomain.SupplierChannelCheckSnapshot{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (supplier_id, supplier_group_id) `+snapshotColumns()+`
			FROM admin_plus_supplier_channel_check_snapshots
			WHERE supplier_id = ANY($1)
			ORDER BY supplier_id, supplier_group_id, captured_at DESC, id DESC
		)
		SELECT `+snapshotColumns()+`
		FROM latest
		ORDER BY supplier_id ASC, effective_rate_multiplier ASC, captured_at DESC, id DESC
	`, pq.Array(supplierIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanSnapshots(rows)
}

func (r *SQLRepository) SetLocalAccountSchedulable(ctx context.Context, localAccountID int64, schedulable bool) error {
	if r == nil || r.db == nil {
		return dbNotConfigured()
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE accounts
		SET schedulable = $2,
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL
	`, localAccountID, schedulable)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return infraerrors.New(http.StatusNotFound, "LOCAL_ACCOUNT_NOT_FOUND", "local Sub2API account not found")
	}
	_ = r.enqueueSchedulerOutbox(ctx, localAccountID)
	return nil
}

func (r *SQLRepository) enqueueSchedulerOutbox(ctx context.Context, accountID int64) error {
	payload, _ := json.Marshal(map[string]any{"source": "admin_plus_channel_check"})
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO scheduler_outbox (event_type, account_id, payload, dedup_key)
		VALUES ('account_changed', $1, $2, $3)
		ON CONFLICT (dedup_key) WHERE dedup_key IS NOT NULL DO NOTHING
	`, accountID, payload, fmt.Sprintf("admin_plus_channel_check:account:%d", accountID))
	return err
}

type snapshotScanner interface {
	Scan(dest ...any) error
}

func scanSnapshots(rows *sql.Rows) ([]*adminplusdomain.SupplierChannelCheckSnapshot, error) {
	items := make([]*adminplusdomain.SupplierChannelCheckSnapshot, 0)
	for rows.Next() {
		item, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func scanSnapshot(scanner snapshotScanner) (*adminplusdomain.SupplierChannelCheckSnapshot, error) {
	var snapshot adminplusdomain.SupplierChannelCheckSnapshot
	var probeStatus string
	if err := scanner.Scan(
		&snapshot.ID,
		&snapshot.SupplierID,
		&snapshot.SupplierGroupID,
		&snapshot.SupplierKeyID,
		&snapshot.SupplierAccountID,
		&snapshot.LocalSub2APIAccountID,
		&snapshot.ExternalGroupID,
		&snapshot.GroupName,
		&snapshot.ProviderFamily,
		&snapshot.ChannelMonitorID,
		&snapshot.ChannelName,
		&snapshot.ChannelProvider,
		&snapshot.PrimaryModel,
		&snapshot.RemoteStatus,
		&snapshot.ProbeModel,
		&probeStatus,
		&snapshot.Recommended,
		&snapshot.EffectiveRateMultiplier,
		&snapshot.FirstTokenMS,
		&snapshot.DurationMS,
		&snapshot.StatusCode,
		&snapshot.ErrorClass,
		&snapshot.ErrorMessage,
		&snapshot.LocalAccountSchedulable,
		&snapshot.CapturedAt,
		&snapshot.CreatedAt,
	); err != nil {
		return nil, err
	}
	snapshot.ProbeStatus = adminplusdomain.SupplierChannelProbeStatus(probeStatus)
	return &snapshot, nil
}

func snapshotColumns() string {
	return strings.Join([]string{
		"id",
		"supplier_id",
		"supplier_group_id",
		"supplier_key_id",
		"supplier_account_id",
		"local_sub2api_account_id",
		"external_group_id",
		"group_name",
		"provider_family",
		"channel_monitor_id",
		"channel_name",
		"channel_provider",
		"primary_model",
		"remote_status",
		"probe_model",
		"probe_status",
		"recommended",
		"effective_rate_multiplier",
		"first_token_ms",
		"duration_ms",
		"status_code",
		"error_class",
		"error_message",
		"local_account_schedulable",
		"captured_at",
		"created_at",
	}, ", ")
}

func dbNotConfigured() error {
	return infraerrors.New(http.StatusInternalServerError, "ADMIN_PLUS_DB_NOT_CONFIGURED", "admin plus database is not configured")
}
