package sitecatalog

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
)

func TestBulkPublishSitesDeduplicatesIDsAndReportsSkipped(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	repo := &bulkPublishSitesRepo{updated: 1}
	service := NewService(repo)
	service.now = func() time.Time { return now }

	result, err := service.BulkPublishSites(context.Background(), BulkPublishSitesInput{IDs: []int64{1, 2, 2, 0, -1}})
	if err != nil {
		t.Fatalf("BulkPublishSites returned error: %v", err)
	}
	if !reflect.DeepEqual(repo.input.IDs, []int64{1, 2}) {
		t.Fatalf("expected deduplicated ids [1 2], got %#v", repo.input.IDs)
	}
	if !repo.publishedAt.Equal(now) {
		t.Fatalf("expected published time %s, got %s", now, repo.publishedAt)
	}
	if result.Total != 1 || result.Updated != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestBulkPublishSitesPublishesByFilterWithoutIDs(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	repo := &bulkPublishSitesRepo{updated: 3}
	service := NewService(repo)
	service.now = func() time.Time { return now }

	result, err := service.BulkPublishSites(context.Background(), BulkPublishSitesInput{
		Query:  " site ",
		Status: adminplusdomain.SiteCatalogStatusDraft,
	})
	if err != nil {
		t.Fatalf("BulkPublishSites returned error: %v", err)
	}
	if repo.input.Query != "site" || repo.input.Status != adminplusdomain.SiteCatalogStatusDraft {
		t.Fatalf("unexpected input: %#v", repo.input)
	}
	if result.Total != 3 || result.Updated != 3 || result.Skipped != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestBulkAddDiscoveryCandidatesCanIncludeUnsupported(t *testing.T) {
	repo := &bulkPublishSitesRepo{}
	service := NewService(repo)

	result, err := service.BulkAddDiscoveryCandidates(context.Background(), []*adminplusdomain.SiteDiscoveryItem{
		{
			ID:                   9,
			Name:                 "Unknown Site",
			Host:                 "unknown.example.com",
			ClassificationStatus: adminplusdomain.SiteDiscoveryClassificationUnknown,
		},
	}, BulkAddDiscoveryCandidatesInput{IncludeUnsupported: true}, nil)
	if err != nil {
		t.Fatalf("BulkAddDiscoveryCandidates returned error: %v", err)
	}
	if result.Created != 1 || result.Skipped != 0 || len(repo.addedIDs) != 1 || repo.addedIDs[0] != 9 {
		t.Fatalf("unexpected result=%#v added=%#v", result, repo.addedIDs)
	}
}

func TestBulkAddDiscoveryCandidatesSkipsUnsupportedByDefault(t *testing.T) {
	repo := &bulkPublishSitesRepo{}
	service := NewService(repo)

	result, err := service.BulkAddDiscoveryCandidates(context.Background(), []*adminplusdomain.SiteDiscoveryItem{
		{
			ID:                   9,
			Name:                 "Unknown Site",
			Host:                 "unknown.example.com",
			ClassificationStatus: adminplusdomain.SiteDiscoveryClassificationUnknown,
		},
	}, BulkAddDiscoveryCandidatesInput{}, nil)
	if err != nil {
		t.Fatalf("BulkAddDiscoveryCandidates returned error: %v", err)
	}
	if result.Created != 0 || result.Skipped != 1 || len(repo.addedIDs) != 0 {
		t.Fatalf("unexpected result=%#v added=%#v", result, repo.addedIDs)
	}
}

type bulkPublishSitesRepo struct {
	input       BulkPublishSitesInput
	publishedAt time.Time
	updated     int64
	addedIDs    []int64
}

func (r *bulkPublishSitesRepo) ListSites(context.Context, SiteFilter) ([]*adminplusdomain.SiteCatalogSite, error) {
	return nil, errors.New("not implemented")
}

func (r *bulkPublishSitesRepo) GetSite(context.Context, int64) (*adminplusdomain.SiteCatalogSite, error) {
	return nil, errors.New("not implemented")
}

func (r *bulkPublishSitesRepo) CreateSite(context.Context, *adminplusdomain.SiteCatalogSite) (*adminplusdomain.SiteCatalogSite, error) {
	return nil, errors.New("not implemented")
}

func (r *bulkPublishSitesRepo) BulkPublishSites(_ context.Context, input BulkPublishSitesInput, publishedAt time.Time) (int64, error) {
	r.input = input
	r.input.IDs = append([]int64(nil), input.IDs...)
	r.publishedAt = publishedAt
	return r.updated, nil
}

func (r *bulkPublishSitesRepo) AddDiscoveryCandidate(_ context.Context, candidate *adminplusdomain.SiteDiscoveryItem, _ AddDiscoveryCandidateInput) (*adminplusdomain.SiteCatalogSite, error) {
	r.addedIDs = append(r.addedIDs, candidate.ID)
	return &adminplusdomain.SiteCatalogSite{
		ID:            candidate.ID,
		Slug:          candidate.Host,
		CanonicalHost: candidate.Host,
		Name:          candidate.Name,
	}, nil
}

func (r *bulkPublishSitesRepo) ListCategories(context.Context) ([]*adminplusdomain.SiteCatalogCategory, error) {
	return nil, errors.New("not implemented")
}

func (r *bulkPublishSitesRepo) ListTags(context.Context) ([]*adminplusdomain.SiteCatalogTag, error) {
	return nil, errors.New("not implemented")
}

func (r *bulkPublishSitesRepo) SlugExists(context.Context, string) (bool, error) {
	return false, nil
}
