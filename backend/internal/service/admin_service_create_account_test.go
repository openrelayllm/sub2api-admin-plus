//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminServiceCreateAccountDefaultsSchedulableOff(t *testing.T) {
	repo := &mockAccountRepoForPlatform{}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "imported-account",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeAPIKey,
		Credentials:          map[string]any{"api_key": "sk-test"},
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.NotNil(t, repo.created)
	require.False(t, repo.created.Schedulable)
	require.False(t, account.Schedulable)
}

func TestAdminServiceCreateAccountAllowsExplicitSchedulableOn(t *testing.T) {
	repo := &mockAccountRepoForPlatform{}
	svc := &adminServiceImpl{accountRepo: repo}
	schedulable := true

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "manual-account",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeAPIKey,
		Credentials:          map[string]any{"api_key": "sk-test"},
		Schedulable:          &schedulable,
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.NotNil(t, repo.created)
	require.True(t, repo.created.Schedulable)
	require.True(t, account.Schedulable)
}
