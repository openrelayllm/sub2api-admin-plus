package sitediscovery

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
)

func TestIsSiteDiscoveryRegisterURLUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "register url unique constraint",
			err:  &pq.Error{Code: "23505", Constraint: "idx_admin_plus_site_discoveries_register_url"},
			want: true,
		},
		{
			name: "wrapped register url unique constraint",
			err:  fmt.Errorf("wrapped: %w", &pq.Error{Code: "23505", Constraint: "idx_admin_plus_site_discoveries_register_url"}),
			want: true,
		},
		{
			name: "other pq error",
			err:  &pq.Error{Code: "23503", Constraint: "idx_admin_plus_site_discoveries_register_url"},
			want: false,
		},
		{
			name: "plain error",
			err:  errors.New("pq: duplicate key"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSiteDiscoveryRegisterURLUniqueViolation(tt.err); got != tt.want {
				t.Fatalf("isSiteDiscoveryRegisterURLUniqueViolation() = %v, want %v", got, tt.want)
			}
		})
	}
}
