package compose

import (
	"maps"
	"os"
	"strings"
)

type EnvVarTracker struct {
	lookup   func(key string) (string, bool)
	accessed map[string]string
}

func NewEnvVarTracker() *EnvVarTracker {
	return &EnvVarTracker{
		lookup:   os.LookupEnv,
		accessed: make(map[string]string),
	}
}

func (e *EnvVarTracker) Accessed() map[string]string {
	return maps.Clone(e.accessed)
}

// the env var tracker is a resource itself (an environment resource)
var _ ResourceWithOutputs = (*EnvVarTracker)(nil)

func (e *EnvVarTracker) LookupOutput(keys ...string) (interface{}, error) {
	if len(keys) == 0 {
		panic("requires at least 1 key")
	}
	envVarKey := strings.ToUpper(strings.Join(keys, "_"))
	envVarKey = strings.ReplaceAll(envVarKey, "-", "_")
	if v, ok := e.lookup(envVarKey); ok {
		e.accessed[envVarKey] = v
	} else {
		e.accessed[envVarKey] = ""
	}
	return "${" + envVarKey + "}", nil
}

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
	return e.inner.LookupOutput(next...)
}

func (e *EnvVarTracker) GenerateResource(resName string) ResourceWithOutputs {
	return &envVarResourceTracker{
		inner:  e,
		prefix: resName,
	}
}
