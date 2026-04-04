SHELL := /bin/zsh

HELMFILE ?= helmfile
HELMFILE_FILE ?= helmfile.yaml.gotmpl
NAMESPACE ?= cryptotax

.PHONY: deploy diff status destroy list
.PHONY: stop start pods

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
