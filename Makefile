SHELL := /bin/zsh

HELMFILE ?= helmfile
HELMFILE_FILE ?= helmfile.yaml.gotmpl
NAMESPACE ?= cryptotax
VAULT_TOKEN ?= root
LOCAL_SECRETS_FILE ?= deploy/local-secrets/vault.local.env

.PHONY: deploy diff status destroy list
.PHONY: stop start pods
.PHONY: bootstrap apply-istio seed-vault-local

deploy:
	$(HELMFILE) -f $(HELMFILE_FILE) -e default sync --state-values-set namespace=$(NAMESPACE)

diff:
	$(HELMFILE) -f $(HELMFILE_FILE) -e default diff --state-values-set namespace=$(NAMESPACE)

status:
	$(HELMFILE) -f $(HELMFILE_FILE) -e default status --state-values-set namespace=$(NAMESPACE)

list:
	$(HELMFILE) -f $(HELMFILE_FILE) -e default list --state-values-set namespace=$(NAMESPACE)

destroy:
	$(HELMFILE) -f $(HELMFILE_FILE) -e default destroy --state-values-set namespace=$(NAMESPACE)

# Pause all deployments in namespace without deleting Helm releases.
stop:
	kubectl -n $(NAMESPACE) scale deployment --all --replicas=0

# Restore desired replicas from Helm values.
start:
	$(HELMFILE) -f $(HELMFILE_FILE) -e default sync --state-values-set namespace=$(NAMESPACE)

pods:
	kubectl get pods -n $(NAMESPACE)

# Bootstrap namespace + ESO SecretStore prerequisites.
bootstrap:
	kubectl create ns $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	kubectl -n $(NAMESPACE) create secret generic vault-token --from-literal=token="$(VAULT_TOKEN)" --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f deploy/k8s/eso/secretstore-vault-backend.yaml

# Apply Istio gateway/routing/security manifests used by CryptoTax.
apply-istio:
	kubectl -n istio-system set env deployment/istiod PILOT_JWT_ENABLE_REMOTE_JWKS=envoy
	kubectl -n istio-system rollout status deployment/istiod
	kubectl apply -f deploy/k8s/istio/gateway.yaml
	kubectl apply -f deploy/k8s/istio/virtualservices.yaml
	kubectl apply -f deploy/k8s/istio/destinationrules.yaml
	kubectl apply -f deploy/k8s/istio/peerauthentication-strict.yaml
	kubectl apply -f deploy/k8s/istio/requestauthentication.yaml
	kubectl apply -f deploy/k8s/istio/ingress-authorizationpolicy.yaml
	kubectl apply -f deploy/k8s/istio/authorizationpolicies.yaml

# Seed local Vault (dev mode) with test secrets for all services.
seed-vault-local:
	./scripts/seed-local-vault.sh "$(LOCAL_SECRETS_FILE)"
