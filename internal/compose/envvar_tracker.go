package compose

import (
	"maps"
	"os"
	"strings"
)

// EnvVarTracker is used to provide the `environment` resource type. This tracks what keys are accessed and replaces
// them with outputs that are environment variable references that docker compose will support.
// This keeps track of which keys were accessed so that we can produce a reference file or list of keys for the user
// to understand what inputs docker compose will require at launch time.
type EnvVarTracker struct {
	// lookup is an environment variable lookup function, if nil this will be defaulted to os.LookupEnv
	lookup func(key string) (string, bool)
	// accessed is the map of accessed environment variables and the value they had at access time
	accessed map[string]string
}

func (e *EnvVarTracker) Accessed() map[string]string {
	return maps.Clone(e.accessed)
}

// the env var tracker is a resource itself (an environment resource)
var _ ResourceWithOutputs = (*EnvVarTracker)(nil)

func (e *EnvVarTracker) lookupOutput(required bool, keys ...string) (interface{}, error) {
	if len(keys) == 0 {
		panic("requires at least 1 key")
	}
	envVarKey := strings.ToUpper(strings.Join(keys, "_"))

	// in theory we can replace more unexpected characters
	envVarKey = strings.ReplaceAll(envVarKey, "-", "_")
	envVarKey = strings.ReplaceAll(envVarKey, ".", "_")

	if e.lookup == nil {
		e.lookup = os.LookupEnv
	}
	if e.accessed == nil {
		e.accessed = make(map[string]string, 1)
	}

	if v, ok := e.lookup(envVarKey); ok {
		e.accessed[envVarKey] = v
	} else {
		e.accessed[envVarKey] = ""
	}
	if required {
		envVarKey += "?required"
	}
	return "${" + envVarKey + "}", nil
}

func (e *EnvVarTracker) LookupOutput(keys ...string) (interface{}, error) {
	return e.lookupOutput(false, keys...)
}

func (e *EnvVarTracker) GenerateResource(resName string) ResourceWithOutputs {
	return &envVarResourceTracker{
		inner:  e,
		prefix: resName,
	}
}

// envVarResourceTracker is a child object of EnvVarTracker and is used as a fallback behavior for resource types
// that are not supported natively: we treat them like environment variables instead with a prefix of the resource name.
type envVarResourceTracker struct {
	prefix string
	inner  *EnvVarTracker
}

func (e *envVarResourceTracker) LookupOutput(keys ...string) (interface{}, error) {
	next := make([]string, 1+len(keys))
	next[0] = e.prefix
	for i, k := range keys {
		next[1+i] = k
	}
	return e.inner.lookupOutput(true, next...)
}
