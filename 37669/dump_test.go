package dump

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"testing"
	"time"
)

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
	req, err := http.NewRequest("POST", "http://abc.com/123", buffer)
	requireNoError(t, err)

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/soap+xml")
	req.Header.Set("Charset", "utf-8")

	httputil.DumpRequestOut(req, true)
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

