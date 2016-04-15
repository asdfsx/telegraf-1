package mesoslave

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

type MesoSlave struct {
	Timeout    int
	Slaves    []string
	SlaveCols []string `toml:"slave_collections"`
}

var defaultMetrics = []string{
	"resources", "slave", "system", "executors",
	"tasks", "messages",
}

var sampleConfig = `
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
`

// SampleConfig returns a sample configuration block
func (m *MesoSlave) SampleConfig() string {
	return sampleConfig
}

// Description just returns a short description of the Mesos plugin
func (m *MesoSlave) Description() string {
	return "Telegraf plugin for gathering metrics from N Mesos slaves"
}

// Gather() metrics from given list of Mesos Slaves
func (m *MesoSlave) Gather(acc telegraf.Accumulator) error {
	var wg sync.WaitGroup
	var errorChannel chan error

	if len(m.Slaves) == 0 {
		m.Slaves = []string{"localhost:5051"}
	}

	errorChannel = make(chan error, len(m.Slaves)*2)

	for _, v := range m.Slaves {
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
func metricsDiff(w []string) []string {
	b := []string{}
	s := make(map[string]bool)

	if len(w) == 0 {
		return b
	}

	for _, v := range w {
		s[v] = true
	}

	for _, d := range defaultMetrics {
		if _, ok := s[d]; !ok {
			b = append(b, d)
		}
	}

	return b
}

// slaveBlocks serves as kind of metrics registry groupping them in sets
func slaveBlocks(g string) []string {
	var m map[string][]string

	m = make(map[string][]string)

	m["resources"] = []string{
		"slave/cpus_percent",
		"slave/cpus_used",
		"slave/cpus_total",
		"slave/cpus_revocable_percent",
		"slave/cpus_revocable_total",
		"slave/cpus_revocable_used",
		"slave/disk_percent",
		"slave/disk_used",
		"slave/disk_total",
		"slave/disk_revocable_percent",
		"slave/disk_revocable_total",
		"slave/disk_revocable_used",
		"slave/mem_percent",
		"slave/mem_used",
		"slave/mem_total",
		"slave/mem_revocable_percent",
		"slave/mem_revocable_total",
		"slave/mem_revocable_used",
	}

	m["slave"] = []string{
		"slave/registere",
		"slave/uptime_secs",
	}

	m["system"] = []string{
		"system/cpus_total",
		"system/load_15min",
		"system/load_5min",
		"system/load_1min",
		"system/mem_free_bytes",
		"system/mem_total_bytes",
	}

	m["executors"] = []string{
		"containerizer/mesos/container_destroy_errors",
		"slave/container_launch_errors",
		"slave/executors_preempted",
		"slave/frameworks_active",
		"slave/executor_directory_max_allowed_age_secs",
		"slave/executors_registering",
		"slave/executors_running",
		"slave/executors_terminated",
		"slave/executors_terminating",
		"slave/recovery_errors",
	}

	m["tasks"] = []string{
		"slave/tasks_failed",
		"slave/tasks_finished",
		"slave/tasks_killed",
		"slave/tasks_lost",
		"slave/tasks_running",
        "slave/tasks_staging",
        "slave/tasks_starting",
	}

	m["messages"] = []string{
		"slave/invalid_framework_messages",
		"slave/invalid_status_updates",
		"slave/valid_framework_messages",
		"slave/valid_status_updates",
	}

	ret, ok := m[g]

	if !ok {
		log.Println("[mesoslave] Unkown metrics group: ", g)
		return []string{}
	}

	return ret
}

// removeGroup(), remove unwanted sets
func (m *MesoSlave) removeGroup(j *map[string]interface{}) {
	var ok bool

	b := metricsDiff(m.SlaveCols)

	for _, k := range b {
		for _, v := range slaveBlocks(k) {
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
func (m *MesoSlave) gatherMetrics(a string, acc telegraf.Accumulator) error {
	var jsonOut map[string]interface{}

	host, _, err := net.SplitHostPort(a)
	if err != nil {
		host = a
		a = a + ":5051"
	}

	tags := map[string]string{
		"server": host,
	}

	if m.Timeout == 0 {
		log.Println("[mesoslave] Missing timeout value, setting default value (100ms)")
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

	acc.AddFields("mesoslave", jf.Fields, tags)

	return nil
}

func init() {
	inputs.Add("mesoslave", func() telegraf.Input {
		return &MesoSlave{}
	})
}
