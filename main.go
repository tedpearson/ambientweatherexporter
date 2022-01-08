package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tedpearson/ambientweatherexporter/weather"
)

func main() {
	port := flag.Int("port", 2184, "Http server port to listen on")
	name := flag.String("station-name", "Unknown",
		"Weather station name for the 'name' label on the metrics")
	flag.Parse()
	registry := prometheus.NewRegistry()
	factory := promauto.With(registry)
	http.Handle("/data/report/", weather.NewParser(*name, &factory))
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		panic(err)
	}
}
