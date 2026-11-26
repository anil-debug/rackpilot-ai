# In-cluster Prometheus manifests

Canonical scrape config for editing: `monitoring/prometheus/prometheus-k8s.yml`

After editing, sync into this directory for Kustomize:

```bash
cp ../../../monitoring/prometheus/prometheus-k8s.yml prometheus/prometheus.yml
kubectl apply -k ..
```