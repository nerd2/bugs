package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"
)

// TestDumpRoundTripRace calls the various Dump functions at the obvious
// locations in implementations of http.RoundTripper and http.Handler.
//
// It is intended to fail if any of the Dump methods introduce a race with
// anything in the HTTP stack.
func main() {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
				go func(i int) {
					labels := pprof.Labels("it", strconv.Itoa(i))
					pprof.Do(context.Background(), labels, func(_ context.Context) {
						testDumpRoundTripRace("http", true)
						wg.Done()
					})
				}(i)
	}
	wg.Wait()
}

func testDumpRoundTripRace(scheme string, dumpBody bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if rand.Intn(2) > 0 {
		fmt.Printf("Cancelled immediately\n")
		cancel()
	} else {
		go func() {
			time.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond)
			fmt.Printf("Cancelled\n")
			cancel()
		}()
	}
	data, err := randomBase64EncodedString(10000)
	requireNoError(err)
	buffer := bytes.NewBuffer([]byte(data))
	req, err := http.NewRequest("POST", "http://abc.com/123", buffer)
	requireNoError(err)

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/soap+xml")
	req.Header.Set("Charset", "utf-8")

	httputil.DumpRequestOut(req, true)
}

func requireNoError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func randomBase64EncodedString(numBytes int) (string, error) {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

