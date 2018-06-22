package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"

	"github.com/orijtech/otils"
)

func main() {
	serverURL := "http://localhost:9889"
	httpClient := &http.Client{Transport: new(ochttp.Transport)}

	urls := []string{
		"https://google.com",
		"https://cloud.google.com/memorystore/",
		"https://opencensus.io",
		"https://twitter.com/opencensusio",
		"https://twitter.com/opencensusio",
		"http://images.mentalfloss.com/sites/default/files/styles/mf_image_16x9/public/62621-vangogh-starry_night.jpg",
	}

	routes := []string{
		"/purge",
		"/fetch",
		"/non-existent",
	}

	// Firstly change the seed
	rand.Seed(time.Now().Unix())

	// Enable tracing and metrics
	enableTracingAndMetrics()

	for i := 0; i < 20000; i++ {
		// Randomly pick len(routes) - 1
		urlsPerm := rand.Perm(len(urls))
		for k := 0; k < len(urlsPerm)-1; k++ {
			routesPerm := rand.Perm(len(routes))
			for j := 0; j < len(routesPerm)-1; j++ {
				route := routes[j]
				url := urls[k]
				body := fmt.Sprintf(`{"url": %q}`, url)
				req, err := http.NewRequest("POST", serverURL+route, strings.NewReader(body))
				if err != nil {
					log.Fatalf("Failed to compose a request %q: %v", route, err)
				}
				// Create a span for the client request and watch it be propagated to the server
				ctx, span := trace.StartSpan(context.Background(), "Client."+strings.Title(route))
				req = req.WithContext(ctx)
				res, err := httpClient.Do(req)
				if err != nil {
					log.Printf("Failed to make request to server:: %q: %v", route, err)
					span.SetStatus(trace.Status{Code: trace.StatusCodeInternal, Message: err.Error()})
					span.End()
					continue
				}
				_, _ = ioutil.ReadAll(res.Body)
				_ = res.Body.Close()
				span.End()
			}
		}

		// Randomly pick the sleep duration
		randSleep := time.Duration(rand.Intn(750)) * time.Millisecond
		fmt.Printf("Request: #%d Sleep %s\r", i+1, randSleep)
		<-time.After(randSleep)
	}
}

func enableTracingAndMetrics() {
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	se, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID:    otils.EnvOrAlternates("GCP_PROJECT_ID", "census-demos"),
		MetricPrefix: otils.EnvOrAlternates("GCP_METRIC_PREFIX", "cmstore"),
	})
	if err != nil {
		log.Fatalf("Failed to create Stackdriver exporter: %v", err)
	}
	trace.RegisterExporter(se)
	view.RegisterExporter(se)
	if err := view.Register(ochttp.DefaultServerViews...); err != nil {
		log.Fatalf("Failed to register server views: %v", err)
	}
	if err := view.Register(ochttp.DefaultClientViews...); err != nil {
		log.Fatalf("Failed to register client views: %v", err)
	}
}
