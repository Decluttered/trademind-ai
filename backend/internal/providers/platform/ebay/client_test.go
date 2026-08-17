package ebay

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func validListing() Listing {
	return Listing{SKU: "MB-1", Title: "Title", Description: "Description", CategoryID: "123", Condition: "NEW", Currency: "EUR", PriceCents: 2999, Quantity: 1, ImageURLs: []string{"https://images.example/item.jpg"}, Aspects: map[string][]string{"Marke": {"MindBay"}}, Marketplace: "EBAY_DE", MerchantLocationKey: "de-warehouse", PaymentPolicyID: "pay", ReturnPolicyID: "return", FulfillmentPolicyID: "ship", ManufacturerName: "Maker", ManufacturerAddress: "Street 1, 10115 Berlin", ResponsiblePersonName: "EU Contact", ResponsibleAddress: "Street 2, 10115 Berlin", SafetyInformation: "Keep dry"}
}

func TestCentsDecimal(t *testing.T) {
	require.Equal(t, "29.99", CentsDecimal(2999))
	require.Equal(t, "0.05", CentsDecimal(5))
}

func TestDecimalCents(t *testing.T) {
	for input, expected := range map[string]int64{"0": 0, "12.3": 1230, "12.34": 1234} {
		actual, err := decimalCents(input)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}
	_, err := decimalCents("12.345")
	require.Error(t, err)
}

func TestDryRunMakesNoMutatingRequest(t *testing.T) {
	calls := 0
	client := Client{Config: RuntimeConfig{Environment: "sandbox", APIBaseURL: "https://api.invalid", Timeout: time.Second}, HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("must not be called") })}}
	out, err := client.Publish(context.Background(), "secret-token", "DRY_RUN", validListing())
	require.NoError(t, err)
	require.True(t, out.DryRun)
	require.Zero(t, calls)
	require.Equal(t, 0, out.ResponseArtifact["mutatingCalls"])
}

func TestUnknownCreateOfferResultReconcilesWithoutDuplicateOffer(t *testing.T) {
	var mu sync.Mutex
	counts := map[string]int{}
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		key := request.Method + " " + request.URL.Path
		mu.Lock()
		counts[key]++
		mu.Unlock()
		if request.Method == http.MethodPost && request.URL.Path == "/sell/inventory/v1/offer" {
			return nil, errors.New("timeout after server accepted offer")
		}
		body := `{}`
		status := 200
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/sell/inventory/v1/offer":
			body = `{"offers":[{"offerId":"offer-1"}]}`
		case strings.HasSuffix(request.URL.Path, "/publish"):
			body = `{"listingId":"listing-1"}`
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}, nil
	})
	client := Client{Config: RuntimeConfig{Environment: "sandbox", APIBaseURL: "https://api.sandbox.ebay.test", Timeout: time.Second}, HTTPClient: &http.Client{Transport: transport}}
	out, err := client.Publish(context.Background(), "token", "SIMULATED_CHECKOUT", validListing())
	require.NoError(t, err)
	require.Equal(t, "offer-1", out.OfferID)
	require.Equal(t, "listing-1", out.ListingID)
	require.Equal(t, 1, counts["POST /sell/inventory/v1/offer"])
}

func TestProductionPublishRequiresLiveMode(t *testing.T) {
	client := Client{Config: RuntimeConfig{Environment: "production", APIBaseURL: "https://api.ebay.com"}}
	_, err := client.Publish(context.Background(), "token", "SIMULATED_CHECKOUT", validListing())
	require.ErrorContains(t, err, "AUTOMATION_MODE=LIVE")
}

func TestDryRunOfferUpdateMakesNoRequest(t *testing.T) {
	calls := 0
	client := Client{Config: RuntimeConfig{Environment: "sandbox", APIBaseURL: "https://api.invalid", Timeout: time.Second}, HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("must not be called") })}}
	row, artifact, dryRun, err := client.UpdateOffer(context.Background(), "", "DRY_RUN", "offer-1", "EUR", 3199)
	require.NoError(t, err)
	require.True(t, dryRun)
	require.Zero(t, calls)
	require.Equal(t, int64(3199), row.PriceCents)
	require.Equal(t, "31.99", artifact["price"])
}

func TestOfferUpdateVerifiesTargetPrice(t *testing.T) {
	methods := []string{}
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		methods = append(methods, request.Method)
		body := `{}`
		if request.Method == http.MethodGet {
			body = `{"offerId":"offer-1","availableQuantity":2,"status":"PUBLISHED","pricingSummary":{"price":{"currency":"EUR","value":"31.99"}}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}, nil
	})
	client := Client{Config: RuntimeConfig{Environment: "sandbox", APIBaseURL: "https://api.sandbox.ebay.test", Timeout: time.Second}, HTTPClient: &http.Client{Transport: transport}}
	row, _, dryRun, err := client.UpdateOffer(context.Background(), "token", "SIMULATED_CHECKOUT", "offer-1", "EUR", 3199)
	require.NoError(t, err)
	require.False(t, dryRun)
	require.Equal(t, int64(3199), row.PriceCents)
	require.Equal(t, []string{http.MethodPut, http.MethodGet}, methods)
}

func TestOfferUsesMetadataSafetyIDsInsteadOfFreeTextStatements(t *testing.T) {
	var offer map[string]any
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := `{}`
		if request.Method == http.MethodPost && request.URL.Path == "/sell/inventory/v1/offer" {
			require.NoError(t, json.NewDecoder(request.Body).Decode(&offer))
			body = `{"offerId":"offer-1"}`
		} else if strings.HasSuffix(request.URL.Path, "/publish") {
			body = `{"listingId":"listing-1"}`
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}, nil
	})
	listing := validListing()
	listing.SafetyStatementIDs = []string{"EBPSS102"}
	client := Client{Config: RuntimeConfig{Environment: "sandbox", APIBaseURL: "https://api.sandbox.ebay.test", Timeout: time.Second}, HTTPClient: &http.Client{Transport: transport}}
	_, err := client.Publish(context.Background(), "token", "SIMULATED_CHECKOUT", listing)
	require.NoError(t, err)
	regulatory := offer["regulatory"].(map[string]any)
	productSafety := regulatory["productSafety"].(map[string]any)
	require.Equal(t, []any{"EBPSS102"}, productSafety["statements"])
	require.NotContains(t, productSafety, "statementText")
}

func TestAuditedGPSROverrideMayOmitRegulatoryBlock(t *testing.T) {
	listing := validListing()
	listing.ManufacturerName = ""
	listing.ManufacturerAddress = ""
	listing.ResponsiblePersonName = ""
	listing.ResponsibleAddress = ""
	listing.SafetyInformation = ""
	listing.GPSROverridden = true
	require.NoError(t, validateListing(listing))
}

func TestGetPrivilegesParsesDocumentedFields(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, request.Method)
		require.Equal(t, "/sell/account/v1/privilege", request.URL.Path)
		require.Equal(t, "EBAY_DE", request.Header.Get("X-EBAY-C-MARKETPLACE-ID"))
		body := `{"sellerRegistrationCompleted":true,"sellingLimit":{"amount":{"currency":"EUR","value":"150.00"},"quantity":10}}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}, nil
	})
	client := Client{Config: RuntimeConfig{APIBaseURL: "https://api.sandbox.ebay.test", Marketplace: "EBAY_DE", Timeout: time.Second}, HTTPClient: &http.Client{Transport: transport}}
	out, err := client.GetPrivileges(context.Background(), "user-token")
	require.NoError(t, err)
	require.NotNil(t, out.SellerRegistrationCompleted)
	require.True(t, *out.SellerRegistrationCompleted)
	require.Equal(t, "150.00", out.SellingLimitAmount)
	require.Equal(t, "EUR", out.SellingLimitCurrency)
	require.NotNil(t, out.SellingLimitQuantity)
	require.Equal(t, 10, *out.SellingLimitQuantity)
}

func TestGetPrivilegesMapsAuthFailure(t *testing.T) {
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 403, Body: io.NopCloser(strings.NewReader(`{"errors":[{"message":"Insufficient permissions"}]}`)), Header: make(http.Header)}, nil
	})
	client := Client{Config: RuntimeConfig{APIBaseURL: "https://api.sandbox.ebay.test", Timeout: time.Second}, HTTPClient: &http.Client{Transport: transport}}
	_, err := client.GetPrivileges(context.Background(), "user-token")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, ErrorAuth, apiErr.Class)
}
