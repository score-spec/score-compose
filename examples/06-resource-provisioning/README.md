# 06 - Resource Provisioning

The previous examples showed some of the basic resource provisioners for `environment` and `volume` resources. `score-compose` also provides built-in support for some more interesting resources as well as a flexible provisioning system that can be customised.

In the following examples, we're using `type: redis` which provisioners a single-container Redis node.

Each resource is independent of others, for example here the `cacheA`, `cacheB`, and `cacheC` Redis instances are independent:

```yaml
metadata:
  name: workload-one
...
resources:
  cacheA:
    type: redis
    
---
metadata:
  name: workload-two
...
resources:
  cacheB:
    type: redis
  cacheC:
    type: redis
```

In some stateful cases, you may need some resources to be shared. This can be done by adding `id: specific-id` to the resource definition. This is unique to the project and shared across workloads. For example, below we share the same cache across both workloads, while an extra independent cache is added to the second workload.

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: workload-one
containers:
  example:
    image: busybox
    command: ["/bin/sh"]
    args: ["-c", "while true; do echo $${CONNECTION}; sleep 5; done"]
    variables:
      CONNECTION: "redis://${resources.cache-a.username}:${resources.cache-a.password}@${resources.cache-a.host}:${resources.cache-a.port}"
resources:
  cache-a:
    type: redis
    id: main-cache
    
---
apiVersion: score.dev/v1b1
metadata:
  name: workload-two
containers:
  example:
    image: busybox
    command: ["/bin/sh"]
    args: ["-c", "while true; do echo $${CONNECTION_A}; echo $${CONNECTION_B}; sleep 5; done"]
    variables:
      CONNECTION_A: "redis://${resources.cache-b.username}:${resources.cache-b.password}@${resources.cache-b.host}:${resources.cache-b.port}"
      CONNECTION_B: "redis://${resources.cache-c.username}:${resources.cache-c.password}@${resources.cache-c.host}:${resources.cache-c.port}"
resources:
  cache-b:
    type: redis
    id: main-cache
  cache-c:
    type: redis
```

Notice how we are using the placeholder syntax to access outputs from the resources and pass these through to the workloads. The syntax uses a `.`-separated path to traverse the resource outputs. The `.` can be escaped inside the placeholder with a backslash, for example: `${resources.cache-a.some\.key}`.

When we provision this with `score-compose` we get an output that shows 2 redis services being created and both workloads have access to the connection strings. This supports applications that need to share data within the same caches, databases, or other stateful resources.

```console
$ score-compose init
$ score-compose generate score.yaml
$ score-compose generate score2.yaml
$ docker compose up
[+] Running 5/5
 ✔ Container 06-resource-provisioning-redis-MSFdjH-1          Created
 ✔ Container 06-resource-provisioning-redis-2tKndj-1          Created
 ✔ Container 06-resource-provisioning-wait-for-resources-1    Created
 ✔ Container 06-resource-provisioning-workload-two-example-1  Created
 ✔ Container 06-resource-provisioning-workload-one-example-1  Created   
Attaching to redis-2tKndj-1, redis-MSFdjH-1, wait-for-resources-1, workload-one-example-1, workload-two-example-1
wait-for-resources-1    |
wait-for-resources-1 exited with code 0
workload-two-example-1  | redis://default:94GBXwsRLtDLagPg@redis-2tKndj:6379
workload-two-example-1  | redis://default:v2otaQdY6052ylWq@redis-MSFdjH:6379
workload-one-example-1  | redis://default:94GBXwsRLtDLagPg@redis-2tKndj:6379
```

## The `*.provisioners.yaml` files

When you run `score-compose init`, a [99-default.provisioners.yaml](https://github.com/score-spec/score-compose/blob/main/internal/command/default.provisioners.yaml) file is created, which is a YAML file holding the definition of the built-in provisioners.

When you run `score-compose generate`, all `*.provisioners.yaml` files are loaded in lexicographic order from the `.score-compose` directory. This allows projects to include their own custom provisioners that extend or override the defaults.

Each entry in the file has the following common fields, other fields may also exist for specific provisioner types.

```
- uri: <provisioner uri>
  type: <resource type>
  class: <optional resource class>
  id: <optional resource id>
```

The uri of each provisioner is a combination of it's implementation (either `template://` or `cmd://`) and a unique identifier.

### The `template://` provisioner

Most built in provisioners are implemented as a series of Go templates using the template provisioner. The implementation can be found [here](https://github.com/score-spec/score-compose/blob/bf87309396d30e155d7d503b2fb917252e039278/internal/provisioners/templateprov/template.go). The Go template engine is [text/template](https://pkg.go.dev/text/template).

The following extra fields can be configured as required on each instance of this provisioner:

| Field         | Type                | Comment                                                                                                                                                                                                                      |
|---------------|---------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `init`        | String, Go template | A Go template for a valid YAML dictionary. The values here will be provided to the next templates as the `.Init` state.                                                                                                      |
| `state`       | String, Go template | A Go template for a valid YAML dictionary. The values here will be persisted into the state file and made available to future executions and are provided to the next templates as the `.State` state.                       |
| `shared`      | String, Go template | A Go template for a valid YAML dictionary. The values here will be _merged_ using a JSON-patch mechanism with the current shared state across all resources and made available to future executions through `.Shared` state. |
| `outputs`     | String, Go template | A Go template for a valid YAML dictionary. The values here are the outputs of the resource that can be accessed through `${resources.*}` placeholder resolution.                                                             |
| `directories` | String, Go template | A Go template for a valid YAML dictionary. Each path -> bool mapping will create (true) or delete (false) a directory relative to the mounts directory.                                                                      |
| `files`       | String, Go template | A Go template for a valid YAML dictionary. Each path -> string\|null will create a relative file (string) or delete it (null) relative to the mounts directory.                                                              |
| `networks`    | String, Go template | A Go template for a valid set of named Compose [Networks](https://github.com/compose-spec/compose-spec/blob/master/06-networks.md). These will be added to the output project.                                               |
| `volumes`     | String, Go template | A Go template for a valid set of named Compose [Volumes](https://github.com/compose-spec/compose-spec/blob/master/07-volumes.md).                                                                                            |
| `services`    | String, Go template | A Go template for a valid set of named Compose [Services](https://github.com/compose-spec/compose-spec/blob/master/05-services.md).                                                                                          |

Each template has access to the [Sprig](http://masterminds.github.io/sprig/) functions library and executes with access to the following structure:

```go
type Data struct {
	Uid      string
	Type     string
	Class    string
	Id       string
	Params   map[string]interface{}
	Metadata map[string]interface{}
	Init   map[string]interface{}
	State  map[string]interface{}
	Shared map[string]interface{}
	WorkloadServices map[string]NetworkService
	ComposeProjectName string
	MountsDirectory    string
}
```

Browse the default provisioners for inspiration or more clues to how these work!

### The `cmd://` provisioner

The command provisioner implementation can be used to execute an external binary or script to provision the resource. The provision IO structures are serialised to json and send on standard-input to the new process, any stdout content is decoded as json and is used as the outputs of the provisioner.

The uri of the provisioner encodes the binary to be executed:

- `cmd://python` will execute the `python` binary on the PATH
- `cmd://../my-script` will execute `../my-script`
- `cmd://./my-script` will execute `my-script` in the current directory
- and `cmd://~/my-script` will execute the `my-script` binary in the home directory

Additional arguments can be provided via the `args` configuration key, for example a basic provisioner can be created using python inline scripts:

```
- uri: "cmd://python"
  args: ["-c", "print({\"resource_outputs\":{}})"]
```

The JSON structures are the `Input` and `ProvisionOutput` structures in [internal/provisioners/core.go](https://github.com/score-spec/score-compose/blob/3cee56c624a70821a55af44a15513ebf8b594f9a/internal/provisioners/core.go#L35).
