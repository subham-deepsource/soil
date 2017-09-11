---
title: API
layout: default
weight: 300
---

# Agent API

The main interface to Soil Agent is a RESTful HTTP API. The API can query the current state of the system as well as modify the state of the system.

## Version

All API routes are prefixed with `/v1/`.

## Addressing and Ports

Soil Agent binds to a specific set of addresses and ports. The HTTP API is served via the http address and port. This `address:port` must be accessible locally. If you bind to 127.0.0.1:7654, the API is only available from that host. If you bind to a private internal IP, the API will be available from within that network. If you bind to a public IP, the API will be available from the public Internet (not recommended).

The default port for the Soil Agent HTTP API is `7654`. This can be overridden via Soil Agent `--address` [flag](/soil/agent/configuration). Here is an example curl request to query a Soil Agent with the default configuration:

```
$ curl http://127.0.0.1:7654/v1/agent/ping
```

The conventions used in the API documentation do not list a port and use the standard address `127.0.0.1`. Be sure to replace this with your Soil Agent URL when using the examples.