package collectdomain

import "testing"

func TestDetectPlatform(t *testing.T) {
	cases := []struct {
		url  string
		want PlatformID
		ok   bool
	}{
		{"https://www.amazon.de/dp/B00TEST123", PlatformAmazonDE, true},
		{"https://amazon.de/gp/product/B00TEST123", PlatformAmazonDE, true},
		{"https://detail.1688.com/offer/1.html", "", false},
		{"https://www.aliexpress.com/item/1.html", "", false},
		{"https://item.taobao.com/item.htm?id=1", "", false},
		{"https://example.com/product/1", "", false},
		{"not-a-url", "", false},
	}
	for _, tc := range cases {
		host := HostnameFromURL(tc.url)
		got, ok := DetectPlatform(host)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("url=%q host=%q got=%q ok=%v want=%q ok=%v", tc.url, host, got, ok, tc.want, tc.ok)
		}
	}
}
