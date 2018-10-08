# httpstats #

**Standard datadog/statsd integration for HTTP services.**

This project contains middleware for HTTP services and clients that uses
[xstats](https://github.com/rs/xstats) emit detailed runtime metrics for
service operations.

## Usage ##

### HTTP Service ###

The middleware exported is a `func(http.Handler) http.Handler` and should
work with virtually any router/mux implementation that supports middleware.

```go
var middleware = httpstats.NewMiddleware(
  httpstats.MiddlewareOptionUDPSender("statsd:8125", 1<<15, 10*time.Second, "myservice."),
)
```

The middleware uses [xstats](https://github.com/rs/xstats) as the base. A stat
client can be extracted using `xstats.FromContext` or `xstats.FromRequest` as
needed to emit custom stats. The standard set of stats emitted automatically
by the middleware are documented below.

Each service installing the middleware should emit stats to a statsd or datadog
agent. The agent location and settings are configurable through the
`httpstats.MiddlewareOptionUDPSender` or
`MiddlewareOptionUDPGlobalRollupSender` option. The global rollup option helps
prevent several forms of skew that can arise from statsd and datadog
aggregation of data and is described in greater detail below.

### HTTP Client ###

In addition to an HTTP middleware, there is also an `http.RoundTripper` wrapper
included that will properly manage spans for outgoing HTTP requests. To apply:

```golang
var client = &http.Client{
  Transport: httpstats.NewTransport(
    httpstats.TransportOptionTag("dependency", "some-other-service"),
  )(http.DefaultTransport),
}
```

Like the service middleware, the client decorator provides a set of default
stats that are detailed below. No other configuration of the client decorator
is needed as it will assume any options set in the middleware by nature of
using the same stat client from the incoming request context.

## Standard Metrics ##

### HTTP Service ###

-   service_bytes_received

    A histogram of the number of bytes received in each incoming request. The
    name for this can be overridden with
    `httpstats.MiddlewareOptionBytesInName`.

-   service_bytes_returned

    A histogram of the number of bytes sent in response to each incoming
    request. The name for this can be overridden with
    `httpstats.MiddlewareOptionBytesOutName`.

-   service_bytes_total

    A histogram of the number of bytes sent or received as part of each incoming
    request. The name for this can be overridden with
    `httpstats.MiddlewareOptionBytesTotalName`.

-   service_time

    A timer of the amount of time spend processing a request. The name
    for this can be overridden with
    `httpstats.MiddlewareOptionRequestTimeName`.

#### Tags ####

This package uses the datadog extensions to the statsd line protocol to inject
metadata for metrics. All HTTP service metrics will be tagged with:

-   method

    The HTTP method used in the incoming request.

-   status_code

    The status code returned by the service.

-   status

    A string representation of the exit status of the request. This will be
    `ok` for `2xx` range responses, `error` for other responses, `timeout` for
    cases where the request context deadline was exceeded, and `cancelled` for
    cases where the request context is explicitly cancelled.

Additional tags may be injected either statically or on a per-request basis
using the `httpstats.MiddlewareOptionTag` and
`httpstats.MiddlewareOptionRequestTag` options respectively.

### HTTP Client ###

-   client_request_bytes_received

    A histogram of the number of bytes read from the response body of an
    outgoing request. This name can be overridden using
    `httpstats.TransportOptionBytesInName`.

-   client_request_bytes_sent

    A histogram of the number of bytes sent in the body of an outgoing request.
    This name can be overridden using `httpstats.TransportOptionBytesOutName`.

-   client_request_bytes_total

    A histogram of the total bytes sent and read as part of an outgoing request.
    This name can be overridden using
    `httpstats.TransportOptionBytesTotalName`.

-   client_request_time

    A timer of how long it took for the round trip to complete. This name can be
    overridden using `stidestats.TransportOptionRequestTimeName`.

    This metric will be tagged with the following:

    -   method

        The HTTP method used in the outgoing request.

    -   status_code

        The HTTP status code received in the response.

    -   status

      A string representation of the exit status of the request. This will be
      `ok` for `2xx` range responses, `error` for other responses, `timeout` for
      cases where the request context deadline was exceeded, and `cancelled` for
      cases where the request context is explicitly cancelled.

-   client_got_connection

    A timer of how long it took for the HTTP client to acquire a TCP connection.
    This name may be overridden using
    `httpstats.TransportOptionGotConnectionName`.

    This metric will be tagged with the following:

    -   reused

        `true` or `false` to indicate whether or not the connection was reused
        or created new respectively.

    -   idle

        `true` or `false` to indicate whether or not the connection was pulled
        from the idle pool.

-   client_connection_idle

    A timer of how long a cached connection spend in the idle pool before it
    was used again. This name may be overridden using
    `httpstats.TransportOptionConnectionIdleName`.

-   client_dns

    A timer of how long it took to resolve the DNS name associated with the
    outgoing request. This name may be overridden using
    `httpstats.TransportOptionDNSName`.

    This metric will be tagged with the following:

    -   coalesced

        `true` or `false` to indicate whether or not the resolution was shared
        by multiple, concurrent lookups for the same address.

    -   error

        `true` or `false` to indicate whether or not there was an error in
        resolving DNS.

-   client_tls

    A timer of how long it took to complete the TLS handshake after acquiring
    a TCP connection to the remote host. This name may be overridden using
    `httpstats.TransportOptionTLSName`.

    This metric will be tagged with the following:

    -   error

        `true` or `false` to indicate whether or not there was an error in
        performing the TLS handshake.

-   client_wrote_headers

    A timer of how long it took between getting a connection and writing the
    request headers to the stream. This measure should closely represent the
    total amount of time it took for a request to leave the service. This
    name may be overridden using `httpstats.TransportOptionWroteHeadersName`.

-   client_first_response_byte

    A timer of how long it took between writing the headers to the stream and
    getting the first byte of a response. This measure should closely represent
    the amount of time it took the remote service to process and respond with
    the latency of the network included in the measure. This name may be
    overridden using `httpstats.TransportOptionFirstResponseByteName`.

-   client_put_idle

    A counter emitted each time a connection is placed into the idle pool. This
    name may be overridden using `httpstats.TransportOptionPutIdleName`.

    This metric will be tagged with the following:

    -   error

        `true` or `false` to indicate whether or not there was an error in
        performing the TLS handshake.


#### Tags ####

Unlike the service middleware, the application of tags is not universal across
all stats. Instead, each client stat comes with tags that are meaningful to that
specific operation. Each metric above should list the applicable tags rather
than enumerating another table here.

## Contributing ##

### License ###

This project is licensed under Apache 2.0. See LICENSE.txt for details.

### Contributing Agreement ###

Atlassian requires signing a contributor's agreement before we can accept a
patch. If you are an individual you can fill out the
[individual CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=3f94fbdc-2fbe-46ac-b14c-5d152700ae5d).
If you are contributing on behalf of your company then please fill out the
[corporate CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=e1c17c66-ca4d-4aab-a953-2c231af4a20b).
