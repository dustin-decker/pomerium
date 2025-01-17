package proxy // import "github.com/pomerium/pomerium/proxy"

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/pomerium/pomerium/internal/config"
)

var fixedDate = time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

func newTestOptions(t *testing.T) *config.Options {
	opts, err := config.NewOptions("https://authenticate.example", "https://authorize.example")
	if err != nil {
		t.Fatal(err)
	}
	opts.CookieSecret = "OromP1gurwGWjQPYb1nNgSxtbVB5NnLzX6z5WOKr0Yw="
	return opts
}

func TestNewReverseProxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		hostname, _, _ := net.SplitHostPort(r.Host)
		w.Write([]byte(hostname))
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	backendHostname, backendPort, _ := net.SplitHostPort(backendURL.Host)
	backendHost := net.JoinHostPort(backendHostname, backendPort)
	proxyURL, _ := url.Parse(backendURL.Scheme + "://" + backendHost + "/")

	proxyHandler := NewReverseProxy(proxyURL)
	frontend := httptest.NewServer(proxyHandler)
	defer frontend.Close()

	getReq, _ := http.NewRequest("GET", frontend.URL, nil)
	res, _ := http.DefaultClient.Do(getReq)
	bodyBytes, _ := ioutil.ReadAll(res.Body)
	if g, e := string(bodyBytes), backendHostname; g != e {
		t.Errorf("got body %q; expected %q", g, e)
	}
}

func TestNewReverseProxyHandler(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		hostname, _, _ := net.SplitHostPort(r.Host)
		w.Write([]byte(hostname))
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	backendHostname, backendPort, _ := net.SplitHostPort(backendURL.Host)
	backendHost := net.JoinHostPort(backendHostname, backendPort)
	proxyURL, _ := url.Parse(backendURL.Scheme + "://" + backendHost + "/")
	proxyHandler := NewReverseProxy(proxyURL)
	opts := newTestOptions(t)
	opts.SigningKey = "LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSU0zbXBaSVdYQ1g5eUVneFU2czU3Q2J0YlVOREJTQ0VBdFFGNWZVV0hwY1FvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFaFBRditMQUNQVk5tQlRLMHhTVHpicEVQa1JyazFlVXQxQk9hMzJTRWZVUHpOaTRJV2VaLwpLS0lUdDJxMUlxcFYyS01TYlZEeXI5aWp2L1hoOThpeUV3PT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo="
	testPolicy := config.Policy{From: "https://corp.example.com", To: "https://example.com", UpstreamTimeout: 1 * time.Second}
	if err := testPolicy.Validate(); err != nil {
		t.Fatal(err)
	}
	p, err := New(*opts)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := p.newReverseProxyHandler(proxyHandler, &testPolicy)
	if err != nil {
		t.Fatal(err)
	}

	frontend := httptest.NewServer(handle)

	defer frontend.Close()

	getReq, _ := http.NewRequest("GET", frontend.URL, nil)

	res, _ := http.DefaultClient.Do(getReq)
	bodyBytes, _ := ioutil.ReadAll(res.Body)
	if g, e := string(bodyBytes), backendHostname; g != e {
		t.Errorf("got body %q; expected %q", g, e)
	}
}

func testOptions(t *testing.T) config.Options {
	authenticateService, _ := url.Parse("https://authenticate.corp.beyondperimeter.com")
	authorizeService, _ := url.Parse("https://authorize.corp.beyondperimeter.com")

	opts := newTestOptions(t)
	testPolicy := config.Policy{From: "https://corp.example.example", To: "https://example.example"}
	opts.Policies = []config.Policy{testPolicy}
	opts.AuthenticateURL = authenticateService
	opts.AuthorizeURL = authorizeService
	opts.SharedKey = "80ldlrU2d7w+wVpKNfevk6fmb8otEx6CqOfshj2LwhQ="
	opts.CookieSecret = "OromP1gurwGWjQPYb1nNgSxtbVB5NnLzX6z5WOKr0Yw="
	opts.CookieName = "pomerium"
	err := opts.Validate()
	if err != nil {
		t.Fatal(err)
	}
	return *opts
}

func testOptionsTestServer(t *testing.T, uri string) config.Options {
	authenticateService, _ := url.Parse("https://authenticate.corp.beyondperimeter.com")
	authorizeService, _ := url.Parse("https://authorize.corp.beyondperimeter.com")
	testPolicy := config.Policy{
		From: "https://httpbin.corp.example",
		To:   uri,
	}
	if err := testPolicy.Validate(); err != nil {
		t.Fatal(err)
	}
	opts := newTestOptions(t)
	opts.Policies = []config.Policy{testPolicy}
	opts.AuthenticateURL = authenticateService
	opts.AuthorizeURL = authorizeService
	opts.SharedKey = "80ldlrU2d7w+wVpKNfevk6fmb8otEx6CqOfshj2LwhQ="
	opts.CookieSecret = "OromP1gurwGWjQPYb1nNgSxtbVB5NnLzX6z5WOKr0Yw="
	opts.CookieName = "pomerium"
	return *opts
}

func testOptionsWithCORS(t *testing.T, uri string) config.Options {
	testPolicy := config.Policy{
		From:               "https://httpbin.corp.example",
		To:                 uri,
		CORSAllowPreflight: true,
	}
	if err := testPolicy.Validate(); err != nil {
		t.Fatal(err)
	}
	opts := testOptionsTestServer(t, uri)
	opts.Policies = []config.Policy{testPolicy}
	return opts
}

func testOptionsWithPublicAccess(t *testing.T, uri string) config.Options {
	testPolicy := config.Policy{
		From:                             "https://httpbin.corp.example",
		To:                               uri,
		AllowPublicUnauthenticatedAccess: true,
	}
	if err := testPolicy.Validate(); err != nil {
		t.Fatal(err)
	}
	opts := testOptions(t)
	opts.Policies = []config.Policy{testPolicy}
	return opts
}

func testOptionsWithEmptyPolicies(t *testing.T, uri string) config.Options {
	opts := testOptionsTestServer(t, uri)
	opts.Policies = []config.Policy{}
	return opts
}

func TestOptions_Validate(t *testing.T) {
	good := testOptions(t)
	badAuthURL := testOptions(t)
	badAuthURL.AuthenticateURL = nil
	authurl, _ := url.Parse("authenticate.corp.beyondperimeter.com")
	authenticateBadScheme := testOptions(t)
	authenticateBadScheme.AuthenticateURL = authurl
	authenticateInternalBadScheme := testOptions(t)
	authenticateInternalBadScheme.AuthenticateInternalAddr = authurl

	authorizeBadSCheme := testOptions(t)
	authorizeBadSCheme.AuthorizeURL = authurl
	authorizeNil := testOptions(t)
	authorizeNil.AuthorizeURL = nil
	emptyCookieSecret := testOptions(t)
	emptyCookieSecret.CookieSecret = ""
	invalidCookieSecret := testOptions(t)
	invalidCookieSecret.CookieSecret = "OromP1gurwGWjQPYb1nNgSxtbVB5NnLzX6z5WOKr0Yw^"
	shortCookieLength := testOptions(t)
	shortCookieLength.CookieSecret = "gN3xnvfsAwfCXxnJorGLKUG4l2wC8sS8nfLMhcStPg=="
	invalidSignKey := testOptions(t)
	invalidSignKey.SigningKey = "OromP1gurwGWjQPYb1nNgSxtbVB5NnLzX6z5WOKr0Yw^"
	badSharedKey := testOptions(t)
	badSharedKey.SharedKey = ""
	sharedKeyBadBas64 := testOptions(t)
	sharedKeyBadBas64.SharedKey = "%(*@389"
	missingPolicy := testOptions(t)
	missingPolicy.Policies = []config.Policy{}

	tests := []struct {
		name    string
		o       config.Options
		wantErr bool
	}{
		{"good - minimum options", good, false},
		{"nil options", config.Options{}, true},
		{"authenticate service url", badAuthURL, true},
		{"authenticate service url no scheme", authenticateBadScheme, true},
		{"internal authenticate service url no scheme", authenticateInternalBadScheme, true},
		{"authorize service url no scheme", authorizeBadSCheme, true},
		{"authorize service cannot be nil", authorizeNil, true},
		{"no cookie secret", emptyCookieSecret, true},
		{"invalid cookie secret", invalidCookieSecret, true},
		{"short cookie secret", shortCookieLength, true},
		{"no shared secret", badSharedKey, true},
		{"invalid signing key", invalidSignKey, true},
		{"shared secret bad base64", sharedKeyBadBas64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := tt.o
			if err := ValidateOptions(o); (err != nil) != tt.wantErr {
				t.Errorf("Options.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {

	good := testOptions(t)
	shortCookieLength := testOptions(t)
	shortCookieLength.CookieSecret = "gN3xnvfsAwfCXxnJorGLKUG4l2wC8sS8nfLMhcStPg=="
	badRoutedProxy := testOptions(t)
	badRoutedProxy.SigningKey = "YmFkIGtleQo="
	tests := []struct {
		name      string
		opts      config.Options
		wantProxy bool
		numRoutes int
		wantErr   bool
	}{
		{"good", good, true, 1, false},
		{"empty options", config.Options{}, false, 0, true},
		{"short secret/validate sanity check", shortCookieLength, false, 0, true},
		{"invalid ec key, valid base64 though", badRoutedProxy, false, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil && tt.wantProxy == true {
				t.Errorf("New() expected valid proxy struct")
			}
			if got != nil && len(got.routeConfigs) != tt.numRoutes {
				t.Errorf("New() = num routeConfigs \n%+v, want \n%+v \nfrom %+v", got, tt.numRoutes, tt.opts)
			}
		})
	}
}

func Test_UpdateOptions(t *testing.T) {

	good := testOptions(t)
	newPolicy := config.Policy{To: "http://foo.example", From: "http://bar.example"}
	newPolicies := testOptions(t)
	newPolicies.Policies = []config.Policy{
		newPolicy,
	}
	err := newPolicy.Validate()
	if err != nil {
		t.Fatal(err)
	}
	badPolicyURL := config.Policy{To: "http://", From: "http://bar.example"}
	badNewPolicy := testOptions(t)
	badNewPolicy.Policies = []config.Policy{
		badPolicyURL,
	}
	disableTLSPolicy := config.Policy{To: "http://foo.example", From: "http://bar.example", TLSSkipVerify: true}
	disableTLSPolicies := testOptions(t)
	disableTLSPolicies.Policies = []config.Policy{
		disableTLSPolicy,
	}
	customCAPolicies := testOptions(t)
	customCAPolicies.Policies = []config.Policy{
		{To: "http://foo.example", From: "http://bar.example", TLSCustomCA: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURlVENDQW1HZ0F3SUJBZ0lKQUszMmhoR0JIcmFtTUEwR0NTcUdTSWIzRFFFQkN3VUFNR0l4Q3pBSkJnTlYKQkFZVEFsVlRNUk13RVFZRFZRUUlEQXBEWVd4cFptOXlibWxoTVJZd0ZBWURWUVFIREExVFlXNGdSbkpoYm1OcApjMk52TVE4d0RRWURWUVFLREFaQ1lXUlRVMHd4RlRBVEJnTlZCQU1NRENvdVltRmtjM05zTG1OdmJUQWVGdzB4Ck9UQTJNVEl4TlRNeE5UbGFGdzB5TVRBMk1URXhOVE14TlRsYU1HSXhDekFKQmdOVkJBWVRBbFZUTVJNd0VRWUQKVlFRSURBcERZV3hwWm05eWJtbGhNUll3RkFZRFZRUUhEQTFUWVc0Z1JuSmhibU5wYzJOdk1ROHdEUVlEVlFRSwpEQVpDWVdSVFUwd3hGVEFUQmdOVkJBTU1EQ291WW1Ga2MzTnNMbU52YlRDQ0FTSXdEUVlKS29aSWh2Y05BUUVCCkJRQURnZ0VQQURDQ0FRb0NnZ0VCQU1JRTdQaU03Z1RDczloUTFYQll6Sk1ZNjF5b2FFbXdJclg1bFo2eEt5eDIKUG16QVMyQk1UT3F5dE1BUGdMYXcrWExKaGdMNVhFRmRFeXQvY2NSTHZPbVVMbEEzcG1jY1lZejJRVUxGUnRNVwpoeWVmZE9zS25SRlNKaUZ6YklSTWVWWGswV3ZvQmoxSUZWS3RzeWpicXY5dS8yQ1ZTbmRyT2ZFazBURzIzVTNBCnhQeFR1VzFDcmJWOC9xNzFGZEl6U09jaWNjZkNGSHBzS09vM1N0L3FiTFZ5dEg1YW9oYmNhYkZYUk5zS0VxdmUKd3c5SGRGeEJJdUdhK1J1VDVxMGlCaWt1c2JwSkhBd25ucVA3aS9kQWNnQ3NrZ2paakZlRVU0RUZ5K2IrYTFTWQpRQ2VGeHhDN2MzRHZhUmhCQjBWVmZQbGtQejBzdzZsODY1TWFUSWJSeW9VQ0F3RUFBYU15TURBd0NRWURWUjBUCkJBSXdBREFqQmdOVkhSRUVIREFhZ2d3cUxtSmhaSE56YkM1amIyMkNDbUpoWkhOemJDNWpiMjB3RFFZSktvWkkKaHZjTkFRRUxCUUFEZ2dFQkFJaTV1OXc4bWdUNnBwQ2M3eHNHK0E5ZkkzVzR6K3FTS2FwaHI1bHM3MEdCS2JpWQpZTEpVWVpoUGZXcGgxcXRra1UwTEhGUG04M1ZhNTJlSUhyalhUMFZlNEt0TzFuMElBZkl0RmFXNjJDSmdoR1luCmp6dzByeXpnQzRQeUZwTk1uTnRCcm9QdS9iUGdXaU1nTE9OcEVaaGlneDRROHdmMVkvVTlzK3pDQ3hvSmxhS1IKTVhidVE4N1g3bS85VlJueHhvNk56NVpmN09USFRwTk9JNlZqYTBCeGJtSUFVNnlyaXc5VXJnaWJYZk9qM2o2bgpNVExCdWdVVklCMGJCYWFzSnNBTUsrdzRMQU52YXBlWjBET1NuT1I0S0syNEowT3lvRjVmSG1wNTllTTE3SW9GClFxQmh6cG1RVWd1bmVjRVc4QlRxck5wRzc5UjF1K1YrNHd3Y2tQYz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="},
	}

	badCustomCAPolicies := testOptions(t)
	badCustomCAPolicies.Policies = []config.Policy{
		{To: "http://foo.example", From: "http://bar.example", TLSCustomCA: "=@@"},
	}
	goodClientCertPolicies := testOptions(t)
	goodClientCertPolicies.Policies = []config.Policy{
		{To: "http://foo.example", From: "http://bar.example",
			TLSClientKey: "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcGdJQkFBS0NBUUVBNjdLanFtUVlHcTBNVnRBQ1ZwZUNtWG1pbmxRYkRQR0xtc1pBVUV3dWVIUW5ydDNXCnR2cERPbTZBbGFKTVVuVytIdTU1ampva2FsS2VWalRLbWdZR2JxVXpWRG9NYlBEYUhla2x0ZEJUTUdsT1VGc1AKNFVKU0RyTzR6ZE4rem80MjhUWDJQbkcyRkNkVktHeTRQRThpbEhiV0xjcjg3MVlqVjUxZnc4Q0xEWDlQWkpOdQo4NjFDRjdWOWlFSm02c1NmUWxtbmhOOGozK1d6VmJQUU55MVdzUjdpOWU5ajYzRXFLdDIyUTlPWEwrV0FjS3NrCm9JU21DTlZSVUFqVThZUlZjZ1FKQit6UTM0QVFQbHowT3A1Ty9RTi9NZWRqYUY4d0xTK2l2L3p2aVM4Y3FQYngKbzZzTHE2Rk5UbHRrL1FreGVDZUtLVFFlLzNrUFl2UUFkbmw2NVFJREFRQUJBb0lCQVFEQVQ0eXN2V2pSY3pxcgpKcU9SeGFPQTJEY3dXazJML1JXOFhtQWhaRmRTWHV2MkNQbGxhTU1yelBmTG41WUlmaHQzSDNzODZnSEdZc3pnClo4aWJiYWtYNUdFQ0t5N3lRSDZuZ3hFS3pRVGpiampBNWR3S0h0UFhQUnJmamQ1Y2FMczVpcDcxaWxCWEYxU3IKWERIaXUycnFtaC9kVTArWGRMLzNmK2VnVDl6bFQ5YzRyUm84dnZueWNYejFyMnVhRVZ2VExsWHVsb2NpeEVrcgoySjlTMmxveWFUb2tFTnNlMDNpSVdaWnpNNElZcVowOGJOeG9IWCszQXVlWExIUStzRkRKMlhaVVdLSkZHMHUyClp3R2w3YlZpRTFQNXdiQUdtZzJDeDVCN1MrdGQyUEpSV3Frb2VxY3F2RVdCc3RFL1FEcDFpVThCOHpiQXd0Y3IKZHc5TXZ6Q2hBb0dCQVBObzRWMjF6MGp6MWdEb2tlTVN5d3JnL2E4RkJSM2R2Y0xZbWV5VXkybmd3eHVucnFsdwo2U2IrOWdrOGovcXEvc3VQSDhVdzNqSHNKYXdGSnNvTkVqNCt2b1ZSM3UrbE5sTEw5b21rMXBoU0dNdVp0b3huCm5nbUxVbkJUMGI1M3BURkJ5WGsveE5CbElreWdBNlg5T2MreW5na3RqNlRyVnMxUERTdnVJY0s1QW9HQkFQZmoKcEUzR2F6cVFSemx6TjRvTHZmQWJBdktCZ1lPaFNnemxsK0ZLZkhzYWJGNkdudFd1dWVhY1FIWFpYZTA1c2tLcApXN2xYQ3dqQU1iUXI3QmdlazcrOSszZElwL1RnYmZCYnN3Syt6Vng3Z2doeWMrdytXRWExaHByWTZ6YXdxdkFaCkhRU2lMUEd1UGp5WXBQa1E2ZFdEczNmWHJGZ1dlTmd4SkhTZkdaT05Bb0dCQUt5WTF3MUM2U3Y2c3VuTC8vNTcKQ2Z5NTAwaXlqNUZBOWRqZkRDNWt4K1JZMnlDV0ExVGsybjZyVmJ6dzg4czBTeDMrYS9IQW1CM2dMRXBSRU5NKwo5NHVwcENFWEQ3VHdlcGUxUnlrTStKbmp4TzlDSE41c2J2U25sUnBQWlMvZzJRTVhlZ3grK2trbkhXNG1ITkFyCndqMlRrMXBBczFXbkJ0TG9WaGVyY01jSkFvR0JBSTYwSGdJb0Y5SysvRUcyY21LbUg5SDV1dGlnZFU2eHEwK0IKWE0zMWMzUHE0amdJaDZlN3pvbFRxa2d0dWtTMjBraE45dC9ibkI2TmhnK1N1WGVwSXFWZldVUnlMejVwZE9ESgo2V1BMTTYzcDdCR3cwY3RPbU1NYi9VRm5Yd0U4OHlzRlNnOUF6VjdVVUQvU0lDYkI5ZHRVMWh4SHJJK0pZRWdWCkFrZWd6N2lCQW9HQkFJRncrQVFJZUIwM01UL0lCbGswNENQTDJEak0rNDhoVGRRdjgwMDBIQU9mUWJrMEVZUDEKQ2FLR3RDbTg2MXpBZjBzcS81REtZQ0l6OS9HUzNYRk00Qm1rRk9nY1NXVENPNmZmTGdLM3FmQzN4WDJudlpIOQpYZGNKTDQrZndhY0x4c2JJKzhhUWNOVHRtb3pkUjEzQnNmUmIrSGpUL2o3dkdrYlFnSkhCT0syegotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=", TLSClientCert: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUVJVENDQWdtZ0F3SUJBZ0lSQVBqTEJxS1lwcWU0ekhQc0dWdFR6T0F3RFFZSktvWklodmNOQVFFTEJRQXcKRWpFUU1BNEdBMVVFQXhNSFoyOXZaQzFqWVRBZUZ3MHhPVEE0TVRBeE9EUTVOREJhRncweU1UQXlNVEF4TnpRdwpNREZhTUJNeEVUQVBCZ05WQkFNVENIQnZiV1Z5YVhWdE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBCk1JSUJDZ0tDQVFFQTY3S2pxbVFZR3EwTVZ0QUNWcGVDbVhtaW5sUWJEUEdMbXNaQVVFd3VlSFFucnQzV3R2cEQKT202QWxhSk1VblcrSHU1NWpqb2thbEtlVmpUS21nWUdicVV6VkRvTWJQRGFIZWtsdGRCVE1HbE9VRnNQNFVKUwpEck80emROK3pvNDI4VFgyUG5HMkZDZFZLR3k0UEU4aWxIYldMY3I4NzFZalY1MWZ3OENMRFg5UFpKTnU4NjFDCkY3VjlpRUptNnNTZlFsbW5oTjhqMytXelZiUFFOeTFXc1I3aTllOWo2M0VxS3QyMlE5T1hMK1dBY0tza29JU20KQ05WUlVBalU4WVJWY2dRSkIrelEzNEFRUGx6ME9wNU8vUU4vTWVkamFGOHdMUytpdi96dmlTOGNxUGJ4bzZzTApxNkZOVGx0ay9Ra3hlQ2VLS1RRZS8za1BZdlFBZG5sNjVRSURBUUFCbzNFd2J6QU9CZ05WSFE4QkFmOEVCQU1DCkE3Z3dIUVlEVlIwbEJCWXdGQVlJS3dZQkJRVUhBd0VHQ0NzR0FRVUZCd01DTUIwR0ExVWREZ1FXQkJRQ1FYbWIKc0hpcS9UQlZUZVhoQ0dpNjhrVy9DakFmQmdOVkhTTUVHREFXZ0JSNTRKQ3pMRlg0T0RTQ1J0dWNBUGZOdVhWegpuREFOQmdrcWhraUc5dzBCQVFzRkFBT0NBZ0VBcm9XL2trMllleFN5NEhaQXFLNDVZaGQ5ay9QVTFiaDlFK1BRCk5jZFgzTUdEY2NDRUFkc1k4dll3NVE1cnhuMGFzcSt3VGFCcGxoYS9rMi9VVW9IQ1RqUVp1Mk94dEF3UTdPaWIKVE1tMEorU3NWT3d4YnFQTW9rK1RqVE16NFdXaFFUTzVwRmNoZDZXZXNCVHlJNzJ0aG1jcDd1c2NLU2h3YktIegpQY2h1QTQ4SzhPdi96WkxmZnduQVNZb3VCczJjd1ZiRDI3ZXZOMzdoMGFzR1BrR1VXdm1PSDduTHNVeTh3TTdqCkNGL3NwMmJmTC9OYVdNclJnTHZBMGZMS2pwWTQrVEpPbkVxQmxPcCsrbHlJTEZMcC9qMHNybjRNUnlKK0t6UTEKR1RPakVtQ1QvVEFtOS9XSThSL0FlYjcwTjEzTytYNEtaOUJHaDAxTzN3T1Vqd3BZZ3lxSnNoRnNRUG50VmMrSQpKQmF4M2VQU3NicUcwTFkzcHdHUkpRNmMrd1lxdGk2Y0tNTjliYlRkMDhCNUk1N1RRTHhNcUoycTFnWmw1R1VUCmVFZGNWRXltMnZmd0NPd0lrbGNBbThxTm5kZGZKV1FabE5VaHNOVWFBMkVINnlDeXdaZm9aak9hSDEwTXowV20KeTNpZ2NSZFQ3Mi9NR2VkZk93MlV0MVVvRFZmdEcxcysrditUQ1lpNmpUQU05dkZPckJ4UGlOeGFkUENHR2NZZAowakZIc2FWOGFPV1dQQjZBQ1JteHdDVDdRTnRTczM2MlpIOUlFWWR4Q00yMDUrZmluVHhkOUcwSmVRRTd2Kyt6CldoeWo2ZmJBWUIxM2wvN1hkRnpNSW5BOGxpekdrVHB2RHMxeTBCUzlwV3ppYmhqbVFoZGZIejdCZGpGTHVvc2wKZzlNZE5sND0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=",
		},
	}
	goodClientCertPolicies.Validate()
	tests := []struct {
		name            string
		originalOptions config.Options
		updatedOptions  config.Options
		signingKey      string
		host            string
		wantErr         bool
		wantRoute       bool
	}{
		{"good no change", good, good, "", "https://corp.example.example", false, true},
		{"changed", good, newPolicies, "", "https://bar.example", false, true},
		{"changed and missing", good, newPolicies, "", "https://corp.example.example", false, false},
		// todo(bdd): not sure what intent of this test is?
		{"bad signing key", good, newPolicies, "^bad base 64", "https://corp.example.example", true, false},
		{"bad change bad policy url", good, badNewPolicy, "", "https://bar.example", true, false},
		// todo: stand up a test server using self signed certificates
		{"disable tls verification", good, disableTLSPolicies, "", "https://bar.example", false, true},
		{"custom root ca", good, customCAPolicies, "", "https://bar.example", false, true},
		{"bad custom root ca base64", good, badCustomCAPolicies, "", "https://bar.example", true, false},
		{"good client certs", good, goodClientCertPolicies, "", "https://bar.example", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := New(tt.originalOptions)
			if err != nil {
				t.Fatal(err)
			}

			p.signingKey = tt.signingKey
			err = p.UpdateOptions(tt.updatedOptions)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateOptions: err = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			// This is only safe if we actually can load policies
			if err == nil {
				req := httptest.NewRequest("GET", tt.host, nil)
				_, ok := p.router(req)
				if ok != tt.wantRoute {
					t.Errorf("Failed to find route handler")
					return
				}
			}
		})
	}

	// Test nil
	var p *Proxy
	p.UpdateOptions(config.Options{})
}
