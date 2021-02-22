NVIDIA GPU Prometheus Exporter - community fork
-----------------------------------------------

This is a [Prometheus Exporter](https://prometheus.io/docs/instrumenting/exporters/) for
exporting NVIDIA GPU metrics. It uses the [Go bindings](https://github.com/cfsmp3/gonvml)
for [NVIDIA Management Library](https://developer.nvidia.com/nvidia-management-library-nvml)
(NVML) which is a C-based API that can be used for monitoring NVIDIA GPU devices.
Unlike some other similar exporters, it does not call the
[`nvidia-smi`](https://developer.nvidia.com/nvidia-system-management-interface) binary.

Note: I'm calling this "community" because the starting point is a merge of pending
requests on the original mindprince's repo, plus a lot of changes from other forks
that for whatever reason weren't sent upstream at all.

And then, whatever else I add.

This fork will be used extensively in production and as such as it will maintained
for the forseable future.

## Building

```
make build
make push
```

## Running

The exporter requires the following:
- access to NVML library (`libnvidia-ml.so.1`).
- access to the GPU devices.

To make sure that the exporter can access the NVML libraries, either add them
to the search path for shared libraries. Or set `LD_LIBRARY_PATH` to point to
their location.

By default the metrics are exposed on port `9445`. This can be updated using
the `-web.listen-address` flag.

## Running inside a container

There's a docker image available on Docker Hub at
[cfsmp3/nvidia_gpu_prometheus_exporter](https://hub.docker.com/r/cfsmp3/nvidia_gpu_prometheus_exporter/)

If you are running the exporter inside a container, you will need to do the
following to give the container access to NVML library:
```
-e LD_LIBRARY_PATH=<path-where-nvml-is-present>
--volume <above-path>:<above-path>
```

And you will need to do one of the following to give it access to the GPU
devices:
- Run with `--privileged`
- If you are on docker v17.04.0-ce or above, run with `--device-cgroup-rule 'c 195:* mrw'`
