package weather

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Parser struct {
	name           string
	temperature    *prometheus.GaugeVec
	battery        *prometheus.GaugeVec // 1 = ok; 0 = low
	humidity       *prometheus.GaugeVec
	barometer      *prometheus.GaugeVec
	windDir        *prometheus.GaugeVec
	windSpeedMph   *prometheus.GaugeVec
	solarRadiation *prometheus.GaugeVec
	rainIn         *prometheus.GaugeVec
}

func NewParser(name string, factory *promauto.Factory) *Parser {
	return &Parser{
		name:           name,
		temperature:    newGauge(factory, "temperature", "name", "sensor"),
		battery:        newGauge(factory, "battery", "name", "sensor"),
		humidity:       newGauge(factory, "humidity", "name", "sensor"),
		barometer:      newGauge(factory, "barometer", "name", "type"),
		windDir:        newGauge(factory, "wind_dir", "name"),
		windSpeedMph:   newGauge(factory, "wind_speed_mph", "name", "type"),
		solarRadiation: newGauge(factory, "solar_radiation", "name"),
		rainIn:         newGauge(factory, "rain_in", "name", "period"),
	}
}

func newGauge(factory *promauto.Factory, name string, labels ...string) *prometheus.GaugeVec {
	opts := prometheus.GaugeOpts{
		Name: name,
	}
	return factory.NewGaugeVec(opts, labels)
}

func (p *Parser) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	// parse request url.
	// make url more easilily parseable
	queryStr := strings.Replace(req.URL.Path, "/data/report/", "", 1)
	// respond immediately
	resp.WriteHeader(http.StatusNoContent)
	values, err := url.ParseQuery(queryStr)
	if err != nil {
		log.Printf("Failed to parse weather observation from request url: %+v", err)
	}
	p.parse(values)
}

func (p *Parser) parse(values url.Values) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Failed to parse incoming request: %+v", r)
		}
	}()
	parseValue := func(name string) float64 {
		array := values[name]
		value, err := strconv.ParseFloat(array[0], 64)
		if err != nil {
			log.Printf("Failed to parse value: '%s': %+v", array[0], err)
			return 0
		}
		return value
	}

	for i := 1; values.Has(fmt.Sprintf("temp%df", i)); i++ {
		iStr := strconv.Itoa(i)
		p.battery.WithLabelValues(p.name, iStr).Set(parseValue(fmt.Sprintf("batt%d", i)))
		p.temperature.WithLabelValues(p.name, iStr).Set(parseValue(fmt.Sprintf("temp%df", i)))
	}

	p.temperature.WithLabelValues(p.name, "indoor").Set(parseValue("tempinf"))
	p.temperature.WithLabelValues(p.name, "outdoor").Set(parseValue("tempf"))
	p.battery.WithLabelValues(p.name, "outdoor").Set(parseValue("battout"))
	p.battery.WithLabelValues(p.name, "indoor").Set(parseValue("battin"))
	p.humidity.WithLabelValues(p.name, "outdoor").Set(parseValue("humidity"))
	p.humidity.WithLabelValues(p.name, "indoor").Set(parseValue("humidityin"))
	p.barometer.WithLabelValues(p.name, "relative").Set(parseValue("baromrelin"))
	p.barometer.WithLabelValues(p.name, "absolute").Set(parseValue("baromabsin"))
	p.windDir.WithLabelValues(p.name).Set(parseValue("winddir"))
	p.windSpeedMph.WithLabelValues(p.name, "sustained").Set(parseValue("windspeedmph"))
	p.windSpeedMph.WithLabelValues(p.name, "gusts").Set(parseValue("windgustmph"))
	p.solarRadiation.WithLabelValues(p.name).Set(parseValue("solarradiation"))
	p.rainIn.WithLabelValues(p.name, "hourly").Set(parseValue("hourlyrainin"))
	p.rainIn.WithLabelValues(p.name, "daily").Set(parseValue("dailyrainin"))
	p.rainIn.WithLabelValues(p.name, "weekly").Set(parseValue("weeklyrainin"))
	p.rainIn.WithLabelValues(p.name, "monthly").Set(parseValue("monthlyrainin"))
	p.rainIn.WithLabelValues(p.name, "yearly").Set(parseValue("yearlyrainin"))
	p.rainIn.WithLabelValues(p.name, "event").Set(parseValue("eventrainin"))
}
