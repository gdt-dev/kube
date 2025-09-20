// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	gdtcontext "github.com/gdt-dev/core/context"
	"github.com/gdt-dev/core/debug"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// genericCondition contains fields that are (mostly) common to many Condition
// objects and that we wish to match against.
type genericCondition struct {
	Type   string
	Status string
	Reason string
}

// conditionFound returns a delta describing the differences found between a
// supplied resource's Conditions and the expected Conditions
func compareConditions(
	res *unstructured.Unstructured,
	expected map[string]*ConditionMatch,
) *delta {
	d := &delta{differences: []string{}}
	conds, found, err := unstructured.NestedSlice(res.Object, "status", "conditions")
	if found && err != nil {
		// this means the resource's Status.Conditions is not a slice of
		// something... which is weird and unexpected, so just panic.
		msg := fmt.Sprintf(
			"found a resource %q with a non-slice Status.Conditions field",
			res.GetKind(),
		)
		panic(msg)
	}
	if (!found || len(conds) == 0) && len(expected) != 0 {
		for condType := range expected {
			d.Add(fmt.Sprintf("no condition with type %q found", condType))
		}
		return d
	}
	// construct a map, keyed by condition type, of the condition fields from
	// the resource so we can do type-based lookups easier.
	gcs := map[string]genericCondition{}
	for _, condAny := range conds {
		condMap, ok := condAny.(map[string]any)
		if !ok {
			// this means the resource's Status.Conditions is not a slice of
			// map[string]interface... which is also weird and unexpected, so
			// just panic.
			msg := fmt.Sprintf(
				"found a resource %q with a non-map[string]any "+
					"Status.Conditions member type: %T",
				res.GetKind(), condAny,
			)
			panic(msg)
		}
		gc := genericCondition{}
		for k, v := range condMap {
			klow := strings.ToLower(k)
			switch klow {
			case "type":
				gc.Type = strings.ToLower(v.(string))
			case "reason":
				gc.Reason = v.(string)
			case "status":
				gc.Status = strings.ToLower(v.(string))
			}
		}
		gcs[gc.Type] = gc

	}
	for condType, condMatch := range expected {
		ctlow := strings.ToLower(condType)
		gc, found := gcs[ctlow]
		if !found {
			d.Add(fmt.Sprintf("no condition with type %q found", condType))
			continue
		}
		if condMatch.Status != nil {
			statusValues := condMatch.Status.Values()
			if gc.Status == "" {
				msg := fmt.Sprintf(
					"condition %q had no status. "+
						"expected status to be one of %s",
					condType, statusValues,
				)
				d.Add(msg)
				continue
			}
			svlow := []string{}
			for _, sv := range statusValues {
				svlow = append(svlow, strings.ToLower(sv))
			}
			if !lo.Contains(svlow, strings.ToLower(gc.Status)) {
				msg := fmt.Sprintf(
					"condition %q had status of %q. "+
						"expected status to be one of %s",
					condType, gc.Status, statusValues,
				)
				d.Add(msg)
				continue
			}
		}
		if condMatch.Reason != "" {
			if gc.Reason != condMatch.Reason {
				msg := fmt.Sprintf(
					"condition %q had reason of %q. "+
						"expected reason to be %q",
					condType, gc.Reason, condMatch.Reason,
				)
				d.Add(msg)
				continue
			}
		}
	}
	return d
}

// matchObjectFromAny returns a map[string]any given any of a filepath,
// an inline YAML string or a map[string]any. The returned
// map[string]any is the collection of resource fields that we will
// match against.
func matchObjectFromAny(
	ctx context.Context,
	m any,
) map[string]any {
	var raw map[string]any
	switch m := m.(type) {
	case string:
		var err error
		var b []byte
		v := m
		if probablyFilePath(v) {
			b, err = os.ReadFile(v)
			if err != nil {
				// NOTE(jaypipes): We already validated that the file exists at
				// parse time. If we get an error here, just panic cuz there's
				// nothing we can really do.
				panic(err)
			}
		} else {
			b = []byte(v)
		}
		var obj map[string]any
		if err = yaml.Unmarshal(b, &obj); err != nil {
			// NOTE(jaypipes): We already validated that the content could be
			// unmarshaled at parse time. If we get an error here, just panic
			// cuz there's nothing we can really do.
			panic(err)
		}
		raw = obj
	case map[string]any:
		raw = m
	}
	// We need to replace any variable references in the match keys or values
	// with the variable values from stored run data
	return lo.MapEntries(raw, func(k string, v any) (string, any) {
		return replaceVariablesInMapEntry(ctx, k, v)
	})
}

func replaceVariablesInMapEntry(
	ctx context.Context,
	k string,
	entry any,
) (string, any) {
	kRep := gdtcontext.ReplaceVariables(ctx, k)
	if k != kRep {
		debug.Printf(
			ctx,
			"kube.assert: replaced match key: %s -> %s",
			k, kRep,
		)
	}
	switch entry := entry.(type) {
	case string:
		entryRep := gdtcontext.ReplaceVariables(ctx, entry)
		if entry != entryRep {
			debug.Printf(
				ctx,
				"kube.assert: replaced match key %s value: %s -> %s",
				k, entry, entryRep,
			)
		}
		return kRep, entryRep
	case map[string]any:
		entry = lo.MapEntries(entry, func(k string, v any) (string, any) {
			return replaceVariablesInMapEntry(ctx, k, v)
		})
		return kRep, entry
	}
	return k, entry
}

// delta collects differences between two objects.
type delta struct {
	differences []string
}

func (d *delta) Add(diff string) {
	d.differences = append(d.differences, diff)
}

func (d *delta) Empty() bool {
	return len(d.differences) == 0
}

func (d *delta) Differences() []string {
	return d.differences
}

// compareResourceToMatchObject returns a delta object containing and
// differences between the supplied resource and the match object.
func compareResourceToMatchObject(
	res *unstructured.Unstructured,
	match map[string]any,
) *delta {
	d := &delta{differences: []string{}}
	collectFieldDifferences("$", match, res.Object, d)
	return d
}

// collectFieldDifferences compares two things and adds any differences between
// them to a supplied set of differences.
func collectFieldDifferences(
	fp string, // the "field path" to the field we are comparing...
	match any,
	subject any,
	delta *delta,
) {
	if !typesComparable(match, subject) {
		diff := fmt.Sprintf(
			"%s non-comparable types: %T and %T.",
			fp, match, subject,
		)
		delta.Add(diff)
		return
	}
	switch match.(type) {
	case map[string]any:
		matchmap := match.(map[string]any)
		subjectmap := subject.(map[string]any)
		for matchk, matchv := range matchmap {
			subjectv, ok := subjectmap[matchk]
			newfp := fp + "." + matchk
			if !ok {
				diff := fmt.Sprintf("%s not present in subject", newfp)
				delta.Add(diff)
				continue
			}
			collectFieldDifferences(newfp, matchv, subjectv, delta)
		}
		return
	case []any:
		matchlist := match.([]any)
		subjectlist := subject.([]any)
		if len(matchlist) != len(subjectlist) {
			diff := fmt.Sprintf(
				"%s had different lengths. expected %d but found %d",
				fp, len(matchlist), len(subjectlist),
			)
			delta.Add(diff)
			return
		}
		// Sort order currently matters, unfortunately...
		for x, matchv := range matchlist {
			subjectv := subjectlist[x]
			newfp := fmt.Sprintf("%s[%d]", fp, x)
			collectFieldDifferences(newfp, matchv, subjectv, delta)
		}
		return
	case int, int8, int16, int32, int64:
		switch subject := subject.(type) {
		case int, int8, int16, int32, int64:
			mv := toInt64(match)
			sv := toInt64(subject)
			if mv != sv {
				diff := fmt.Sprintf(
					"%s had different values. expected %v but found %v",
					fp, match, subject,
				)
				delta.Add(diff)
			}
		case uint, uint8, uint16, uint32, uint64:
			mv := toUint64(match)
			sv := toUint64(subject)
			if mv != sv {
				diff := fmt.Sprintf(
					"%s had different values. expected %v but found %v",
					fp, match, subject,
				)
				delta.Add(diff)
			}
		case string:
			mv := toInt64(match)
			sv, err := strconv.Atoi(subject)
			if err != nil {
				diff := fmt.Sprintf(
					"%s had different values. expected %v but found %v",
					fp, match, subject,
				)
				delta.Add(diff)
				return
			}
			if mv != int64(sv) {
				diff := fmt.Sprintf(
					"%s had different values. expected %v but found %v",
					fp, match, subject,
				)
				delta.Add(diff)
			}
		}
		return
	case string:
		switch subject.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64:
			mv := match.(string)
			si := subject.(int)
			sv := strconv.Itoa(si)
			if mv != sv {
				diff := fmt.Sprintf(
					"%s had different values. expected %v but found %v",
					fp, match, subject,
				)
				delta.Add(diff)
			}
		case string:
			mv, _ := match.(string)
			if mv != subject {
				diff := fmt.Sprintf(
					"%s had different values. expected %v but found %v",
					fp, match, subject,
				)
				delta.Add(diff)
			}
		}
		return
	}
	if !reflect.DeepEqual(match, subject) {
		diff := fmt.Sprintf(
			"%s had different values. expected %v but found %v",
			fp, match, subject,
		)
		delta.Add(diff)
	}
}

// typesComparable returns true if the two supplied things are comparable,
// false otherwise
func typesComparable(a, b any) bool {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)
	at := av.Kind()
	bt := bv.Kind()
	switch at {
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		switch bt {
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64,
			reflect.String:
			return true
		default:
			return false
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
		switch bt {
		case reflect.Uint, reflect.Uint8, reflect.Uint32,
			reflect.Uint64, reflect.String:
			return true
		default:
			return false
		}
	case reflect.Complex64, reflect.Complex128:
		switch bt {
		case reflect.Complex64, reflect.Complex128, reflect.String:
			return true
		default:
			return false
		}
	case reflect.String:
		switch bt {
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64,
			reflect.Complex64, reflect.Complex128, reflect.String:
			return true
		default:
			return false
		}
	}
	return reflect.TypeOf(a) == reflect.TypeOf(b)
}

// toUint64 takes an interface and returns a uint64
func toUint64(v any) uint64 {
	switch v := v.(type) {
	case uint64:
		return v
	case uint8:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint32:
		return uint64(v)
	case uint:
		return uint64(v)
	}
	return 0
}

// toInt64 takes an interface and returns an int64
func toInt64(v any) int64 {
	switch v := v.(type) {
	case int64:
		return v
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int:
		return int64(v)
	}
	return 0
}
