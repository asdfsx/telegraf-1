package mesos2

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	jsonparser "github.com/influxdata/telegraf/plugins/parsers/json"
)

type Mesos struct {
	Timeout    int
    Servers    []string
	Collections []string `toml:"collections"`
	DefaultMetrics    []string
	Config string
    Desc string
    Measurement string
    DefaultPort string
}

// SampleConfig returns a sample configuration block
func (m *Mesos) SampleConfig() string {
	return m.Config
}

// Description just returns a short description of the Mesos plugin
func (m *Mesos) Description() string {
	return m.Desc
}

// Gather() metrics from given list of Mesos Servers
func (m *Mesos) Gather(acc telegraf.Accumulator) error {
	var wg sync.WaitGroup
	var errorChannel chan error

	if len(m.Servers) == 0 {
		m.Servers = []string{"localhost:" + m.DefaultPort}
	}

	errorChannel = make(chan error, len(m.Servers)*2)

	for _, v := range m.Servers {
		wg.Add(1)
		go func(c string) {
			errorChannel <- m.gatherMetrics(c, acc)
			wg.Done()
			return
		}(v)
	}

	wg.Wait()
	close(errorChannel)
	errorStrings := []string{}

	// Gather all errors for returning them at once
	for err := range errorChannel {
		if err != nil {
			errorStrings = append(errorStrings, err.Error())
		}
	}

	if len(errorStrings) > 0 {
		return errors.New(strings.Join(errorStrings, "\n"))
	}
	return nil
}

// metricsDiff() returns set names for removal
func (m * Mesos) metricsDiff(w []string) []string {
	b := []string{}
	s := make(map[string]bool)

	if len(w) == 0 {
		return b
	}

	for _, v := range w {
		s[v] = true
	}

	for _, d := range m.DefaultMetrics {
		if _, ok := s[d]; !ok {
			b = append(b, d)
		}
	}

	return b
}

// Blocks serves as kind of metrics registry groupping them in sets
func (m *Mesos) Blocks(g string) []string {
	return []string{} 
}

// removeGroup(), remove unwanted sets
func (m *Mesos) removeGroup(j *map[string]interface{}) {
	var ok bool

	b := m.metricsDiff(m.Collections)

	for _, k := range b {
		for _, v := range m.Blocks(k) {
			if _, ok = (*j)[v]; ok {
				delete((*j), v)
			}
		}
	}
}

var tr = &http.Transport{
	ResponseHeaderTimeout: time.Duration(3 * time.Second),
}

var client = &http.Client{
	Transport: tr,
	Timeout:   time.Duration(4 * time.Second),
}

// This should not belong to the object
func (m *Mesos) gatherMetrics(a string, acc telegraf.Accumulator) error {
	var jsonOut map[string]interface{}

	host, _, err := net.SplitHostPort(a)
	if err != nil {
		host = a
		a = a + ":" + m.DefaultPort
	}

	tags := map[string]string{
		"server": host,
	}

	if m.Timeout == 0 {
		log.Println("[mesos] Missing timeout value, setting default value (100ms)")
		m.Timeout = 100
	}

	ts := strconv.Itoa(m.Timeout) + "ms"

	resp, err := client.Get("http://" + a + "/metrics/snapshot?timeout=" + ts)

	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(data), &jsonOut); err != nil {
		return errors.New("Error decoding JSON response")
	}

	m.removeGroup(&jsonOut)

	jf := jsonparser.JSONFlattener{}

	err = jf.FlattenJSON("", jsonOut)

	if err != nil {
		return err
	}

	acc.AddFields(m.Measurement, jf.Fields, tags)

	return nil
}

func init() {
	inputs.Add("mesos_master", func() telegraf.Input {
		return &MesosMaster{Mesos{
            DefaultPort : "5050",
            Measurement : "mesos_master",
            DefaultMetrics : []string{
	            "resources", "master", "system", "slaves", "frameworks",
	            "tasks", "messages", "evqueue", "messages", "registrar",
            },
            Config : `
  # Timeout, in ms.
  timeout = 100
  # A list of Mesos masters, default value is localhost:5050.
  masters = ["localhost:5050"]
  # Metrics groups to be collected, by default, all enabled.
  master_collections = [
    "resources",
    "master",
    "system",
    "slaves",
    "frameworks",
    "messages",
    "evqueue",
    "registrar",
  ]
`,
            Desc : "Telegraf plugin for gathering metrics from N Mesos masters",
        }}
	})
    inputs.Add("mesos_slave", func() telegraf.Input {
		return &MesoSlave{Mesos{
            DefaultPort : "5051",
            Measurement : "mesos_slave",
            DefaultMetrics : []string{
	            "resources", "slave", "system", "executors",
	            "tasks", "messages",
            },
            Config : `
  # Timeout, in ms.
  timeout = 100
  # A list of Mesos slaves, default value is localhost:5050.
  slaves = ["localhost:5051"]
  # Metrics groups to be collected, by default, all enabled.
  slave_collections = [
    "resources",
    "slave",
    "system",
    "executors",
    "tasks",
    "messages",
  ]
`,
            Desc : "Telegraf plugin for gathering metrics from N Mesos slaves",
        }}
	})
}
