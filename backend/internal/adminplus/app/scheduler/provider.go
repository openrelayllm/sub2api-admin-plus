package scheduler

import (
	announcementsapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/announcements"
	balancesapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/balances"
	channelchecksapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/channelchecks"
	healthapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/health"
	ratesapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/rates"
	sessionsapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/sessions"
	suppliergroupsapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/suppliergroups"
	usagecostsapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/usagecosts"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewSQLRepository,
	wire.Bind(new(Repository), new(*SQLRepository)),
	wire.Bind(new(GroupSyncer), new(*suppliergroupsapp.Service)),
	wire.Bind(new(RateSyncer), new(*ratesapp.Service)),
	wire.Bind(new(BalanceSyncer), new(*balancesapp.Service)),
	wire.Bind(new(AnnouncementSyncer), new(*announcementsapp.Service)),
	wire.Bind(new(HealthSyncer), new(*healthapp.Service)),
	wire.Bind(new(UsageCostSyncer), new(*usagecostsapp.Service)),
	wire.Bind(new(ChannelChecker), new(*channelchecksapp.Service)),
	wire.Bind(new(SessionRefresher), new(*sessionsapp.Service)),
	ProvideService,
	ProvideWorker,
)
