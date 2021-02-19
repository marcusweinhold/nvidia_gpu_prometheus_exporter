build:
	docker build -t cfsmp3/nvidia_gpu_prometheus_exporter .

push: build
	docker push cfsmp3/nvidia_gpu_prometheus_exporter

.PHONY: build push
