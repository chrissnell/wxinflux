# wxinflux
This is a small daemon that reads weather measurements from a Si2000 radio dongle that has been loaded
with tridge's [DavisSi2000 firmware](https://github.com/tridge/DavisSi1000).  ```wxinflux``` processes
the readings and sends them off to [InfluxDB](https://influxdb.com) for storage.  Once they are being
sent to InfluxDB, you can use a tool like [Grafana](https://github.com/torkelo/grafana) to create beautiful,
dynamic dashboards for your weather station.

# Installation and Usage
1. You will need [Go 1.5](http://golang.org) or later installed on your computer
2. You will also need [InfluxDB](http://influxdb.com) up and running somewhere.
2. ```git clone git@github.com:chrissnell/wxinflux.git```
3. ```cd wxinflux```
4. ```go build```
5. Copy ```config.yaml.sample``` to ```config.yaml``` and edit as appropriate.
6. ```./wxinflux```

# Author
```wxinflux``` was written by Chris Snell, [http://chrissnell.com](http://chrissnell.com)
