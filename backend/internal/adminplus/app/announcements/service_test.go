package announcements

import (
	"context"
	"net/http"
	"testing"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/ports"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestServiceRecordAnnouncementRecommendsRechargeForEmptySupplier(t *testing.T) {
	repo := newFakeAnnouncementRepository()
	notifier := &fakeAnnouncementNotifier{}
	svc := NewServiceWithNotifier(repo, notifier)
	bonus := 20.0

	event, err := svc.RecordAnnouncement(context.Background(), RecordAnnouncementInput{
		SupplierID:       7,
		Type:             adminplusdomain.AnnouncementTypeRechargeBonus,
		Title:            "June recharge bonus",
		MinRechargeCents: 10000,
		BonusPercent:     &bonus,
		RuntimeStatus:    adminplusdomain.SupplierRuntimeStatusMonitorOnly,
		BalanceCents:     0,
		Currency:         "usd",
	})

	require.NoError(t, err)
	require.Equal(t, adminplusdomain.AnnouncementRecommendationRechargeToUnlock, event.Recommendation)
	require.False(t, event.SwitchEligible)
	require.Equal(t, "USD", event.Currency)
	require.Equal(t, adminplusdomain.AnnouncementStatusOpen, event.Status)
	require.Len(t, notifier.events, 1)
	require.Equal(t, event.ID, notifier.events[0].ID)
}

func TestServiceRecordAnnouncementMarksSwitchCandidateWhenBalanceIsUsable(t *testing.T) {
	repo := newFakeAnnouncementRepository()
	svc := NewService(repo)
	discount := 15.0

	event, err := svc.RecordAnnouncement(context.Background(), RecordAnnouncementInput{
		SupplierID:      7,
		Type:            adminplusdomain.AnnouncementTypeRateDiscount,
		Title:           "Lower GPT rate",
		DiscountPercent: &discount,
		RuntimeStatus:   adminplusdomain.SupplierRuntimeStatusCandidate,
		BalanceCents:    3000,
		Currency:        "USD",
	})

	require.NoError(t, err)
	require.Equal(t, adminplusdomain.AnnouncementRecommendationSwitchCandidate, event.Recommendation)
	require.True(t, event.SwitchEligible)
}

func TestServiceRecordAnnouncementDisabledSupplierIsInformational(t *testing.T) {
	repo := newFakeAnnouncementRepository()
	svc := NewService(repo)

	event, err := svc.RecordAnnouncement(context.Background(), RecordAnnouncementInput{
		SupplierID:    7,
		Type:          adminplusdomain.AnnouncementTypeLimitedOffer,
		Title:         "Paused supplier campaign",
		RuntimeStatus: adminplusdomain.SupplierRuntimeStatusDisabled,
		BalanceCents:  5000,
		Currency:      "USD",
	})

	require.NoError(t, err)
	require.Equal(t, adminplusdomain.AnnouncementRecommendationInformational, event.Recommendation)
	require.False(t, event.SwitchEligible)
}

func TestServiceRecordAnnouncementNoticeIsInformationalEvenWhenBalanceIsUsable(t *testing.T) {
	repo := newFakeAnnouncementRepository()
	svc := NewService(repo)

	event, err := svc.RecordAnnouncement(context.Background(), RecordAnnouncementInput{
		SupplierID:    7,
		Type:          adminplusdomain.AnnouncementTypeNotice,
		Title:         "Dashboard notice",
		RuntimeStatus: adminplusdomain.SupplierRuntimeStatusCandidate,
		BalanceCents:  3000,
		Currency:      "USD",
	})

	require.NoError(t, err)
	require.Equal(t, adminplusdomain.AnnouncementRecommendationInformational, event.Recommendation)
	require.False(t, event.SwitchEligible)
}

func TestServiceRecordAnnouncementValidatesInput(t *testing.T) {
	svc := NewService(newFakeAnnouncementRepository())
	discount := 101.0

	_, err := svc.RecordAnnouncement(context.Background(), RecordAnnouncementInput{
		SupplierID:      7,
		Type:            adminplusdomain.AnnouncementTypeRateDiscount,
		Title:           "Bad discount",
		DiscountPercent: &discount,
	})

	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "ANNOUNCEMENT_DISCOUNT_PERCENT_INVALID", infraerrors.Reason(err))
}

func TestServiceRecordAnnouncementValidatesTimeRange(t *testing.T) {
	svc := NewService(newFakeAnnouncementRepository())
	start := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	end := start.Add(-time.Minute)

	_, err := svc.RecordAnnouncement(context.Background(), RecordAnnouncementInput{
		SupplierID: 7,
		Type:       adminplusdomain.AnnouncementTypePackageDeal,
		Title:      "Invalid range",
		StartsAt:   &start,
		EndsAt:     &end,
	})

	require.Error(t, err)
	require.Equal(t, "ANNOUNCEMENT_TIME_RANGE_INVALID", infraerrors.Reason(err))
}

func TestServiceSyncFromSessionReadsProviderAnnouncements(t *testing.T) {
	repo := newFakeAnnouncementRepository()
	capturedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	bonus := 20.0
	session := &fakeAnnouncementSessionReader{
		input: ports.SessionProbeInput{
			SupplierID: 7,
			Origin:     "https://relay.example.com",
			APIBaseURL: "https://relay.example.com/api/v1",
			Bundle:     map[string]any{"access_token": "browser-token"},
		},
	}
	reader := &fakeAnnouncementReader{
		result: &ports.ReadAnnouncementsResult{
			SupplierID: 7,
			SystemType: "sub2api",
			Origin:     "https://relay.example.com",
			APIBaseURL: "https://relay.example.com/api/v1",
			CapturedAt: capturedAt,
			Announcements: []ports.ProviderAnnouncement{
				{
					Type:             adminplusdomain.AnnouncementTypeRechargeBonus,
					Title:            "June recharge bonus",
					Currency:         "usd",
					MinRechargeCents: 10000,
					BonusPercent:     &bonus,
					RuntimeStatus:    adminplusdomain.SupplierRuntimeStatusCandidate,
					BalanceCents:     5000,
					RawPayload:       map[string]any{"id": "promo-1"},
				},
			},
		},
	}
	svc := NewServiceWithDependencies(repo, nil, session, reader)

	result, err := svc.SyncFromSession(context.Background(), SyncFromSessionInput{SupplierID: 7})

	require.NoError(t, err)
	require.Equal(t, int64(7), session.seenSupplierID)
	require.Equal(t, int64(7), reader.seen.SupplierID)
	require.Equal(t, "sub2api", result.SystemType)
	require.Equal(t, capturedAt, result.SyncedAt)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Events, 1)
	require.Equal(t, "provider_session", result.Events[0].Source)
	require.Equal(t, adminplusdomain.AnnouncementTypeRechargeBonus, result.Events[0].Type)
	require.Equal(t, int64(10000), result.Events[0].MinRechargeCents)
	require.Equal(t, "USD", result.Events[0].Currency)
}

func TestServiceSyncFromSessionPropagatesProviderError(t *testing.T) {
	session := &fakeAnnouncementSessionReader{}
	reader := &fakeAnnouncementReader{err: infraerrors.New(http.StatusConflict, "SUPPLIER_ANNOUNCEMENT_CAPABILITY_MISSING", "supplier announcements are unavailable")}
	svc := NewServiceWithDependencies(newFakeAnnouncementRepository(), nil, session, reader)

	_, err := svc.SyncFromSession(context.Background(), SyncFromSessionInput{SupplierID: 7})

	require.Error(t, err)
	require.Equal(t, "SUPPLIER_ANNOUNCEMENT_CAPABILITY_MISSING", infraerrors.Reason(err))
}

type fakeAnnouncementRepository struct {
	nextEventID int64
	events      []*adminplusdomain.AnnouncementEvent
}

type fakeAnnouncementNotifier struct {
	events []*adminplusdomain.AnnouncementEvent
}

type fakeAnnouncementSessionReader struct {
	input          ports.SessionProbeInput
	seenSupplierID int64
}

func (r *fakeAnnouncementSessionReader) DecryptedProbeInput(_ context.Context, supplierID int64) (ports.SessionProbeInput, error) {
	r.seenSupplierID = supplierID
	if r.input.SupplierID == 0 {
		r.input.SupplierID = supplierID
	}
	return r.input, nil
}

type fakeAnnouncementReader struct {
	result *ports.ReadAnnouncementsResult
	err    error
	seen   ports.SessionProbeInput
}

func (r *fakeAnnouncementReader) ReadAnnouncements(_ context.Context, in ports.SessionProbeInput) (*ports.ReadAnnouncementsResult, error) {
	r.seen = in
	if r.err != nil {
		return nil, r.err
	}
	return r.result, nil
}

func (n *fakeAnnouncementNotifier) NotifyAnnouncement(_ context.Context, event *adminplusdomain.AnnouncementEvent) error {
	n.events = append(n.events, cloneAnnouncementEvent(event))
	return nil
}

func newFakeAnnouncementRepository() *fakeAnnouncementRepository {
	return &fakeAnnouncementRepository{nextEventID: 1}
}

func (r *fakeAnnouncementRepository) CreateEvent(_ context.Context, event *adminplusdomain.AnnouncementEvent) (*adminplusdomain.AnnouncementEvent, error) {
	cp := cloneAnnouncementEvent(event)
	cp.ID = r.nextEventID
	r.nextEventID++
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = cp.CapturedAt
	}
	r.events = append(r.events, cp)
	return cloneAnnouncementEvent(cp), nil
}

func (r *fakeAnnouncementRepository) ListEvents(_ context.Context, _ EventFilter) ([]*adminplusdomain.AnnouncementEvent, error) {
	items := make([]*adminplusdomain.AnnouncementEvent, 0, len(r.events))
	for _, item := range r.events {
		items = append(items, cloneAnnouncementEvent(item))
	}
	return items, nil
}

func (r *fakeAnnouncementRepository) UpdateEventStatus(_ context.Context, id int64, status adminplusdomain.AnnouncementStatus) (*adminplusdomain.AnnouncementEvent, error) {
	for _, event := range r.events {
		if event.ID == id {
			event.Status = status
			return cloneAnnouncementEvent(event), nil
		}
	}
	return nil, nil
}

func cloneAnnouncementEvent(in *adminplusdomain.AnnouncementEvent) *adminplusdomain.AnnouncementEvent {
	if in == nil {
		return nil
	}
	out := *in
	if in.BonusPercent != nil {
		v := *in.BonusPercent
		out.BonusPercent = &v
	}
	if in.DiscountPercent != nil {
		v := *in.DiscountPercent
		out.DiscountPercent = &v
	}
	if in.StartsAt != nil {
		t := *in.StartsAt
		out.StartsAt = &t
	}
	if in.EndsAt != nil {
		t := *in.EndsAt
		out.EndsAt = &t
	}
	if in.AcknowledgedAt != nil {
		t := *in.AcknowledgedAt
		out.AcknowledgedAt = &t
	}
	if in.RawPayload != nil {
		out.RawPayload = make(map[string]any, len(in.RawPayload))
		for k, v := range in.RawPayload {
			out.RawPayload[k] = v
		}
	}
	return &out
}
