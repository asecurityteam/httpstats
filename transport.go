package httpstats

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/xstats"
)

type recordingClientResponseBodyReadCloser struct {
	io.ReadCloser
	bytesRead        *int32
	requestBytesRead int
	statName         string
	totalStatName    string
	tags             []string
	ctx              context.Context
}

func (r *recordingClientResponseBodyReadCloser) Read(p []byte) (int, error) {
	var n, e = r.ReadCloser.Read(p)
	atomic.AddInt32(r.bytesRead, int32(n))
	return n, e
}

func (r *recordingClientResponseBodyReadCloser) Close() error {
	var bytesRead = float64(atomic.LoadInt32(r.bytesRead))
	xstats.FromContext(r.ctx).Histogram(r.statName, bytesRead, r.tags...)
	xstats.FromContext(r.ctx).Histogram(r.totalStatName, bytesRead+float64(r.requestBytesRead), r.tags...)
	return r.ReadCloser.Close()
}

type traceStater struct {
	stat               xstats.XStater
	getConnTime        time.Time
	dnsStartTime       time.Time
	tlsStartTime       time.Time
	gotConnTime        time.Time
	wroteHeaderTime    time.Time
	tags               []string
	lock               *sync.Mutex
	gotConnectionName  string
	connectionIdleName string
	dnsName            string
	tlsName            string
	wroteHeadersName   string
	firstByteName      string
	putIdleName        string
}

func (t *traceStater) getConn(hostPort string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.getConnTime = time.Now()
}

func (t *traceStater) gotConn(info httptrace.GotConnInfo) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.gotConnTime = time.Now()
	var d = time.Since(t.getConnTime)
	var tags = append(t.tags[:], fmt.Sprintf("reused:%v", info.Reused), fmt.Sprintf("idle:%v", info.WasIdle))
	t.stat.Timing(t.gotConnectionName, d, tags...)
	if info.WasIdle {
		t.stat.Timing(t.connectionIdleName, info.IdleTime, t.tags...)
	}
}

func (t *traceStater) dnsStart(httptrace.DNSStartInfo) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.dnsStartTime = time.Now()
}

func (t *traceStater) dnsDone(info httptrace.DNSDoneInfo) {
	t.lock.Lock()
	defer t.lock.Unlock()
	var d = time.Since(t.dnsStartTime)
	var tags = append(t.tags[:], fmt.Sprintf("coalesced:%v", info.Coalesced), fmt.Sprintf("error:%v", info.Err != nil))
	t.stat.Timing(t.dnsName, d, tags...)
}

func (t *traceStater) tlsHandshakeStart() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.tlsStartTime = time.Now()
}

func (t *traceStater) tlsHandshakeDone(info tls.ConnectionState, e error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	var d = time.Since(t.tlsStartTime)
	var tags = append(t.tags[:], fmt.Sprintf("error:%v", e != nil))
	t.stat.Timing(t.tlsName, d, tags...)
}

func (t *traceStater) wroteHeaders() {
	t.lock.Lock()
	defer t.lock.Unlock()
	var d = time.Since(t.gotConnTime)
	t.wroteHeaderTime = time.Now()
	t.stat.Timing(t.wroteHeadersName, d, t.tags...)
}

func (t *traceStater) firstByte() {
	t.lock.Lock()
	defer t.lock.Unlock()
	var d = time.Since(t.wroteHeaderTime)
	t.stat.Timing(t.firstByteName, d, t.tags...)
}

func (t *traceStater) putIdleConn(e error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	var tags = append(t.tags[:], fmt.Sprintf("error:%v", e != nil))
	t.stat.Count(t.putIdleName, 1, tags...)
}

func newClientTrace(stat xstats.XStater, tags []string, gotConnectionName string, connectionIdleName string, dnsName string, tlsName string, wroteHeadersName string, firstByteName string, putIdleName string) *httptrace.ClientTrace {
	var tstat = &traceStater{
		stat:               stat,
		tags:               tags,
		lock:               &sync.Mutex{},
		gotConnectionName:  gotConnectionName,
		connectionIdleName: connectionIdleName,
		dnsName:            dnsName,
		tlsName:            tlsName,
		wroteHeadersName:   wroteHeadersName,
		firstByteName:      firstByteName,
		putIdleName:        putIdleName,
	}
	return &httptrace.ClientTrace{
		GetConn:              tstat.getConn,
		GotConn:              tstat.gotConn,
		DNSStart:             tstat.dnsStart,
		DNSDone:              tstat.dnsDone,
		TLSHandshakeStart:    tstat.tlsHandshakeStart,
		TLSHandshakeDone:     tstat.tlsHandshakeDone,
		WroteHeaders:         tstat.wroteHeaders,
		GotFirstResponseByte: tstat.firstByte,
		PutIdleConn:          tstat.putIdleConn,
	}
}

// Transport is an http.RoundTripper wrapper that instruments HTTP clients with
// the standard SecDev metrics.
type Transport struct {
	tags           []string
	next           http.RoundTripper
	requestTime    string
	bytesIn        string
	bytesOut       string
	bytesTotal     string
	gotConnection  string
	connectionIdle string
	dns            string
	tls            string
	wroteHeader    string
	firstByte      string
	putIdle        string
	requestTaggers []func(*http.Request) (string, string)
}

// RoundTrip instruments the HTTP request/response cycle with metrics.
func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	var tags = make([]string, 0, len(t.requestTaggers))
	for _, tagger := range t.requestTaggers {
		var k, v = tagger(r)
		tags = append(tags, fmt.Sprintf("%s:%s", k, v))
	}
	tags = append(tags, t.tags...)
	var bodyWrapper = &recordingReader{r.Body, new(int32)}
	if r.Body != nil {
		r.Body = bodyWrapper
	}
	r = r.WithContext(
		httptrace.WithClientTrace(
			r.Context(),
			newClientTrace(
				xstats.FromRequest(r),
				tags,
				t.gotConnection,
				t.connectionIdle,
				t.dns,
				t.tls,
				t.wroteHeader,
				t.firstByte,
				t.putIdle,
			),
		),
	)
	var start = time.Now()
	var resp, e = t.next.RoundTrip(r)
	var duration = time.Since(start)
	var statusCode string
	var status string
	var bytesRead = 0
	if e == nil {
		statusCode = fmt.Sprintf("%d", resp.StatusCode)
		status = responseStatus(r.Context(), resp.StatusCode)
		if r.Body != nil {
			bytesRead = bodyWrapper.BytesRead()
		}
		resp.Body = &recordingClientResponseBodyReadCloser{
			ReadCloser:       resp.Body,
			bytesRead:        new(int32),
			requestBytesRead: bytesRead,
			statName:         t.bytesOut,
			totalStatName:    t.bytesTotal,
			tags:             tags,
			ctx:              r.Context(),
		}
	} else {
		var errorStatusCode = errorToStatusCode(e)
		statusCode = fmt.Sprintf("%d", errorStatusCode)
		status = responseStatus(r.Context(), errorStatusCode)
	}
	var timerTags = append(tags, fmt.Sprintf("method:%s", r.Method), fmt.Sprintf("status_code:%s", statusCode), fmt.Sprintf("status:%s", status))
	xstats.FromRequest(r).Timing(t.requestTime, duration, timerTags...)
	xstats.FromRequest(r).Histogram(t.bytesIn, float64(bytesRead), tags...)
	return resp, e
}

// TransportOption is used to configure the HTTP transport middleware.
type TransportOption func(*Transport) *Transport

// TransportOptionTag adds a static key/value annotation to all metrics emitted
// by the middleware.
func TransportOptionTag(tagName string, tagValue string) TransportOption {
	return func(m *Transport) *Transport {
		m.tags = append(m.tags, fmt.Sprintf("%s:%s", tagName, tagValue))
		return m
	}
}

// TransportOptionBytesInName sets the name of the metric used to track the
// number of bytes read as part of the HTTP response from a request. The
// default value is client_request_bytes_received.
func TransportOptionBytesInName(name string) TransportOption {
	return func(m *Transport) *Transport {
		m.bytesIn = name
		return m
	}
}

// TransportOptionBytesOutName sets the name of the metric used to track the
// number of bytes written as part of the HTTP request body. The default value
// is client_request_bytes_sent.
func TransportOptionBytesOutName(name string) TransportOption {
	return func(m *Transport) *Transport {
		m.bytesOut = name
		return m
	}
}

// TransportOptionBytesTotalName sets the name of the metric used to track the
// number of bytes written and read as part of the HTTP request. The default
// value is client_request_bytes_total.
func TransportOptionBytesTotalName(name string) TransportOption {
	return func(m *Transport) *Transport {
		m.bytesTotal = name
		return m
	}
}

// TransportOptionRequestTimeName sets the name of the metric used to track the
// duration of the HTTP request. The default value is client_request_time.
func TransportOptionRequestTimeName(name string) TransportOption {
	return func(m *Transport) *Transport {
		m.requestTime = name
		return m
	}
}

// TransportOptionGotConnectionName sets the name of the metric used to track
// the duration of the creating/fetching a TCP connection. The default value is
// client_got_connection.
func TransportOptionGotConnectionName(name string) TransportOption {
	return func(m *Transport) *Transport {
		m.gotConnection = name
		return m
	}
}

// TransportOptionConnectionIdleName sets the name of the metric used to track
// the duration a TCP connection spent in the idle pool. The default value is
// client_connection_idle.
func TransportOptionConnectionIdleName(name string) TransportOption {
	return func(m *Transport) *Transport {
		m.connectionIdle = name
		return m
	}
}

// TransportOptionDNSName sets the name of the metric used to track the duration
// of resolving the target DNS name of hte request. The default value is
// client_dns.
func TransportOptionDNSName(name string) TransportOption {
	return func(m *Transport) *Transport {
		m.dns = name
		return m
	}
}

// TransportOptionTLSName sets the name of the metric used to track the duration
// of performing the TLS handshake with the remote server. The default value is
// client_tls.
func TransportOptionTLSName(name string) TransportOption {
	return func(m *Transport) *Transport {
		m.tls = name
		return m
	}
}

// TransportOptionWroteHeadersName sets the name of the metric used to track the
// duration between the time to get a TCP connection and when the request
// headers are written to the stream. The default value is client_wrote_headers.
func TransportOptionWroteHeadersName(name string) TransportOption {
	return func(m *Transport) *Transport {
		m.wroteHeader = name
		return m
	}
}

// TransportOptionFirstResponseByteName sets the name of the metric used to
// track the duration between the time when the request headers are written to
// the stream and the first byte of the response being read back. The default
// value is client_first_response_byte.
func TransportOptionFirstResponseByteName(name string) TransportOption {
	return func(m *Transport) *Transport {
		m.firstByte = name
		return m
	}
}

// TransportOptionPutIdleName sets the name of the metric used to count the
// number of connections that are placed in the idle pool. The default value is
// client_put_idle.
func TransportOptionPutIdleName(name string) TransportOption {
	return func(m *Transport) *Transport {
		m.putIdle = name
		return m
	}
}

// TransportOptionRequestTag is called on each round trip with the request
// instance. The key/value pair returned by this function will be used to
// annotate all metrics emitted by the transport middleware.
func TransportOptionRequestTag(tagger func(*http.Request) (string, string)) TransportOption {
	return func(m *Transport) *Transport {
		m.requestTaggers = append(m.requestTaggers, tagger)
		return m
	}
}

// NewTransport configures and returns an HTTP Transport middleware.
func NewTransport(options ...TransportOption) func(http.RoundTripper) http.RoundTripper {
	xstats.DisablePooling = true
	return func(next http.RoundTripper) http.RoundTripper {
		var m = &Transport{
			bytesIn:        "client_request_bytes_received",
			bytesOut:       "client_request_bytes_sent",
			bytesTotal:     "client_request_bytes_total",
			requestTime:    "client_request_time",
			gotConnection:  "client_got_connection",
			connectionIdle: "client_connection_idle",
			dns:            "client_dns",
			tls:            "client_tls",
			wroteHeader:    "client_wrote_headers",
			firstByte:      "client_first_response_byte",
			putIdle:        "client_put_idle",
			next:           next,
		}
		for _, option := range options {
			m = option(m)
		}
		return m
	}
}
