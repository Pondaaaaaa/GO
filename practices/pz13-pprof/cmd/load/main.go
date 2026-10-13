// A bounded, sequential, scheduled workload with correctness checks.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func main() {
	endpoint := flag.String("url", "", "loopback /work URL")
	count := flag.Int("count", 24, "requests, 1..100")
	interval := flag.Duration("interval", 500*time.Millisecond, "spacing between arrivals")
	flag.Parse()
	u, err := url.Parse(*endpoint)
	if err != nil || u.Hostname() != "127.0.0.1" || u.Scheme != "http" || *count < 1 || *count > 100 || *interval < 100*time.Millisecond {
		log.Fatal("invalid bounded loopback workload")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	start := time.Now()
	var total, max time.Duration
	late := 0
	for i := 0; i < *count; i++ {
		due := start.Add(time.Duration(i) * (*interval))
		if delay := time.Until(due); delay > 0 {
			time.Sleep(delay)
		} else if i > 0 && -delay > 5*time.Millisecond {
			late++
		}
		t0 := time.Now()
		resp, err := client.Get(*endpoint)
		if err != nil {
			log.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 || strings.TrimSpace(string(body)) != "39088169" {
			log.Fatalf("invalid /work result: status=%d body=%q err=%v", resp.StatusCode, body, err)
		}
		duration := time.Since(t0)
		total += duration
		if duration > max {
			max = duration
		}
	}
	fmt.Printf("requests=%d concurrency=1 interval=%s target_rps=%.1f elapsed=%s mean_http=%s max_http=%s late_arrivals=%d result=39088169\n", *count, *interval, float64(time.Second)/float64(*interval), time.Since(start), total/time.Duration(*count), max, late)
}
