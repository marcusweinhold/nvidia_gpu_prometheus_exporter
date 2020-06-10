build:
	docker build -t mbigras/nvidia_gpu_prometheus_exporter .

push: build
	docker push mbigras/nvidia_gpu_prometheus_exporter

.PHONY: build push
