package sub2api

import (
	"context"
	"testing"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/stretchr/testify/require"
)

type stubUsageRepository struct {
	filter UsageFilter
}

func (r *stubUsageRepository) ListLocalUsageLines(_ context.Context, filter UsageFilter) ([]*adminplusdomain.LocalUsageLine, error) {
	r.filter = filter
	return []*adminplusdomain.LocalUsageLine{}, nil
}

func (r *stubUsageRepository) ListLocalUsageSummaries(_ context.Context, filter UsageFilter) ([]*adminplusdomain.LocalUsageSummary, error) {
	r.filter = filter
	return []*adminplusdomain.LocalUsageSummary{}, nil
}

func TestServiceListLocalUsageLinesDefaultsToLast24Hours(t *testing.T) {
	repo := &stubUsageRepository{}
	svc := NewService(repo)
	now := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	_, err := svc.ListLocalUsageLines(context.Background(), UsageFilter{})

	require.NoError(t, err)
	require.Equal(t, now.Add(-24*time.Hour), repo.filter.From)
	require.Equal(t, now, repo.filter.To)
	require.Equal(t, 200, repo.filter.Limit)
}

func TestServiceRejectsTooLargeUsageRange(t *testing.T) {
	svc := NewService(&stubUsageRepository{})
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(32 * 24 * time.Hour)

	_, err := svc.ListLocalUsageSummaries(context.Background(), UsageFilter{From: from, To: to})

	require.Error(t, err)
	require.Contains(t, err.Error(), "LOCAL_USAGE_TIME_RANGE_TOO_LARGE")
}
