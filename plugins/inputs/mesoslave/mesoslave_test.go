package mesoslave

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/influxdata/telegraf/testutil"
)

var mesosMetrics map[string]interface{}
var ts *httptest.Server

func generateMetrics() {
	mesosMetrics = make(map[string]interface{})

	metricNames := []string{"slave/cpus_percent", "slave/cpus_used", "slave/cpus_total",
		"slave/cpus_revocable_percent", "slave/cpus_revocable_total", "slave/cpus_revocable_used",
		"slave/disk_percent", "slave/disk_used", "slave/disk_total", "slave/disk_revocable_percent",
		"slave/disk_revocable_total", "slave/disk_revocable_used", "slave/mem_percent",
		"slave/mem_used", "slave/mem_total", "slave/mem_revocable_percent", "slave/mem_revocable_total",
		"slave/mem_revocable_used", "slave/registere", "slave/uptime_secs", "system/cpus_total",
		"system/load_15min", "system/load_5min", "system/load_1min", "system/mem_free_bytes",
		"system/mem_total_bytes", "containerizer/mesos/container_destroy_errors",
		"slave/container_launch_errors", "slave/executors_preempted", "slave/frameworks_active",
		"slave/executor_directory_max_allowed_age_secs", "slave/executors_registering",
		"slave/executors_running", "slave/executors_terminated", "slave/executors_terminating",
		"slave/recovery_errors", "slave/tasks_failed", "slave/tasks_finished", "slave/tasks_killed",
		"slave/tasks_lost", "slave/tasks_running", "slave/tasks_staging", "slave/tasks_starting",
        "slave/invalid_framework_messages","slave/invalid_status_updates","slave/valid_framework_messages",
		"slave/valid_status_updates",}

	for _, k := range metricNames {
		mesosMetrics[k] = rand.Float64()
	}
}

func TestMain(m *testing.M) {
	generateMetrics()
	r := http.NewServeMux()
	r.HandleFunc("/metrics/snapshot", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mesosMetrics)
	})
	ts = httptest.NewServer(r)
	rc := m.Run()
	ts.Close()
	os.Exit(rc)
}

func TestMesoSlave(t *testing.T) {
	var acc testutil.Accumulator

	m := MesoSlave{
		Slaves: []string{ts.Listener.Addr().String()},
		Timeout: 10,
	}

	err := m.Gather(&acc)

	if err != nil {
		t.Errorf(err.Error())
	}

	acc.AssertContainsFields(t, "mesoslave", mesosMetrics)
}

func TestRemoveGroup(t *testing.T) {
	generateMetrics()

	m := MesoSlave{
		SlaveCols: []string{
			"resources", "slave",
		},
	}
	b := []string{
		"system", "executors", "tasks",
		"messages",
	}

	m.removeGroup(&mesosMetrics)

	for _, v := range b {
		for _, x := range slaveBlocks(v) {
			if _, ok := mesosMetrics[x]; ok {
				t.Errorf("Found key %s, it should be gone.", x)
			}
		}
	}
	for _, v := range m.SlaveCols {
		for _, x := range slaveBlocks(v) {
			if _, ok := mesosMetrics[x]; !ok {
				t.Errorf("Didn't find key %s, it should present.", x)
			}
		}
	}
}
