VERSION ?= $(shell git describe --tags --always --dirty)

.PHONY: test

# We clear the test cache because some of the tests require an out-of-band KinD
# cluster to run against and we want to re-run tests against that KinD cluster
# instead of from cached unit test results.
test:
	@go clean -testcache
	@go test -v ./...
