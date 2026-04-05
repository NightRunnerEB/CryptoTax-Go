# Istio Routing + mTLS

Apply:

```bash
kubectl apply -f deploy/k8s/istio/gateway.yaml
kubectl apply -f deploy/k8s/istio/virtualservices.yaml
kubectl apply -f deploy/k8s/istio/destinationrules.yaml
kubectl apply -f deploy/k8s/istio/peerauthentication-strict.yaml
kubectl apply -f deploy/k8s/istio/authorizationpolicies.yaml
```

Check:

```bash
kubectl -n cryptotax get gateway,virtualservice,destinationrule,peerauthentication,authorizationpolicy
```

Local test via port-forward:

```bash
kubectl -n istio-system port-forward svc/istio-ingressgateway 8080:80
```

Then call:

- `curl http://127.0.0.1:8080/auth/verify?token=dummy`
- `curl http://127.0.0.1:8080/v1/exchanges/supported`
- `curl http://127.0.0.1:8080/v1/fiat-currencies`
- `curl http://127.0.0.1:8080/v1/tenants/<TENANT_ID>/tax/reports`

Mesh check:

```bash
kubectl -n cryptotax run mesh-curl --image=curlimages/curl:8.12.1 --restart=Never --command -- sleep 3600
kubectl -n cryptotax wait --for=condition=Ready pod/mesh-curl --timeout=120s
kubectl -n cryptotax exec mesh-curl -c mesh-curl -- curl -sS -w '\n%{http_code}\n' http://auth-svc:8085/health
kubectl -n cryptotax exec mesh-curl -c mesh-curl -- curl -sS -w '\n%{http_code}\n' http://ledger-svc:8086/health
kubectl -n cryptotax delete pod mesh-curl
```
