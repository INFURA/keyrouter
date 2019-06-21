# keyrouter

```
usage: keyrouter [<flags>]

A simple microservice for consistent hashing of service entries

Flags:
  --help                    Show context-sensitive help (also try --help-long and --help-man).
  --services=services.toml  location of services.toml
  --address=:8080           address to bind to
  --log-level="INFO"        minimum log level
  --version                 Show application version.
```

This is a simple service that maps keys to one or more service endpoint nodes in a consistent manner.  Here _consistent_
means that as the number of available nodes changes over time, the majority of keys should  continue to map the same 
nodes.  Thus, these nodes can be used for "sticky sessions" or sharding a key space.  Multiple nodes can be returned to
support shard replication or similar scenarios. 

## Services.toml file

Services should be defined in a `services.toml` file, of the form:

```toml
[[Services]]
  Name = "foo"
  Nodes = [
    "foo1:8080",
    "foo2:8080",
  ]

[[Services]]
  Name = "bar"
  Nodes = [
   "bar1:8080",
   "bar2:8080",
  ]
```

The values in `Nodes` are not validated in any way, but are typically service endpoints.

### Reload on SIGHUP

The `services.toml` file is reloaded when receiving the `SIGHUP` signal, and added or removed
entries for each service are updated.  

## Consul-Template Usage

This microservice is meant to be fed an up-to-date `services.toml` via a service discovery system; we suggest using 
`consul-template`.  `services.toml.tmpl` is a very cut and dry consul-template template that would render _all_ services
registered with Consul.  This can be used like:

`consul-template -template services.toml.tmpl:/tmp/services.toml -exec keyrouter --services /tmp/services.toml -exec-reload-signal SIGHUP`

This will run consul-template in daemon mode, which will in turn exec `keyrouter` and send it a `SIGHUP` whenever the services.toml file changes.

## Requesting Consistent Service Entries

To get a set of consistent entries from a service, `POST` a request to `/service/:servicename` with the following arguments:

- `key`: The key to hash on.
- `min`: The minimum number of entries to return
- `max`: The maximum number of entries to return

Arguments can be sent as query strings, form data, or JSON.  All three of these examples are equivalent:

- `curl -v 'http://:8080/service/bar?key=ryans&min=1&max=3'`
- `curl -v http://:8080/service/bar -F "key=ryans" -F "min=1" -F "max=3"`
- `curl -v http://:8080/service/bar -H "Content-Type: application/json" --data '{"key":"ryans","min":1,"max":3}'`

The response is a JSON-encoded array of between `min` and `max` members, which represent the N members "closest" to the 
input `key` in the service hash ring:

```text
curl http://:8080/service/bar -H "Content-Type: application/json" --data '{"key":"k","min":1,"max":3}'
["bar1:8080","bar2:8080"]
```

Importantly different input `key` values will return the services in a different, but consistent order.  If the 
available nodes for a service change over time, keys should largely stay mapped to the same nodes.





