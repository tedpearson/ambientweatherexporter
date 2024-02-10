package weather

import (
	"fmt"
	"log"
	"math"
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
	ultraviolet    *prometheus.GaugeVec
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
		ultraviolet:    newGauge(factory, "ultraviolet", "name"),
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
	parseValue := func(name string) (float64, error) {
		array, ok := values[name]
		if !ok {
			return 0, fmt.Errorf("no such param: %s", name)
		}
		value, err := strconv.ParseFloat(array[0], 64)
		if err != nil {
			e := fmt.Errorf("failed to parse value: '%s': %+v", array[0], err)
			log.Println(e)
			return 0, e
		}
		return value, nil
	}

	for i := 1; i <= 10; i++ {
		iStr := strconv.Itoa(i)
		if values.Has(fmt.Sprintf("temp%df", i)) {
			updateGauge(p.temperature.WithLabelValues(p.name, iStr))(parseValue(fmt.Sprintf("temp%df", i)))
			updateGauge(p.battery.WithLabelValues(p.name, iStr))(parseValue("batt" + iStr))
		}
		if values.Has("soilhum" + iStr) {
			updateGauge(p.humidity.WithLabelValues(p.name, "soil"+iStr))(parseValue("soilhum" + iStr))
			updateGauge(p.battery.WithLabelValues(p.name, "soil"+iStr))(parseValue("battsm" + iStr))
		}
		if values.Has("humidity" + iStr) {
			updateGauge(p.humidity.WithLabelValues(p.name, iStr))(parseValue("humidity" + iStr))
		}
	}

	updateGauge(p.temperature.WithLabelValues(p.name, "indoor"))(parseValue("tempinf"))
	tempF, err := parseValue("tempf")
	if err != nil {
		p.temperature.WithLabelValues(p.name, "outdoor").Set(tempF)
		feelsLike := tempF
		windSpeedMph, err := parseValue("windspeedmph")
		if err != nil {
			p.windSpeedMph.WithLabelValues(p.name, "sustained").Set(windSpeedMph)
			if tempF <= 40 {
				feelsLike = calculateWindChill(tempF, windSpeedMph)
			}
		}
		humidity, err := parseValue("humidity")
		if err != nil {
			p.humidity.WithLabelValues(p.name, "outdoor").Set(humidity)
			p.temperature.WithLabelValues(p.name, "dewpoint").Set(calculateDewPoint(tempF, humidity))
			if tempF >= 80 {
				feelsLike = calculateHeatIndex(tempF, humidity)
			}
		}
		p.temperature.WithLabelValues(p.name, "feelsLike").Set(feelsLike)
	}

	updateGauge(p.battery.WithLabelValues(p.name, "outdoor"))(parseValue("battout"))
	updateGauge(p.battery.WithLabelValues(p.name, "indoor"))(parseValue("battin"))
	updateGauge(p.humidity.WithLabelValues(p.name, "indoor"))(parseValue("humidityin"))
	updateGauge(p.barometer.WithLabelValues(p.name, "relative"))(parseValue("baromrelin"))
	updateGauge(p.barometer.WithLabelValues(p.name, "absolute"))(parseValue("baromabsin"))
	updateGauge(p.windDir.WithLabelValues(p.name))(parseValue("winddir"))
	updateGauge(p.windSpeedMph.WithLabelValues(p.name, "gusts"))(parseValue("windgustmph"))
	updateGauge(p.solarRadiation.WithLabelValues(p.name))(parseValue("solarradiation"))
	updateGauge(p.rainIn.WithLabelValues(p.name, "daily"))(parseValue("dailyrainin"))
	updateGauge(p.rainIn.WithLabelValues(p.name, "weekly"))(parseValue("weeklyrainin"))
	updateGauge(p.rainIn.WithLabelValues(p.name, "monthly"))(parseValue("monthlyrainin"))
	updateGauge(p.rainIn.WithLabelValues(p.name, "yearly"))(parseValue("yearlyrainin"))
	updateGauge(p.rainIn.WithLabelValues(p.name, "event"))(parseValue("eventrainin"))
	updateGauge(p.ultraviolet.WithLabelValues(p.name))(parseValue("uv"))
}

func updateGauge(gauge prometheus.Gauge) func(float64, error) {
	return func(value float64, err error) {
		if err == nil {
			gauge.Set(value)
		}
	}
}

func calculateWindChill(tempF float64, windSpeedMph float64) float64 {
	if tempF > 40 || windSpeedMph < 5 {
		return tempF
	}
	windExp := math.Pow(windSpeedMph, 0.16)
	return 35.74 + (0.6215 * tempF) - (35.75 * windExp) + (0.4275 * tempF * windExp)
}

// following equation from https://www.wpc.ncep.noaa.gov/html/heatindex_equation.shtml
func calculateHeatIndex(tempF float64, rh float64) float64 {
	if tempF < 80 {
		return tempF
	}
	simpleHI := 0.5 * (tempF + 61 + ((tempF - 68) * 1.2) + (rh * .094))
	if simpleHI < 80 {
		return simpleHI
	}
	hi := -42.379 +
		2.04901523*tempF +
		10.14333127*rh -
		.22475541*tempF*rh -
		.00683783*tempF*tempF -
		.05481717*rh*rh +
		.00122874*tempF*tempF*rh +
		.00085282*tempF*rh*rh -
		.00000199*tempF*tempF*rh*rh
	if rh < 13 && tempF >= 80 && tempF <= 112 {
		hi = hi - ((13-rh)/4)*math.Sqrt((17-math.Abs(tempF-95))/17)
	} else if rh > 85 && tempF >= 80 && tempF <= 87 {
		hi = hi + ((rh-85)/10)*((87-tempF)/5)
	}
	return hi
}

func calculateDewPoint(tempF float64, rh float64) float64 {
	a := 17.625
	b := 243.04
	t := (tempF - 32) * 5 / 9
	alpha := math.Log(rh/100) + ((a * t) / (b + t))
	return (b * alpha / (a - alpha) * 9 / 5) + 32
}
