# Go Declarative Testing - Kubernetes

[![Go Reference](https://pkg.go.dev/badge/github.com/gdt-dev/kube.svg)](https://pkg.go.dev/github.com/gdt-dev/kube)
[![Go Report Card](https://goreportcard.com/badge/github.com/gdt-dev/kube)](https://goreportcard.com/report/github.com/gdt-dev/kube)
[![Build Status](https://github.com/gdt-dev/kube/actions/workflows/gate-tests.yml/badge.svg?branch=main)](https://github.com/gdt-dev/kube/actions)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

<div style="float: left">
<img align=left src="static/gdtkubelogo400x544.png" width=200px />
</div>

[`gdt`][gdt] is a testing library that allows test authors to cleanly describe tests
in a YAML file. `gdt` reads YAML files that describe a test's assertions and
then builds a set of Go structures that the standard Go
[`testing`](https://golang.org/pkg/testing/) package can execute.

[gdt]: https://github.com/gdt-dev/gdt

This `github.com/gdt-dev/kube` (shortened hereafter to `gdt-kube`) repository
is a companion Go library for `gdt` that allows test authors to cleanly
describe functional tests of Kubernetes resources and actions using a simple,
clear YAML format. `gdt-kube` parses YAML files that describe Kubernetes
client/API requests and assertions about those client calls.

## Usage

`gdt-kube` is a Go library and is intended to be included in your own Go
application's test code as a Go package dependency.

Import the `gdt` and `gdt-kube` libraries in a Go test file:

```go
import (
    "github.com/gdt-dev/gdt"
    gdtkube "github.com/gdt-dev/kube"
)
```

In a standard Go test function, use the `gdt.From()` function to instantiate a
test object (either a `Scenario` or a `Suite`) that can be `Run()` with a
standard Go `context.Context` and a standard Go `*testing.T` type:

```go
func TestExample(t *testing.T) {
    s, err := gdt.From("path/to/test.yaml")
    if err != nil {
        t.Fatalf("failed to load tests: %s", err)
    }

    ctx := context.Background()
    err = s.Run(ctx, t)
    if err != nil {
        t.Fatalf("failed to run tests: %s", err)
    }
}
```

To execute the tests, just run `go test` per the standard Go testing practice.

`gdt` is a *declarative testing framework* and the meat of your tests is going
to be in the YAML files that describe the actions and assertions for one or
more tests. Read on for an explanation of how to write tests in this
declarative YAML format.

## `gdt-kube` test file structure

A `gdt` test scenario (or just "scenario") is simply a YAML file.

All `gdt` scenarios have the following fields:

* `name`: (optional) string describing the contents of the test file. If
  missing or empty, the filename is used as the name
* `description`: (optional) string with longer description of the test file
  contents
* `defaults`: (optional) is a map, keyed by a plugin name, of default options
  and configuration values for that plugin.
* `fixtures`: (optional) list of strings indicating named fixtures that will be
  started before any of the tests in the file are run
* `tests`: list of [`Spec`][basespec] specializations that represent the
  runnable test units in the test scenario.

[basespec]: https://github.com/gdt-dev/gdt/blob/2791e11105fd3c36d1f11a7d111e089be7cdc84c/types/spec.go#L27-L44

### `gdt-kube` test configuration defaults

To set `gdt-kube`-specific default configuration values for the test scenario,
set the `defaults.kube` field to an object containing any of these fields:

* `defaults.kube.config`: (optional) file path to a `kubeconfig` to use for the
  test scenario.
* `defaults.kube.context`: (optional) string containing the name of the kube
  context to use for the test scenario.
* `defaults.kube.namespace`: (optional) string containing the Kubernetes
  namespace to use when performing some action for the test scenario.

As an example, let's say that I wanted to override the Kubernetes namespace and
the kube context used for a particular test scenario. I would do the following:

```yaml
name: example-test-with-defaults
defaults:
  kube:
    context: my-kube-context
    namespace: my-namespace
```

### `gdt-kube` test spec structure

All `gdt` test specs have the same [base fields][base-spec-fields]:

* `name`: (optional) string describing the test unit.
* `description`: (optional) string with longer description of the test unit.
* `timeout`: (optional) an object containing [timeout information][timeout] for the test
  unit.
* `timeout.after`: a string duration of time the test unit is expected to
  complete within.
* `timeout.expected`: a bool indicating that the test unit is expected to not
  complete before `timeout.after`. This is really only useful in unit testing.
* `wait` (optional) an object containing [wait information][wait] for the test
  unit.
* `wait.before`: a string duration of time that gdt should wait before
  executing the test unit's action.
* `wait.after`: a string duration of time that gdt should wait after executing
  the test unit's action.

[timeout]: https://github.com/gdt-dev/gdt/blob/2791e11105fd3c36d1f11a7d111e089be7cdc84c/types/timeout.go#L11-L22
[wait]: https://github.com/gdt-dev/gdt/blob/2791e11105fd3c36d1f11a7d111e089be7cdc84c/types/wait.go#L11-L25

`gdt-kube` test specs have some additional fields that allow you to take some
action against a Kubernetes API and assert that the response from the API
matches some expectation:

* `config`: (optional) file path to the `kubeconfig` to use for this specific
  test. This allows you to override the `defaults.config` value from the test
  scenario.
* `context`: (optional) string containing the name of the kube context to use
  for this specific test. This allows you to override the `defaults.context`
  value from the test scenario.
* `namespace`: (optional) string containing the name of the Kubernetes
  namespace to use when performing some action for this specific test. This
  allows you to override the `defaults.namespace` value from the test scenario.
* `kube`: (optional) an object containing actions and assertions the test takes
  against the Kubernetes API server.
* `kube.get`: (optional) string containing either a resource specifier (e.g.
  `pods`, `po/nginx` or a file path to a YAML manifest containing resources
  that will be read from the Kubernetes API server.
* `kube.create`: (optional) string containing either a file path to a YAML
  manifest or a string of raw YAML containing the resource(s) to create.
* `kube.apply`: (optional) string containing either a file path to a YAML
  manifest or a string of raw YAML containing the resource(s) for which
  `gdt-kube` will perform a Kubernetes Apply call.
* `kube.delete`: (optional) string containing either a resource specifier (e.g.
  `pods`, `po/nginx` or a file path to a YAML manifest containing resources
  that will be deleted.
* `kube.with`: (optional) object containing selectors with which to filter
  `get` and `delete` operations.
* `kube.with.labels`: (optional) `map[string]string` containing the label keys
  and values to use in constructing an equality label selector (for all listed
  labels)
* `assert`: (optional) object containing assertions to make about the
  action performed by the test.
* `assert.error`: (optional) string to match a returned error from the
  Kubernetes API server.
* `assert.len`: (optional) int with the expected number of items returned.
* `assert.notfound`: (optional) bool indicating the test author expects
  the Kubernetes API to return a 404/Not Found for a resource.
* `assert.unknown`: (optional) bool indicating the test author expects the
  Kubernetes API server to respond that it does not know the type of resource
  attempting to be fetched or created.
* `assert.matches`: (optional) a YAML string, a filepath, or a
  `map[string]interface{}` representing the content that you expect to find in
  the returned result from the `kube.get` call. If `assert.matches` is a
  string, the string can be either a file path to a YAML manifest or
  inline an YAML string containing the resource fields to compare.
  Only fields present in the Matches resource are compared. There is a
  check for existence in the retrieved resource as well as a check that
  the value of the fields match. Only scalar fields are matched entirely.
  In other words, you do not need to specify every field of a struct field
  in order to compare the value of a single field in the nested struct.
* `assert.conditions`: (optional) a map, keyed by `ConditionType` string,
  of any of the following:
  - a string containing the `Status` value that the `Condition` with the
    `ConditionType` should have.
  - a list of strings containing the `Status` value that the `Condition` with
    the `ConditionType` should have.
  - an object containing two fields:
    * `status` which itself is either a single string or a list of strings
      containing the `Status` values that the `Condition` with the
      `ConditionType` should have
    * `reason` which is the exact string that should be present in the
      `Condition` with the `ConditionType`
* `assert.json`: (optional) object describing the assertions to make about
  resource(s) returned from the `kube.get` call to the Kubernetes API server.
* `assert.json.len`: (optional) integer representing the number of bytes in the
  resulting JSON object after successfully parsing the resource.
* `assert.json.paths`: (optional) map of strings where the keys of the map
  are JSONPath expressions and the values of the map are the expected value to
  be found when evaluating the JSONPath expression
* `assert.json.path_formats`: (optional) map of strings where the keys of the map are
  JSONPath expressions and the values of the map are the expected format of the
  value to be found when evaluating the JSONPath expression. See the
  [list of valid format strings](#valid-format-strings)
* `assert.json.schema`: (optional) string containing a filepath to a
  JSONSchema document.  If present, the resource's structure will be validated
  against this JSONSChema document.

## Examples

Here are some examples of `gdt-kube` tests.

Testing that a Pod with the name `nginx` exists:

```yaml
name: test-nginx-pod-exists
tests:
 - kube:
     get: pods/nginx
 # These are equivalent. "kube.get" is a shortcut for the longer object.field
 # form above.
 - kube.get: pods/nginx
```

Testing that a Pod with the name `nginx` *does not* exist:

```yaml
name: test-nginx-pod-not-exist
tests:
 - kube:
     get: pods/nginx
   assert:
     notfound: true
```

Testing that there are two Pods having the label `app:nginx`:

```yaml`
name: list-pods-with-labels
tests:
  - name: verify-pods-with-app-nginx-label
    kube:
      get: pods
      with:
        labels:
          app: nginx
    assert:
      len: 2
```

Testing that a Pod with the name `nginx` exists by the specified timeout
(essentially, `gdt-kube` will retry the get call and assertion until the end of
the timeout):

```yaml
name: test-nginx-pod-exists-within-1-minute
tests:
 - kube:
     get: pods/nginx
     timeout: 1m
```

Testing creation and subsequent fetch then delete of a Pod, specifying the Pod
definition contained in a YAML file:

```yaml
name: create-get-delete-pod
description: create, get and delete a Pod
fixtures:
  - kind
tests:
  - name: create-pod
    kube:
      create: manifests/nginx-pod.yaml
  - name: pod-exists
    kube:
      get: pods/nginx
  - name: delete-pod
    kube:
      delete: pods/nginx
```

Testing creation and subsequent fetch then delete of a Pod, specifying the Pod
definition using an inline YAML blob:

```yaml
name: create-get-delete-pod
description: create, get and delete a Pod
fixtures:
  - kind
tests:
  # "kube.create" is a shortcut for the longer object->field format
  - kube.create: |
        apiVersion: v1
        kind: Pod
        metadata:
          name: nginx
        spec:
          containers:
          - name: nginx
            image: nginx
            imagePullPolicy: IfNotPresent
  # "kube.get" is a shortcut for the longer object->field format
  - kube.get: pods/nginx
  # "kube.delete" is a shortcut for the longer object->field format
  - kube.delete: pods/nginx
```

### Executing arbitrary commands or shell scripts

You can mix other `gdt` test types in a single `gdt` test scenario. For
example, here we are testing the creation of a Pod, waiting a little while with
the `wait.after` directive, then using the `gdt` `exec` test type to test SSH
connectivity to the Pod.

```yaml
name: create-check-ssh
description: create a Deployment then check SSH connectivity
fixtures:
  - kind
tests:
  - kube.create: manifests/deployment.yaml
    wait:
      after: 30s
  - exec: ssh -T someuser@ip
```

### Asserting resource fields using `assert.matches`

The `assert.matches` field of a `gdt-kube` test Spec allows a test author
to specify expected fields and those field contents in a resource that was
returned by the Kubernetes API server from the result of a `kube.get` call.

Suppose you have a Deployment resource and you want to write a test that checks
that a Deployment resource's `Status.ReadyReplicas` field is `2`.

You do not need to specify all other `Deployment.Status` fields like
`Status.Replicas` in order to match the `Status.ReadyReplicas` field value. You
only need to include the `Status.ReadyReplicas` field in the `Matches` value as
these examples demonstrate:

```yaml
tests:
 - name: check deployment's ready replicas is 2
   kube:
     get: deployments/my-deployment
   assert:
     matches: |
       kind: Deployment
       metadata:
         name: my-deployment
       status:
         readyReplicas: 2
```

you don't even need to include the kind and metadata in `assert.matches`.
If missing, no kind and name matching will be performed.

```yaml
tests:
 - name: check deployment's ready replicas is 2
   kube:
     get: deployments/my-deployment
   assert:
     matches: |
       status:
         readyReplicas: 2
```

In fact, you don't need to use an inline multiline YAML string. You can
use a `map[string]interface{}` as well:

```yaml
tests:
 - name: check deployment's ready replicas is 2
   kube:
     get: deployments/my-deployment
   assert:
     matches:
       status:
         readyReplicas: 2
```

### Asserting resource `Conditions` using `assert.conditions`

`assertion.conditions` contains the assertions to make about a resource's
`Status.Conditions` collection. It is a map, keyed by the ConditionType
(matched case-insensitively), of assertions to make about that Condition. The
assertions can be:

* a string which is the ConditionStatus that should be found for that
  Condition
* a list of strings containing ConditionStatuses, any of which should be
  found for that Condition
* an object of type `ConditionExpect` that contains more fine-grained
  assertions about that Condition's Status and Reason

A simple example that asserts that a Pod's `Ready` Condition has a
status of `True`. Note that both the condition type ("Ready") and the
status ("True") are matched case-insensitively, which means you can just
use lowercase strings:

```yaml
tests:
 - kube:
     get: pods/nginx
   assert:
     conditions:
       ready: true
```

If we wanted to assert that the `ContainersReady` Condition had a status
of either `False` or `Unknown`, we could write the test like this:

```yaml
tests:
 - kube:
     get: pods/nginx
   assert:
     conditions:
       containersReady:
        - false
        - unknown
```

Finally, if we wanted to assert that a Deployment's `Progressing`
Condition had a Reason field with a value "NewReplicaSetAvailable"
(matched case-sensitively), we could do the following:

```yaml
tests:
 - kube:
     get: deployments/nginx
   assert:
     conditions:
       progressing:
         status: true
         reason: NewReplicaSetAvailable
```

### Asserting resource fields using `assert.json`

The `assert.json` field of a `gdt-kube` test Spec allows a test author to
specify expected fields, the value of those fields as well as the format of
field values in a resource that was returned by the Kubernetes API server from
the result of a `kube.get` call.

Suppose you have a Deployment resource and you want to write a test that checks
that a Deployment resource's `Status.ReadyReplicas` field is `2`.

You can specify this expectation using the `assert.json.paths` field,
which is a `map[string]interface{}` that takes map keys that are JSONPath
expressions and map values of what the field at that JSONPath expression should
contain:

```yaml
tests:
 - name: check deployment's ready replicas is 2
   kube:
     get: deployments/my-deployment
   assert:
     json:
       paths:
         $.status.readyReplicas: 2 
```

JSONPath expressions can be fairly complex, allowing the test author to, for
example, assert the value of a nested map field with a particular key, as this
example shows:

```yaml
tests:
 - name: check deployment's pod template "app" label is "nginx"
   kube:
     get: deployments/my-deployment
   assert:
     json:
       paths:
         $.spec.template.labels["app"]: nginx
```

You can check that the value of a particular field at a JSONPath is formatted
in a particular fashion using `assert.json.path_formats`. This is a map,
keyed by JSONPath expression, of the data format the value of the field at that
JSONPath expression should have. Valid data formats are:

* `date`
* `date-time`
* `email`
* `hostname`
* `idn-email`
* `ipv4`
* `ipv6`
* `iri`
* `iri-reference`
* `json-pointer`
* `regex`
* `relative-json-pointer`
* `time`
* `uri`
* `uri-reference`
* `uri-template`
* `uuid`
* `uuid4`

[Read more about JSONSchema formats](https://json-schema.org/understanding-json-schema/reference/string.html#built-in-formats).

For example, suppose we wanted to verify that a Deployment's `metadata.uid`
field was a UUID-4 and that its `metadata.creationTimestamp` field was a
date-time timestamp:

```yaml
tests:
  - kube:
      get: deployments/nginx
    assert:
      json:
        path_formats:
          $.metadata.uid: uuid4
          $.metadata.creationTimestamp: date-time
```

### Updating a resource and asserting corresponding field changes

Here is an example of creating a Deployment with an initial `spec.replicas`
count of 2, then applying a change to `spec.replicas` of 1, then asserting that
the `status.readyReplicas` gets updated to 1.

file `testdata/manifests/nginx-deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  selector:
    matchLabels:
      app: nginx
  replicas: 2
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
```

file `testdata/apply-deployment.yaml`:

```yaml
name: apply-deployment
description: create, get, apply a change, get, delete a Deployment
fixtures:
  - kind
tests:
  - name: create-deployment
    kube:
      create: testdata/manifests/nginx-deployment.yaml
  - name: deployment-has-2-replicas
    timeout:
      after: 20s
    kube:
      get: deployments/nginx
    assert:
      matches:
        status:
          readyReplicas: 2
  - name: apply-deployment-change
    kube:
      apply: |
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: nginx
        spec:
          replicas: 1
  - name: deployment-has-1-replica
    timeout:
      after: 20s
    kube:
      get: deployments/nginx
    assert:
      matches:
        status:
          readyReplicas: 1
  - name: delete-deployment
    kube:
      delete: deployments/nginx
```

### Timeouts and retrying `kube.get` assertions

When evaluating assertions for `kube.get`, `gdt` inspects the test's
`timeout.after` value to determine how long to retry the `get` call and recheck
the assertions.

If a test's `timeout.after` is empty, `gdt` inspects the scenario's
`defaults.timeout.after` value. If both of those values are empty, `gdt` will
use a **default timeout of 5 seconds**.

If you're interested in seeing the individual results of `gdt`'s
assertion-checks for a single `get` call, you can use the `gdt.WithDebug()`
function, like this test function demonstrates:

file: `testdata/matches.yaml`:

```yaml
name: matches
description: create a deployment and check the matches condition succeeds
fixtures:
  - kind
tests:
  - name: create-deployment
    kube:
      create: testdata/manifests/nginx-deployment.yaml
  - name: deployment-exists
    kube:
      get: deployments/nginx
    assert:
      matches:
        spec:
          replicas: 2
          template:
            metadata:
              labels:
                app: nginx
        status:
          readyReplicas: 2
  - name: delete-deployment
    kube:
      delete: deployments/nginx
```

file: `matches_test.go`

```go
import (
    "github.com/gdt-dev/gdt"
    _ "github.com/gdt-dev/kube"
    kindfix "github.com/gdt-dev/kube/fixture/kind"
)

func TestMatches(t *testing.T) {
	fp := filepath.Join("testdata", "matches.yaml")

	kfix := kindfix.New()

	s, err := gdt.From(fp)

	ctx := gdt.NewContext(gdt.WithDebug())
	ctx = gdt.RegisterFixture(ctx, "kind", kfix)
	s.Run(ctx, t)
}
```

Here's what running `go test -v matches_test.go` would look like:

```
$ go test -v matches_test.go
=== RUN   TestMatches
=== RUN   TestMatches/matches
=== RUN   TestMatches/matches/create-deployment
=== RUN   TestMatches/matches/deployment-exists
deployment-exists (try 1 after 1.303µs) ok: false, terminal: false
deployment-exists (try 1 after 1.303µs) failure: assertion failed: match field not equal: $.status.readyReplicas not present in subject
deployment-exists (try 2 after 595.62786ms) ok: false, terminal: false
deployment-exists (try 2 after 595.62786ms) failure: assertion failed: match field not equal: $.status.readyReplicas not present in subject
deployment-exists (try 3 after 1.020003807s) ok: false, terminal: false
deployment-exists (try 3 after 1.020003807s) failure: assertion failed: match field not equal: $.status.readyReplicas not present in subject
deployment-exists (try 4 after 1.760006109s) ok: false, terminal: false
deployment-exists (try 4 after 1.760006109s) failure: assertion failed: match field not equal: $.status.readyReplicas had different values. expected 2 but found 1
deployment-exists (try 5 after 2.772416449s) ok: true, terminal: false
=== RUN   TestMatches/matches/delete-deployment
--- PASS: TestMatches (3.32s)
    --- PASS: TestMatches/matches (3.30s)
        --- PASS: TestMatches/matches/create-deployment (0.01s)
        --- PASS: TestMatches/matches/deployment-exists (2.78s)
        --- PASS: TestMatches/matches/delete-deployment (0.02s)
PASS
ok  	command-line-arguments	3.683s
```

You can see from the debug output above that `gdt` created the Deployment and
then did a `kube.get` for the `deployments/nginx` Deployment. Initially
(attempt 1), the `assert.matches` assertion failed because the
`status.readyReplicas` field was not present in the returned resource. `gdt`
retried the `kube.get` call 4 more times (attempts 2-5), with attempts 2 and 3
failed the existence check for the `status.readyReplicas` field and attempt 4
failing the *value* check for the `status.readyReplicas` field being `1`
instead of the expected `2`. Finally, when the Deployment was completely rolled
out, attempt 5 succeeded in all the `assert.matches` assertions.

## Determining Kubernetes config, context and namespace values

When evaluating how to construct a Kubernetes client `gdt-kube` uses the following
precedence to determine the `kubeconfig` and kube context:

1) The individual test spec's `config` or `context` value
2) Any `gdt` Fixture that exposes a `gdt.kube.config` or `gdt.kube.context`
   state key (e.g. [`KindFixture`][kind-fixture]).
3) The test file's `defaults.kube` `config` or `context` value.

For the `kubeconfig` file path, if none of the above yielded a value, the
following precedence is used to determine the `kubeconfig`:

4) A non-empty `KUBECONFIG` environment variable pointing at a file.
5) In-cluster config if running in cluster.
6) `$HOME/.kube/config` if it exists.

[kube-fixture]: https://github.com/gdt-dev/kube/blob/main/fixtures/kind/kind.go

## `gdt-kube` Fixtures

`gdt` Fixtures are objects that help set up and tear down a testing
environment. The `gdt-kube` library has some utility fixtures to make testing
with Kubernetes easier.

### `KindFixture`

The `KindFixture` eases integration of `gdt-kube` tests with the KinD local
Kubernetes development system.

To use it, import the `gdt-kube/fixtures/kind` package:

```go
import (
    "github.com/gdt-dev/gdt"
    gdtkube "github.com/gdt-dev/kube"
    gdtkind "github.com/gdt-dev/kube/fixtures/kind"
)
```

and then register the fixture with your `gdt` `Context`, like so:

```go
func TestExample(t *testing.T) {
    s, err := gdt.From("path/to/test.yaml")
    if err != nil {
        t.Fatalf("failed to load tests: %s", err)
    }

    ctx := context.Background()
    ctx = gdt.RegisterFixture(ctx, "kind", gdtkind.New())
    err = s.Run(ctx, t)
    if err != nil {
        t.Fatalf("failed to run tests: %s", err)
    }
}
```

In your test file, you would list the "kind" fixture in the `fixtures` list:

```yaml
name: example-using-kind
fixtures:
 - kind
tests:
 - kube.get: pods/nginx
```

## Contributing and acknowledgements

`gdt` was inspired by [Gabbi](https://github.com/cdent/gabbi), the excellent
Python declarative testing framework. `gdt` tries to bring the same clear,
concise test definitions to the world of Go functional testing.

The Go gopher logo, from which gdt's logo was derived, was created by Renee
French.

Contributions to `gdt-kube` are welcomed! Feel free to open a Github issue or
submit a pull request.
