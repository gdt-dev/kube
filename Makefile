VERSION ?= $(shell git describe --tags --always --dirty)
CLUSTERS = $(shell kind get clusters)

.PHONY: test

kind-install:
ifeq (, $(shell command -v kind))
	go install sigs.k8s.io/kind@v0.30.0
endif

# We clear the test cache because some of the tests require an out-of-band KinD
# cluster to run against and we want to re-run tests against that KinD cluster
# instead of from cached unit test results.
clear-test-cache:
	@echo -n "clearing Go test cache ... "
	@go clean -testcache
	@echo "ok."

kind-clear-clusters:
ifneq (, $(shell command -v kind))
ifneq (, $(CLUSTERS))
	@echo -n "clearing KinD clusters ... "
	@for c in $(CLUSTERS); do kind delete cluster -q --name $$c; done
	@echo "ok."
endif
endif

kind-create-cluster:
ifneq (, $(shell command -v kind))
	@echo -n "creating 'kind' cluster ... "
	@kind create cluster -q
	@echo "ok."
	@sleep 5
endif

test: clear-test-cache kind-clear-clusters kind-create-cluster test-kind-simple

test-kind-simple: clear-test-cache kind-clear-clusters
	@go test -v ./parse_test.go
	@go test -v ./eval_test.go

test-all: test kind-clear-clusters
	@go test -v ./fixtures/kind/kind_test.go
	@go test -v ./placement_test.go

test-placement: clear-test-cache kind-clear-clusters
	@go test -v ./placement_test.go
