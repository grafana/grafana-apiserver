# grafana-apiserver

<hr/>

**This repository is currently *experimental*, which means that interfaces and behavior may change as it evolves.**

<hr/>

create kind k8s cluster, build grafana-apiserver, and run it: 

```
tilt up
```

add dashboard GrafanaResourceDefinition:
```
kubectl apply -f deploy/example/dashboard-grd.yaml
```

add test dashboard:
```
kubectl apply -f deploy/example/dashboard-test.yaml
```

list dashboards:
```
kubectl get dashboards
```
