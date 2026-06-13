// Package versiongate enforces the per-service API version policy: if a service
// changes its contract, it must bump its spec info.version. This is the only
// versioning the delivery automation enforces (layer 2 of the version model).
//
// The gate compares the current per-service digests against the set last
// delivered (read from labels on the live KrakenD config object). It never
// classifies breaking vs non-breaking changes — it only requires *a* version
// increase on *any* contract change. Semver discipline beyond that is on the
// service author.
package versiongate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
)

// Digest is what the gate tracks per service.
type Digest struct {
	Version      string `json:"version"` // the service's spec info.version
	ContractHash string `json:"hash"`    // hash of the contract (paths + schemas)
}

// ContractHash returns a stable hash of the contract-relevant parts of an
// OpenAPI spec — its paths and component schemas. info.version is deliberately
// excluded so the hash changes only when the contract itself changes, which is
// what lets the gate detect "contract changed but version not bumped". Go's
// json.Marshal sorts map keys, so the encoding is deterministic.
func ContractHash(spec map[string]interface{}) string {
	subset := map[string]interface{}{"paths": spec["paths"]}
	if comps, ok := spec["components"].(map[string]interface{}); ok {
		if schemas, ok := comps["schemas"]; ok {
			subset["schemas"] = schemas
		}
	}
	b, _ := json.Marshal(subset)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Violation reports a service whose contract changed without a version increase.
type Violation struct {
	Service    string
	OldVersion string
	NewVersion string
}

// Check compares the current per-service digests against those last delivered.
// It returns a violation for every service whose contract hash changed without
// its version strictly increasing. New services (absent from lastDelivered) are
// not gated — their first delivery establishes the baseline. An empty result
// means delivery may proceed.
func Check(current, lastDelivered map[string]Digest) []Violation {
	var violations []Violation
	for name, cur := range current {
		last, existed := lastDelivered[name]
		if !existed {
			continue // first delivery of this service
		}
		if cur.ContractHash == last.ContractHash {
			continue // contract unchanged
		}
		if compareVersions(cur.Version, last.Version) <= 0 {
			violations = append(violations, Violation{
				Service:    name,
				OldVersion: last.Version,
				NewVersion: cur.Version,
			})
		}
	}
	return violations
}

// compareVersions compares dotted-numeric versions with an optional leading 'v'
// (e.g. "v1", "1.2.3"). Components are compared numerically when both are
// numeric, otherwise lexically; a missing component counts as 0 ("1.0" == "1.0.0").
// Returns -1 if a<b, 0 if equal, 1 if a>b.
func compareVersions(a, b string) int {
	as := splitVersion(a)
	bs := splitVersion(b)
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		ac := component(as, i)
		bc := component(bs, i)
		ai, aErr := strconv.Atoi(ac)
		bi, bErr := strconv.Atoi(bc)
		if aErr == nil && bErr == nil {
			if ai != bi {
				if ai < bi {
					return -1
				}
				return 1
			}
			continue
		}
		if ac != bc {
			if ac < bc {
				return -1
			}
			return 1
		}
	}
	return 0
}

func splitVersion(v string) []string {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v = strings.TrimPrefix(v, "V")
	if v == "" {
		return nil
	}
	return strings.Split(v, ".")
}

func component(parts []string, i int) string {
	if i < len(parts) {
		return parts[i]
	}
	return "0"
}
