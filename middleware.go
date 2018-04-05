package stridestats

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/rs/xstats"
	"github.com/rs/xstats/dogstatsd"
)

// Middleware is an http.Handler wrapper that instruments HTTP servers with the
// standard Stride metrics.
type Middleware struct {
	senders          []xstats.Sender
	tags             []string
	tagMap           map[string]string
	next             http.Handler
	requestTime      string
	bytesIn          string
	bytesOut         string
	bytesTotal       string
	requestTaggers   []func(*http.Request) (string, string)
	finalSender      xstats.Sender
	xstatsMiddleware func(http.Handler) http.Handler
}

type recordingReader struct {
	io.ReadCloser
	bytesRead *int32
}

func (r *recordingReader) BytesRead() int {
	return int(atomic.LoadInt32(r.bytesRead))
}

func (r *recordingReader) Read(p []byte) (int, error) {
	var n, e = r.ReadCloser.Read(p)
	atomic.AddInt32(r.bytesRead, int32(n))
	return n, e
}

func (m *Middleware) serveHTTP(w http.ResponseWriter, r *http.Request) {
	var tags []string
	for _, tagger := range m.requestTaggers {
		var k, v = tagger(r)
		tags = append(tags, fmt.Sprintf("%s:%s", k, v))
	}
	xstats.FromRequest(r).AddTags(tags...)
	var wrapper = wrapWriter(w, r.ProtoMajor)
	var bodyWrapper = &recordingReader{r.Body, new(int32)}
	r.Body = bodyWrapper
	var start = time.Now()
	m.next.ServeHTTP(wrapper, r)
	var duration = time.Since(start)
	tags = []string{
		fmt.Sprintf("method:%s", r.Method),
		fmt.Sprintf("status_code:%d", wrapper.Status()),
		fmt.Sprintf("status:%s", responseStatus(r.Context(), wrapper.Status())),
	}
	xstats.FromRequest(r).Timing(m.requestTime, duration, tags...)
	xstats.FromRequest(r).Histogram(m.bytesIn, float64(bodyWrapper.BytesRead()), tags...)
	xstats.FromRequest(r).Histogram(m.bytesOut, float64(wrapper.BytesWritten()), tags...)
	xstats.FromRequest(r).Histogram(m.bytesTotal, float64(bodyWrapper.BytesRead()+wrapper.BytesWritten()), tags...)
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.xstatsMiddleware(http.HandlerFunc(m.serveHTTP)).ServeHTTP(w, r)
}

func responseStatus(ctx context.Context, statusCode int) string {
	if ctx.Err() != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "timeout"
		}
		return "cancelled"
	}
	if statusCode >= 200 && statusCode < 300 {
		return "ok"
	}
	return "error"
}

// MiddlewareOption is used to configure the HTTP server middleware.
type MiddlewareOption func(*Middleware) (*Middleware, error)

// MiddlewareOptionTag applies a static key/value pair to all metrics.
func MiddlewareOptionTag(tagName string, tagValue string) MiddlewareOption {
	return func(m *Middleware) (*Middleware, error) {
		m.tags = append(m.tags, fmt.Sprintf("%s:%s", tagName, tagValue))
		m.tagMap[tagName] = tagValue
		return m, nil
	}
}

// MiddlewareOptionUDPSender enables datadog style statsd emissions over UDP
func MiddlewareOptionUDPSender(host string, maxPacketSize int, flushInterval time.Duration, prefix string) MiddlewareOption {
	return middlewareOptionUDPSenderDialer(host, maxPacketSize, flushInterval, prefix, net.Dial)
}

func middlewareOptionUDPSenderDialer(host string, maxPacketSize int, flushInterval time.Duration, prefix string, dialer func(network string, address string) (net.Conn, error)) MiddlewareOption {
	return func(m *Middleware) (*Middleware, error) {
		var statWriter, e = dialer("udp", host)
		if e != nil {
			return nil, e
		}
		var sender = dogstatsd.NewMaxPacket(statWriter, flushInterval, maxPacketSize)
		if len(prefix) > 0 {
			sender = xstats.NewPrefix(sender, prefix)
		}
		m.senders = append(m.senders, sender)
		return m, nil
	}
}

// MiddlewareOptionUDPGlobalRollupSender enables datadog style statsd emissions
// over UDP but specifically for timers and percentiles which might need global
// aggregation to prevent host outliers from skewing percentiles.
func MiddlewareOptionUDPGlobalRollupSender(host string, maxPacketSize int, flushInterval time.Duration, prefix string, rollupTags []string) MiddlewareOption {
	return middlewareOptionUDPGlobalRollupSenderDialer(host, maxPacketSize, flushInterval, prefix, rollupTags, net.Dial)
}

func middlewareOptionUDPGlobalRollupSenderDialer(host string, maxPacketSize int, flushInterval time.Duration, prefix string, rollupTags []string, dialer func(network string, address string) (net.Conn, error)) MiddlewareOption {
	return func(m *Middleware) (*Middleware, error) {
		var globalWriter, e = dialer("udp", host)
		if e != nil {
			return nil, e
		}
		var globalSender = dogstatsd.NewMaxPacket(globalWriter, flushInterval, maxPacketSize)
		if len(prefix) > 0 {
			globalSender = xstats.NewPrefix(globalSender, prefix)
		}
		var rollupSender = &rollupStatWrapper{
			Sender:  globalSender,
			globals: rollupTags,
		}
		m.senders = append(m.senders, rollupSender)
		return m, nil
	}
}

// MiddlewareOptionBytesInName sets the metric name used to identity the number
// of bytes read from an incoming HTTP request. The default value is
// service_bytes_received
func MiddlewareOptionBytesInName(name string) MiddlewareOption {
	return func(m *Middleware) (*Middleware, error) {
		m.bytesIn = name
		return m, nil
	}
}

// MiddlewareOptionBytesOutName sets the metric name used to identify the
// number of bytes written as the result of a request. The default value is
// service_bytes_sent
func MiddlewareOptionBytesOutName(name string) MiddlewareOption {
	return func(m *Middleware) (*Middleware, error) {
		m.bytesOut = name
		return m, nil
	}
}

// MiddlewareOptionBytesTotalName sets the metric name used to identify the
// total number of bytes read or written as the result of handling a request.
// The default value is service_bytes_total
func MiddlewareOptionBytesTotalName(name string) MiddlewareOption {
	return func(m *Middleware) (*Middleware, error) {
		m.bytesTotal = name
		return m, nil
	}
}

// MiddlewareOptionRequestTimeName sets the metric name used to indentify the
// duration of handling requests. The default value is service_time.
func MiddlewareOptionRequestTimeName(name string) MiddlewareOption {
	return func(m *Middleware) (*Middleware, error) {
		m.requestTime = name
		return m, nil
	}
}

// MiddlewareOptionRequestTag is a function that is run on every incoming
// request. The resulting key/value pair emitted is added to the stat sender
// such that all stats emitted during the lifetime of the request will have the
// annotations.
func MiddlewareOptionRequestTag(tagger func(*http.Request) (string, string)) MiddlewareOption {
	return func(m *Middleware) (*Middleware, error) {
		m.requestTaggers = append(m.requestTaggers, tagger)
		return m, nil
	}
}

// NewMiddleware configures and constructs a stat emitting HTTP middleware.
func NewMiddleware(options ...MiddlewareOption) (func(http.Handler) http.Handler, error) {
	xstats.DisablePooling = true
	var e error
	var m = &Middleware{
		bytesIn:     "service_bytes_received",
		bytesOut:    "service_bytes_returned",
		bytesTotal:  "service_bytes_total",
		requestTime: "service_time",
		tagMap:      make(map[string]string),
	}

	for _, option := range options {
		m, e = option(m)
		if e != nil {
			return nil, e
		}
	}

	if len(m.senders) < 1 {
		m.senders = append(m.senders, dogstatsd.New(ioutil.Discard, 10*time.Second))
	}

	var taggedSender = xstats.New(xstats.MultiSender(m.senders))
	taggedSender.AddTags(m.tags...)

	return func(next http.Handler) http.Handler {
		return &Middleware{
			bytesIn:          m.bytesIn,
			bytesOut:         m.bytesOut,
			bytesTotal:       m.bytesTotal,
			requestTime:      m.requestTime,
			tags:             m.tags,
			tagMap:           m.tagMap,
			next:             next,
			requestTaggers:   m.requestTaggers,
			finalSender:      taggedSender,
			senders:          m.senders,
			xstatsMiddleware: xstats.NewHandler(taggedSender, nil),
		}
	}, nil
}

// OutOfBand returns a context with all of the configuration provided to the
// middleware. This is provided with the primary intent of allowing for stats
// emissions during runtime setup (such as main.go) and background routines that
// are not attached to a request or request context.
func OutOfBand(ctx context.Context, middleware func(http.Handler) http.Handler) context.Context {
	if m, ok := middleware(nil).(*Middleware); ok {
		ctx = xstats.NewContext(ctx, xstats.New(m.finalSender))
	}
	return ctx
}