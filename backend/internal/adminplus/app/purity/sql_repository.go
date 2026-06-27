package purity

import (
	"context"
	"database/sql"
	"encoding/json"
)

type SQLRepository struct {
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) SavePublicReport(ctx context.Context, record PublicReportRecord) error {
	if r == nil || r.db == nil || record.Report == nil {
		return nil
	}
	checksJSON, err := json.Marshal(record.Report.Checks)
	if err != nil {
		return err
	}
	metricsJSON, err := json.Marshal(record.Report.Metrics)
	if err != nil {
		return err
	}
	summaryJSON, err := json.Marshal(record.PublicSummaryJSON)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO admin_plus_purity_public_reports (
			request_hash, provider, api_base_host, score, verdict,
			checks_json, metrics_json, public_summary_json, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		record.RequestHash,
		record.Provider,
		record.APIBaseHost,
		record.Report.Score,
		record.Report.Verdict,
		checksJSON,
		metricsJSON,
		summaryJSON,
		record.Report.CheckedAt,
	)
	return err
}
