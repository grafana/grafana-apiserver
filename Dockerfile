FROM alpine
ADD build/grafana-apiserver /
ENTRYPOINT ["/grafana-apiserver"]
