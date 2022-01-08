# ambientweatherexporter

Parses HTTP requests from Ambient Weather Stations and exposes a prometheus metrics endpoint
for scraping by a prometheus or VictoriaMetrics instance.

Does not implement all possible parameters from the 
[Device Data Specs](https://github.com/ambient-weather/api-docs/wiki/Device-Data-Specs).
Currently implements enough to support parameters from the WS-2000 including the indoor
sensor and the addon temperature sensors. If anyone needs more parameters supported, it is
trivial to add support.

## Usage:

### Install
- Download a binary from the latest [Release][release] if you use arm 32 bit (raspberry pi)

      curl -O https://github.com/tedpearson/weather2influxdb/releases/download/v1.1.0/ambientweatherexporter-linux-arm

- Make the binary executable

      chmod +x ambientweatherexporter-linux-arm

- If your architecture is not avaialable, you'll need to build from source:
  - Clone this repo
  - [Install Go][install-go]
  - 
        cd ambientweatherexporter
        go build

### Run

    ./ambientweatherexporter --port 1234 --station-name "your station"

Arguments:
- `--port` port to listen for ambient weather requests and prometheus scrapes
- `--station-name` the name of your weather station,
  which will populate the "name" label in the time series.

## How to configure a WS-2000 station to send http requests

1. Check the version of firmware and wifi firmware by [following these instructions](check).
   Your firmware needs to be 1.6.9 or later, with wifi firmware 4.2.8 or later.
2. Upgrade firmware if necessary:
   1. [Update hardware firmware using a MicroSD card](upgrade-hw)
   2. [Update wifi firmware using the awnet mobile app](upgrade-wifi)
      1. The WS-2000 will automatically show up in the awnet app, you don't need to press
         any of the weather station types that show up.
      2. You must be connected to the same network and not be using something
         like UniFi's device isolation setting or it won't work.
3. Configure the WS-2000 to send data to your server
   1. Go into Setup > Weather Server
   2. Select Customized > Setup and go into it
      1. Change State to Enable
      2. Protocol type should be Same as AMBWeather
      3. Put your IP/hostname and port
      4. Choose whatever interval you want reports. 
         You can scrape the metrics endpoint at whatever interval you desire as well.
      5. Leave the path as "/data/report/"
      6. All done! Go hit `http://yourip:port/metrics` and you should see your data!

[check]: https://help.ambientweather.net/help/ambient-weather-ws-2000-firmware-download-center/
[upgrade-hw]: https://help.ambientweather.net/help/how-do-i-update-firmware-ws-2000/
[upgrade-wifi]: https://help.ambientweather.net/help/how-do-i-update-the-wifi-firmware/