package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	addr = flag.String("addr", "localhost:8080", "http service address")
)

type device struct {
	id   string
	name string
	temp float64
	hum  float64
}

var devices = map[string]device{
	"device1": {"device1", "Living Room", 22.5, 60.0},
	"device2": {"device2", "Kitchen", 20.0, 50.0},
}

var (
	tempMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "device_temp_celsius",
		Help: "Current temperature in Celsius.",
	}, []string{"device"})
	humMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "device_humidity",
		Help: "Current humidity.",
	}, []string{"device"})
)

func updateMetrics() {
	for _, d := range devices {
		tempMetric.WithLabelValues(d.id).Set(d.temp)
		humMetric.WithLabelValues(d.id).Set(d.hum)
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		if mt == websocket.TextMessage {
			deviceID := string(message)
			d, ok := devices[deviceID]
			if ok {
				c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"id": "%s", "name": "%s", "temp": %f, "hum": %f}`, d.id, d.name, d.temp, d.hum)))
			}
		}
	}
}

func main() {
	flag.Parse()

	go func() {
		for range time.Tick(time.Second * 5) {
			updateMetrics()
		}
	}()

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	http.HandleFunc("/ws", wsHandler)

	http.ListenAndServe(*addr, nil)
}