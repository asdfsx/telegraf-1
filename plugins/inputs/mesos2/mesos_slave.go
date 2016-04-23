package mesos2

import (
    "log"
)


type MesoSlave struct {
	Mesos
}

// slaveBlocks serves as kind of metrics registry groupping them in sets
func (slave *MesoSlave) Blocks (g string) []string {
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
