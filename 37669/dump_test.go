package dump

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"
	"time"
)

type loggingRoundTripper struct {
	sub http.RoundTripper
}

func (lrt *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	httputil.DumpRequestOut(req, true)

	return lrt.sub.RoundTrip(req)
}

func NewLoggingRoundTripper(sub http.RoundTripper) http.RoundTripper {
	return &loggingRoundTripper{
		sub: sub,
	}
}

// TestDumpRoundTripRace calls the various Dump functions at the obvious
// locations in implementations of http.RoundTripper and http.Handler.
//
// It is intended to fail if any of the Dump methods introduce a race with
// anything in the HTTP stack.
func TestDumpRoundTripRace(t *testing.T) {
	for i := 0; i < 100; i++ {
				t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
					testDumpRoundTripRace(t, "http", true)
				})
	}
}

func testDumpRoundTripRace(t *testing.T, scheme string, dumpBody bool) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(3000)))
		ioutil.ReadAll(r.Body)
	})

	var srv *httptest.Server
	switch scheme {
	case "http":
		srv = httptest.NewServer(handler)
	case "https":
		srv = httptest.NewTLSServer(handler)
	default:
		t.Fatalf("unexpected scheme %s", scheme)
	}
	defer srv.Close()

	c := http.Client{
		// Transport based on http.DefaultTransport with keep alives
		// disabled for cameras that only support one request per connection
		Transport: NewLoggingRoundTripper(&http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   time.Second,
				DualStack: true,
			}).DialContext,
			DisableKeepAlives:     true,
			TLSHandshakeTimeout:   1 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		}),
		Timeout: 2 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	if rand.Intn(2) > 0 {
		fmt.Printf("Cancelled immediately\n")
		//cancel()
	} else {
		go func() {
			time.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond)
			fmt.Printf("Cancelled\n")
			cancel()
		}()
	}
	data, err := randomBase64EncodedString(10000)
	requireNoError(t, err)
	buffer := bytes.NewBuffer([]byte(data))
	req, err := http.NewRequest("POST", srv.URL, buffer)
	requireNoError(t, err)

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/soap+xml")
	req.Header.Set("Charset", "utf-8")

	//fmt.Printf("Do request\n")
	resp, err := c.Do(req)
	//fmt.Printf("Responded with %v\n", err)
	if err == nil {
		resp.Body.Close()
	}
	//fmt.Printf("Log: %s\n", log.String())
}

func requireNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err.Error())
	}
}

func randomBase64EncodedString(numBytes int) (string, error) {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

