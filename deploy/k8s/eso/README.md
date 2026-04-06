# ESO Bootstrap (Vault SecretStore)

This folder stores bootstrap manifests that must exist before Helm charts with `ExternalSecret` can sync.

## 1) Create Kubernetes Secret with an existing Vault token

`vault-token` is a Kubernetes Secret wrapper, not a generated Vault token.

For local dev with `vault -dev` from `docker-compose.yml`, the token is:
- `root` (default), or
- value of `${VAULT_DEV_ROOT_TOKEN_ID}` if you changed it.

```bash
kubectl create ns cryptotax --dry-run=client -o yaml | kubectl apply -f -

# set Vault token value used by Vault server
export VAULT_TOKEN="${VAULT_DEV_ROOT_TOKEN_ID:-root}"

kubectl -n cryptotax create secret generic vault-token \
  --from-literal=token="${VAULT_TOKEN}" \
  --dry-run=client -o yaml | kubectl apply -f -
```

## 2) Apply SecretStore

```bash
kubectl apply -f deploy/k8s/eso/secretstore-vault-backend.yaml
```

## 3) Verify

```bash
kubectl -n cryptotax get secretstore vault-backend
kubectl -n cryptotax describe secretstore vault-backend
```
