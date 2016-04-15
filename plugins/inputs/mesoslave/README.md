# Mesos Slave Input Plugin

This input plugin gathers metrics from Mesos slave .
For more information, please check the [Mesos Observability Metrics](http://mesos.apache.org/documentation/latest/monitoring/) page.

### Configuration:

```toml
# Telegraf plugin for gathering metrics from N Mesos masters
[[inputs.mesoslave]]
  # Timeout, in ms.
  timeout = 100
  # A list of Mesos masters, default value is localhost:5050.
  slaves = ["localhost:5051"]
  # Metrics groups to be collected, by default, all enabled.
  slave_collections = ["resources", "slave", "system", "executors", "tasks", "messages",]
```

