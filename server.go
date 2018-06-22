package main

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/orijtech/otils"

	"go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

func main() {
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
	if err := view.Register(redis.ObservabilityMetricViews...); err != nil {
		log.Fatalf("Failed to register redis observability metric views: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/fetch", fetch)
	mux.HandleFunc("/purge", purge)
	h := &ochttp.Handler{Handler: mux}
	if err := http.ListenAndServe(":9889", h); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

type params struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

var redisPool *redis.Pool = &redis.Pool{
	MaxIdle:     5,
	IdleTimeout: 300 * time.Second,
	Dial: func() (redis.Conn, error) {
		return redis.Dial("tcp", otils.EnvOrAlternates("REDIS_SERVER_ADDR", ":6379"))
	},
}

func newRedisConn(ctx context.Context) redis.Conn {
	return redisPool.GetWithContext(ctx)
}

func parseJSON(ctx context.Context, rc io.ReadCloser, save interface{}) error {
	_, span := trace.StartSpan(ctx, "parseJSON")
	defer span.End()

	blob, err := ioutil.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return err
	}
	return json.Unmarshal(blob, save)
}

var httpClient = &http.Client{
	Transport: &ochttp.Transport{},
}

const crawledTable = "crawled"

func fetch(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "Fetch")
	defer span.End()

	pm := new(params)
	if err := parseJSON(ctx, r.Body, pm); err != nil {
		span.SetStatus(trace.Status{Code: int32(trace.StatusCodeInternal), Message: err.Error()})
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	conn := newRedisConn(ctx)
	defer conn.Close()

	// Check cache if we already downloaded the file
	data, err := conn.Do("GET", crawledTable, pm.URL)
	if err == nil && data != nil {
		dt := data.([]byte)
		w.Write(dt)
		return
	}

	span.SetStatus(trace.Status{Code: int32(trace.StatusCodeNotFound), Message: "Cache miss"})
	// Otherwise now fetch it
	req, err := http.NewRequest("GET", pm.URL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req = req.WithContext(ctx)

	res, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	blob, err := ioutil.ReadAll(res.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Now cache it
        _3HoursInSeconds := 3 * 60 * 60
	if _, err := conn.Do("SETEX", crawledTable, pm.URL, _3HoursInSeconds, blob); err != nil {
		span.SetStatus(trace.Status{Code: int32(trace.StatusCodeInternal), Message: err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Then send back the response
	w.Write(blob)
}

func purge(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "Purge")
	defer span.End()

	pm := new(params)
	if err := parseJSON(ctx, r.Body, pm); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	conn := newRedisConn(ctx)
	defer conn.Close()

	_, err := conn.Do("HDEL", crawledTable, pm.URL)
	if err != nil {
		span.SetStatus(trace.Status{Code: int32(trace.StatusCodeInternal), Message: err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
