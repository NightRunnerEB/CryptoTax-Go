# Istio Routing + mTLS

Prerequisite for direct `JWKS` fetching from `auth-svc`:

```bash
kubectl -n istio-system set env deployment/istiod PILOT_JWT_ENABLE_REMOTE_JWKS=envoy
kubectl -n istio-system rollout status deployment/istiod
```

Why this is needed:

- with the default Istio behavior, `jwksUri` fetching is handled by `istiod`
- for our setup the correct model is Envoy-based remote JWKS fetching
- this lets `RequestAuthentication` use the direct in-cluster URL:
  - `http://auth-svc.cryptotax.svc.cluster.local:8085/.well-known/jwks.json`

Apply:

```bash
kubectl apply -f deploy/k8s/istio/gateway.yaml
kubectl apply -f deploy/k8s/istio/virtualservices.yaml
kubectl apply -f deploy/k8s/istio/destinationrules.yaml
kubectl apply -f deploy/k8s/istio/peerauthentication-strict.yaml
kubectl apply -f deploy/k8s/istio/requestauthentication.yaml
kubectl apply -f deploy/k8s/istio/ingress-authorizationpolicy.yaml
kubectl apply -f deploy/k8s/istio/authorizationpolicies.yaml
```

Check:

```bash
kubectl -n cryptotax get gateway,virtualservice,destinationrule,peerauthentication,authorizationpolicy
kubectl -n istio-system get requestauthentication,authorizationpolicy
```

Local test via port-forward:

```bash
kubectl -n istio-system port-forward svc/istio-ingressgateway 8080:80
```

Public routes:

- `curl http://127.0.0.1:8080/.well-known/jwks.json`
- `curl -X POST http://127.0.0.1:8080/auth/login`
- `curl -X POST http://127.0.0.1:8080/auth/register`
- `curl http://127.0.0.1:8080/auth/verify?token=dummy`

Protected routes must carry a valid bearer token. Example:

```bash
TOKEN="<access-token>"

curl -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8080/exchanges/supported

curl -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8080/fiat-currencies

curl -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8080/tax/profile
```

Runtime behavior:

- `RequestAuthentication` validates JWT on `istio-ingressgateway`
- validated claims are copied to:
  - `x-user-id` from `sub`
  - `x-role` from `role`
- ingress `AuthorizationPolicy` requires a valid JWT for protected routes
- backend `AuthorizationPolicy` requires these propagated headers on ingress-to-service traffic

Mesh check:

```bash
kubectl -n cryptotax run mesh-curl --image=curlimages/curl:8.12.1 --restart=Never --command -- sleep 3600
kubectl -n cryptotax wait --for=condition=Ready pod/mesh-curl --timeout=120s
kubectl -n cryptotax exec mesh-curl -c mesh-curl -- curl -sS -w '\n%{http_code}\n' http://auth-svc:8085/health
kubectl -n cryptotax exec mesh-curl -c mesh-curl -- curl -sS -w '\n%{http_code}\n' http://ledger-svc:8086/health
kubectl -n cryptotax delete pod mesh-curl
```
