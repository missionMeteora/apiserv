package router

import (
	"net/http"
	"strings"
	"testing"
)

func TestJumpRouter(t *testing.T) {
	r := buildMeteoraAPIRouter(t, true)
	for _, m := range meteoraAPI {
		ep := m.url
		req, _ := http.NewRequest("GET", ep, nil)
		r.ServeHTTP(nil, req)
		req, _ = http.NewRequest("OTHER", ep, nil)
		r.ServeHTTP(nil, req)
	}
}

func TestJumpRouterStar(t *testing.T) {
	r := New(nil)
	fn := func(_ http.ResponseWriter, req *http.Request, p Params) {}
	r.GET("/home", nil)
	r.GET("/home/*path", fn)
	if h, p := r.Match("GET", "/home"); h != nil || len(p) != 0 {
		t.Fatalf("expected a 0 match, got %v %v", h, len(p))
	}
	if h, p := r.Match("GET", "/home/file"); h == nil || len(p) != 1 || p.Get("path") != "file" {
		t.Fatalf("expected a 1 match, got %v %v", h, p)
	}
	if h, p := r.Match("GET", "/home/file/file2/report.json"); h == nil || len(p) != 1 || p.Get("path") != "file/file2/report.json" {
		t.Fatalf("expected a 1 match, got %v %v", h, p)
	}
}

func BenchmarkJumpRouter5Params(b *testing.B) {
	req, _ := http.NewRequest("GET", "/campaignReport/:id/:cid/:start-date/:end-date/:filename", nil)
	r := buildMeteoraAPIRouter(b, false)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.ServeHTTP(nil, req)
		}
	})
}

func BenchmarkJumpRouterStatic(b *testing.B) {
	req, _ := http.NewRequest("GET", "/dashboard", nil)
	r := buildMeteoraAPIRouter(b, false)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.ServeHTTP(nil, req)
		}
	})
}

type errorer interface {
	Fatalf(fmt string, args ...interface{})
	Logf(fmt string, args ...interface{})
}

func buildMeteoraAPIRouter(l errorer, print bool) (r *Router) {
	r = New(nil)
	for _, m := range meteoraAPI {
		ep := m.url
		cnt := strings.Count(ep, ":")
		fn := func(_ http.ResponseWriter, req *http.Request, p Params) {
			if ep != req.URL.EscapedPath() {
				l.Fatalf("urls don't match, expected %s, got %s", ep, req.URL.EscapedPath())
			}
			if cnt != len(p) {
				l.Fatalf("{%q: %q} expected %d params, got %d", ep, p, cnt, len(p))
			}
			if print {
				l.Logf("[%s] %s %q", req.Method, ep, p)
			}
		}
		r.GET(ep, fn)
		r.AddRoute("OTHER", ep, fn)
	}
	r.NotFoundHandler = func(_ http.ResponseWriter, req *http.Request, _ Params) {
		panic(req.URL.String())
	}
	return
}

var meteoraAPI = [...]struct{ url string }{
	// conflicts
	{
		"/dashboard/home",
	},
	{
		"/campaignTemplate/:id",
	},
	{
		"/topN/:resource/:n/:id",
	},
	{
		"/staffPick/:resource/:n/:id",
	},
	{
		"/agencies/list",
	},

	// end conflicts
	{
		"/dashboard",
	},
	{
		"/dashboard/:id",
	},
	{
		"/billing/:id",
	},
	{
		"/billingAdvanced/:id",
	},
	{
		"/billingHistory/:id",
	},
	{
		"/balances/:id",
	},
	{
		"/segmentsAll",
	},
	{
		"/advertisersAll",
	},
	{
		"/campaignsAll",
	},
	{
		"/storeList",
	},
	{
		"/campaignBudgets",
	},
	{
		"/biddingAgentList",
	},
	{
		"/segments/:id",
	},
	{
		"/adGroups/:id",
	},
	{
		"/adGroupsList/:id",
	},
	{
		"/segmentsList/:id",
	},
	{
		"/ads/:id",
	},
	{
		"/adsList/:id",
	},
	{
		"/serveAds/:id",
	},
	{
		"/campaigns/:id",
	},
	{
		"/campaignsList/:id",
	},
	{
		"/frequencyCapList",
	},
	{
		"/brokenCampaignsList",
	},
	{
		"/campaignsBySegmentID/:id",
	},
	{
		"/campaignsBySegmentIDs",
	},
	{
		"/profiles/:id",
	},
	{
		"/productFeed/:id",
	},
	{
		"/images/:id",
	},
	{
		"/images/:id/:type",
	},
	{
		"/reporting/:id",
	},
	{
		"/reporting/:id/:period",
	},
	{
		"/reporting/:id/:period/:offset",
	},
	{
		"/adminReporting",
	},
	{
		"/advertisers/:id",
	},
	{
		"/advertiserList/:id",
	},
	{
		"/agencies/:id",
	},
	{
		"/agenciesList",
	},
	{
		"/managersList",
	},
	{
		"/managers/:id",
	},
	{
		"/developersList",
	},
	{
		"/users/:id",
	},
	{
		"/userID",
	},
	{
		"/loginEmail",
	},
	{
		"/logins/:id",
	},
	{
		"/tokens/:id",
	},
	{
		"/changePassword/:id",
	},
	{
		"/signUp/advertiser",
	},
	{
		"/signUp/advertiser/:id",
	},
	{
		"/signUp/agency",
	},
	{
		"/signUp/manager",
	},
	{
		"/signUp/demo/:id",
	},
	{
		"/signUp/developer",
	},
	{
		"/signIn",
	},
	{
		"/logout",
	},
	{
		"/adminMessage",
	},
	{
		"/zipSearch/:zipcode",
	},
	{
		"/citySearch/:cityName",
	},
	{
		"/eventSearch/:kw/:period/:when",
	},
	{
		"/collegeSearch/:kw",
	},
	{
		"/contentCategoriesSearch/:categoryName",
	},
	{
		"/pendingTransfers",
	},
	{
		"/checkPendingTransfers",
	},
	{
		"/campaignValues",
	},
	{
		"/campaignValues/:id",
	},
	{
		"/campaignsSpent/:id/:period",
	},
	{
		"/agencyCampaignValuesList/:id",
	},
	{
		"/agencyCampaignsSpentList/:id/:period",
	},
	{
		"/prosCampaignValues",
	},
	{
		"/activeAdvertisers",
	},
	{
		"/advertiserAgency/:id",
	},
	{
		"/advertisersWithoutAgency",
	},
	{
		"/setUserToAdvertiser",
	},
	{
		"/statsReport/:id/:period/:filename",
	},
	{
		"/agencyReport/:id/:start-date/:end-date/:filename",
	},
	{
		"/campaignReport/:id/:cid/:start-date/:end-date/:filename",
	},
	{
		"/salesRepReport/:date.csv",
	},
	{
		"/weeklyTotalsReport.csv",
	},
	{
		"/weeklyTotalsReport/:period/report.csv",
	},
	{
		"/totalsReport/:period/report.csv",
	},
	{
		"/statsReportRange/:id/:start-date/:end-date/:filename",
	},
	{
		"/cachedExchangeData/:period",
	},
	{
		"/conversionsUpdate/:period",
	},
	{
		"/refreshCampaignFunding/:id",
	},
	{
		"/version",
	},
	{
		"/weatherMap/:cond/:timeframe",
	},
	{
		"/resetPassword",
	},
	{
		"/resetPassword/:id/:token",
	},
	{
		"/rtb/:endpoint/:id",
	},
	{
		"/rtb-totals/:endpoint/:id/:period",
	},
	{
		"/notifications/:limit",
	},
	{
		"/fbSegments/:id/:segID",
	},
	{
		"/fbSegments/:id",
	},
	{
		"/fbSegmentsList/:id",
	},
	{
		"/fbAds/:id/:adID",
	},
	{
		"/fbAds/:id",
	},
	{
		"/fbAdGroupsList/:id",
	},
	{
		"/fbAdsList/:id",
	},
	{
		"/integrations/:id",
	},
	{
		"/staMedia/ads/:id",
	},
	{
		"/domainLists/:id",
	},
	{
		"/domainList/:id",
	},
	{
		"/domainList/:id/:listID",
	},
	{
		"/allDomainLists",
	},
	{
		"/revisions/feesHistory/:id",
	},
	{
		"/reportSchedules/:id",
	},
	{
		"/reportScheduleList/:id",
	},
	{
		"/reportScheduleListALL",
	},
	{
		"/screenshot/:id",
	},
	{
		"/health",
	},
	{
		"/search/:type",
	},
	{
		"/campaignTemplate/:tid/:id",
	},
	{
		"/campaignTemplates/list/byId/:id",
	},
	{
		"/media/byId/:type/:id",
	},
	{
		"/appMedia/byId/:type/:id",
	},
	{
		"/media/byUser/:type/:id",
	},
	{
		"/appMedia/byUser/:type/:id",
	},
	{
		"/pages/agency/:id",
	},
	{
		"/pages/template/:id",
	},
	{
		"/pages/application/:id",
	},
	{
		"/like/:resource/:id",
	},
	{
		"/topN/:resource/:id",
	},
	{
		"/staffPick/:resource/:id",
	},
	{
		"/apps/byId/:id",
	},
	{
		"/apps/byUid/:id",
	},
	{
		"/apps/query/:cat/:cost/:name/:author",
	},
	{
		"/resources/:resource",
	},
	{
		"/campaignTemplates/list",
	},
	{
		"/apps/list",
	},
	{
		"/apps/list/short/:id",
	},
	{
		"/campaign/app/byId/:id",
	},
	{
		"/addApps/campaignDraft/byId/:id",
	},
	{
		"/campaign/app/byKey/:appKey/:id",
	},
	{
		"/adCache",
	},
	{
		"/campaignDrafts/:id",
	},
	{
		"/campaignDraft/:id",
	},
	{
		"/ioState/:id",
	},
	{
		"/ioState/:id/:state",
	},
	{
		"/reportQueue/:id",
	},
	{
		"/reportDownload/:id/:rid/:filename",
	},
	{
		"/campaignForecast/:id",
	},
	{
		"/campaignDraftForecast/:id",
	},
}
