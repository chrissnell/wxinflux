package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	influx "github.com/influxdb/influxdb/client/v2"
	"github.com/tarm/goserial"

	"github.com/chrissnell/wxinflux/config"
)

// ConnStatus is used to indicate the status of the serial connection
type ConnStatus int

const (
	NotConnected ConnStatus = iota
	Connecting
	Connected
)

// WxPacket defines packets, as we receive them from the Si1000 ISS receiver
type WxPacket struct {
	Ready          bool    `json:"ready"`
	Status         string  `json:"status"`
	TransmitterID  uint8   `json:"transmitter_id,omitempty"`
	RSSI           uint16  `json:"RSSI,omitempty"`
	RxPackets      uint16  `json:"recv_packets,omitempty"`
	LostPackets    uint16  `json:"lost_packets,omitempty"`
	BadCRCPackets  uint16  `json:"bad_CRC,omitempty"`
	WindSpeed      uint8   `json:"wind_speed_mph,omitempty"`
	WindDir        uint16  `json:"wind_direction_degrees,omitempty"`
	Temperature    float32 `json:"temperature_F,omitempty"`
	Humidity       float32 `json:"humidity_pct,omitempty"`
	UVIndex        float32 `json:"UV_index,omitempty"`
	SolarRadiation float32 `json:"solar_Wm2,omitempty"`
	RainSpoons     uint32  `json:"rain_spoons,omitempty"`
	Raw            string  `json:"raw,omitempty"`
	Version        string  `json:"version,omitempty"`
}

// WxReport holds a weather report, derived from a WxPacket.
type WxReport struct {
	TransmitterID  uint8
	WindSpeed      uint8
	WindDir        uint16
	Temperature    float32
	Humidity       float32
	Dewpoint       float32
	HeatIndex      float32
	WindChill      float32
	UVIndex        float32
	SolarRadiation float32
	Rainfall       float32
}

// DavisSi1000 hold our connection to the Si1000-based USB ISS receiver
type DavisSi1000 struct {
	config   config.Config
	conn     io.ReadWriteCloser
	status   ConnStatus
	statusMu sync.RWMutex
}

// NewDavisSi1000 returns a new DavisSi1000 object
func NewDavisSi1000() *DavisSi1000 {
	return &DavisSi1000{}
}

func (d *DavisSi1000) connectToSerialSi1000() {
	var err error

	d.statusMu.RLock()

	switch d.status {
	case Connecting:
		d.statusMu.RUnlock()
		log.Println("Skipping reconnect since connection is in progress")
		return
	case NotConnected:
		d.statusMu.RUnlock()
		d.statusMu.Lock()
		d.status = Connecting
		d.statusMu.Unlock()
		log.Println("Connecting to local Si1000 on", d.config.Si2000.Device, "...")
		for {
			d.conn, err = serial.OpenPort(&serial.Config{Name: d.config.Si2000.Device, Baud: int(d.config.Si2000.Baud)})
			if err != nil {
				log.Println("Sleeping 5 seconds and trying again")
				time.Sleep(5 * time.Second)
			} else {
				// We connected.
				d.statusMu.Lock()
				d.status = Connected
				d.statusMu.Unlock()
				log.Println("Connection to local Si1000 on", d.config.Si2000.Device, "successful.")
				return
			}
		}
	}
}

// ReadReports reads wx reports from the Si1000 and sends them off for processing
func (d *DavisSi1000) readReports(reportChan chan<- WxReport) {
	for {
		// We recreate a json.Decoder with each loop because the connection may have dropped
		// and if it has, we'll need a fresh Decoder over that new net.Conn
		dec := json.NewDecoder(d.conn)

		for {
			var packet WxPacket
			if err := dec.Decode(&packet); err == io.EOF {
				log.Println("Error reading from device:", err)
				d.statusMu.Lock()
				d.status = NotConnected
				d.statusMu.Unlock()
				d.connectToSerialSi1000()
				break
			}
			report := generateWxReport(&packet)
			reportChan <- report
		}
	}
}

func (d *DavisSi1000) storeReports(reportChan <-chan WxReport, ic influx.Client) {
	for {
		select {
		case report := <-reportChan:
			bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
				Database:  d.config.InfluxDB.InfluxDBName,
				Precision: "s",
			})
			if err != nil {
				log.Println("Error logging report to InfluxDB:", err)
				continue
			}
			tags := map[string]string{"transmitter-id": string(report.TransmitterID)}
			fields := map[string]interface{}{
				"wind_speed":      report.WindSpeed,
				"wind_dir":        report.WindDir,
				"temperature":     report.Temperature,
				"humidity":        report.Humidity,
				"dewpoint":        report.Dewpoint,
				"heat_index":      report.HeatIndex,
				"wind_chill":      report.WindChill,
				"uv_index":        report.UVIndex,
				"solar_radiation": report.SolarRadiation,
				"rainfall":        report.Rainfall,
			}

			pt := influx.NewPoint("wxreport", tags, fields, time.Now())
			bp.AddPoint(pt)
			err = ic.Write(bp)
			if err != nil {
				log.Println("Error logging data point to InfluxDB:", err)
				continue
			}
			log.Printf("Received report: %+v\n", report)

		}
	}
}

// generateWxReport creates a human-usable weather report from the raw WxPacket
func generateWxReport(p *WxPacket) WxReport {
	r := WxReport{
		TransmitterID:  p.TransmitterID,
		WindSpeed:      p.WindSpeed,
		WindDir:        p.WindDir,
		Temperature:    p.Temperature,
		Humidity:       p.Humidity,
		Dewpoint:       dewpointFahrenheit(p.Temperature, p.Humidity),
		HeatIndex:      heatIndexFahrenheit(p.Temperature, p.Humidity),
		WindChill:      windchillFahrenheit(p.Temperature, float32(p.WindSpeed)),
		UVIndex:        p.UVIndex,
		SolarRadiation: p.SolarRadiation,
		Rainfall:       float32(p.RainSpoons) * float32(0.1),
	}
	return r
}

func main() {
	cfgFile := flag.String("config", "config.yaml", "Path to config file (default: ./config.yaml)")
	flag.Parse()

	reportChan := make(chan WxReport)

	d := NewDavisSi1000()

	// Read our server configuration
	filename, _ := filepath.Abs(*cfgFile)
	cfg, err := config.New(filename)
	if err != nil {
		log.Fatalln("Error reading config file.  Did you pass the -config flag?  Run with -h for help.\n", err)
	}
	d.config = cfg

	// Connect to influxdb
	u, _ := url.Parse(d.config.InfluxDB.InfluxURL)
	ic := influx.NewClient(influx.Config{
		URL:      u,
		Username: d.config.InfluxDB.InfluxUser,
		Password: d.config.InfluxDB.InfluxPass,
	})

	d.connectToSerialSi1000()
	go d.storeReports(reportChan, ic)
	d.readReports(reportChan)
}
