package stridestats

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rs/xstats"
)

func TestTraceStatGetConnGotConn(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var stat = NewMockXStater(ctrl)
	var trace = newClientTrace(stat, []string{"test:test"}, "gotconn", "connectionidle", "dns", "tls", "wroteheader", "firstbyte", "putidle")

	stat.EXPECT().Timing("gotconn", gomock.Any(), "test:test", "reused:false", "idle:false")
	trace.GetConn("")
	trace.GotConn(httptrace.GotConnInfo{})

	stat.EXPECT().Timing("gotconn", gomock.Any(), "test:test", "reused:true", "idle:true")
	stat.EXPECT().Timing("connectionidle", gomock.Any(), "test:test")
	trace.GetConn("")
	trace.GotConn(httptrace.GotConnInfo{Reused: true, WasIdle: true})
}

func TestTraceStatDNS(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var stat = NewMockXStater(ctrl)
	var trace = newClientTrace(stat, []string{"test:test"}, "gotconn", "connectionidle", "dns", "tls", "wroteheader", "firstbyte", "putidle")

	stat.EXPECT().Timing("dns", gomock.Any(), "test:test", "coalesced:false", "error:false")
	trace.DNSStart(httptrace.DNSStartInfo{})
	trace.DNSDone(httptrace.DNSDoneInfo{})

	stat.EXPECT().Timing("dns", gomock.Any(), "test:test", "coalesced:true", "error:true")
	trace.DNSStart(httptrace.DNSStartInfo{})
	trace.DNSDone(httptrace.DNSDoneInfo{Coalesced: true, Err: errors.New("")})
}

func TestTraceStatTLS(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var stat = NewMockXStater(ctrl)
	var trace = newClientTrace(stat, []string{"test:test"}, "gotconn", "connectionidle", "dns", "tls", "wroteheader", "firstbyte", "putidle")

	stat.EXPECT().Timing("tls", gomock.Any(), "test:test", "error:false")
	trace.TLSHandshakeStart()
	trace.TLSHandshakeDone(tls.ConnectionState{}, nil)

	stat.EXPECT().Timing("tls", gomock.Any(), "test:test", "error:true")
	trace.TLSHandshakeStart()
	trace.TLSHandshakeDone(tls.ConnectionState{}, errors.New(""))
}

func TestTraceStatWroteHeaders(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var stat = NewMockXStater(ctrl)
	var trace = newClientTrace(stat, []string{"test:test"}, "gotconn", "connectionidle", "dns", "tls", "wroteheader", "firstbyte", "putidle")

	stat.EXPECT().Timing("wroteheader", gomock.Any(), "test:test")
	trace.WroteHeaders()
}

func TestTraceStatFirstByte(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var stat = NewMockXStater(ctrl)
	var trace = newClientTrace(stat, []string{"test:test"}, "gotconn", "connectionidle", "dns", "tls", "wroteheader", "firstbyte", "putidle")

	stat.EXPECT().Timing("firstbyte", gomock.Any(), "test:test")
	trace.GotFirstResponseByte()
}

func TestTraceStatPutIdle(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var stat = NewMockXStater(ctrl)
	var trace = newClientTrace(stat, []string{"test:test"}, "gotconn", "connectionidle", "dns", "tls", "wroteheader", "firstbyte", "putidle")

	stat.EXPECT().Count("putidle", gomock.Any(), "test:test", "error:false")
	trace.PutIdleConn(nil)
	stat.EXPECT().Count("putidle", gomock.Any(), "test:test", "error:true")
	trace.PutIdleConn(errors.New(""))
}

type fixtureTransport struct {
	response *http.Response
	err      error
}

func (r *fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var trace = httptrace.ContextClientTrace(req.Context())
	trace.DNSStart(httptrace.DNSStartInfo{})
	trace.DNSDone(httptrace.DNSDoneInfo{})
	trace.GetConn("")
	trace.GotConn(httptrace.GotConnInfo{})
	trace.TLSHandshakeStart()
	trace.TLSHandshakeDone(tls.ConnectionState{}, nil)
	trace.WroteHeaders()
	trace.GotFirstResponseByte()
	trace.PutIdleConn(nil)

	return r.response, r.err
}

func TestTransportOptionTag(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(TransportOptionTag("test", "test"))
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing(r.requestTime, gomock.Any(), "test:test", "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(r.bytesIn, gomock.Any(), "test:test")
	sender.EXPECT().Histogram(r.bytesOut, gomock.Any(), "test:test")
	sender.EXPECT().Histogram(r.bytesTotal, gomock.Any(), "test:test")
	sender.EXPECT().Timing(r.dns, gomock.Any(), "test:test", "coalesced:false", "error:false")
	sender.EXPECT().Timing(r.gotConnection, gomock.Any(), "test:test", "reused:false", "idle:false")
	sender.EXPECT().Timing(r.tls, gomock.Any(), "test:test", "error:false")
	sender.EXPECT().Timing(r.wroteHeader, gomock.Any(), "test:test")
	sender.EXPECT().Timing(r.firstByte, gomock.Any(), "test:test")
	sender.EXPECT().Count(r.putIdle, gomock.Any(), "test:test", "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}

func TestTransportOptionBytesInName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(TransportOptionBytesInName("test"))
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing(r.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram("test", gomock.Any())
	sender.EXPECT().Histogram(r.bytesOut, gomock.Any())
	sender.EXPECT().Histogram(r.bytesTotal, gomock.Any())
	sender.EXPECT().Timing(r.dns, gomock.Any(), "coalesced:false", "error:false")
	sender.EXPECT().Timing(r.gotConnection, gomock.Any(), "reused:false", "idle:false")
	sender.EXPECT().Timing(r.tls, gomock.Any(), "error:false")
	sender.EXPECT().Timing(r.wroteHeader, gomock.Any())
	sender.EXPECT().Timing(r.firstByte, gomock.Any())
	sender.EXPECT().Count(r.putIdle, gomock.Any(), "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}

func TestTransportOptionBytesOutName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(TransportOptionBytesOutName("test"))
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing(r.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(r.bytesIn, gomock.Any())
	sender.EXPECT().Histogram("test", gomock.Any())
	sender.EXPECT().Histogram(r.bytesTotal, gomock.Any())
	sender.EXPECT().Timing(r.dns, gomock.Any(), "coalesced:false", "error:false")
	sender.EXPECT().Timing(r.gotConnection, gomock.Any(), "reused:false", "idle:false")
	sender.EXPECT().Timing(r.tls, gomock.Any(), "error:false")
	sender.EXPECT().Timing(r.wroteHeader, gomock.Any())
	sender.EXPECT().Timing(r.firstByte, gomock.Any())
	sender.EXPECT().Count(r.putIdle, gomock.Any(), "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}

func TestTransportOptionBytesTotalName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(TransportOptionBytesTotalName("test"))
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing(r.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(r.bytesIn, gomock.Any())
	sender.EXPECT().Histogram(r.bytesOut, gomock.Any())
	sender.EXPECT().Histogram("test", gomock.Any())
	sender.EXPECT().Timing(r.dns, gomock.Any(), "coalesced:false", "error:false")
	sender.EXPECT().Timing(r.gotConnection, gomock.Any(), "reused:false", "idle:false")
	sender.EXPECT().Timing(r.tls, gomock.Any(), "error:false")
	sender.EXPECT().Timing(r.wroteHeader, gomock.Any())
	sender.EXPECT().Timing(r.firstByte, gomock.Any())
	sender.EXPECT().Count(r.putIdle, gomock.Any(), "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}

func TestTransportOptionRequestTimeName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(TransportOptionRequestTimeName("test"))
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing("test", gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(r.bytesIn, gomock.Any())
	sender.EXPECT().Histogram(r.bytesOut, gomock.Any())
	sender.EXPECT().Histogram(r.bytesTotal, gomock.Any())
	sender.EXPECT().Timing(r.dns, gomock.Any(), "coalesced:false", "error:false")
	sender.EXPECT().Timing(r.gotConnection, gomock.Any(), "reused:false", "idle:false")
	sender.EXPECT().Timing(r.tls, gomock.Any(), "error:false")
	sender.EXPECT().Timing(r.wroteHeader, gomock.Any())
	sender.EXPECT().Timing(r.firstByte, gomock.Any())
	sender.EXPECT().Count(r.putIdle, gomock.Any(), "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}

func TestTransportOptionGotConnectionName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(TransportOptionGotConnectionName("test"))
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing(r.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(r.bytesIn, gomock.Any())
	sender.EXPECT().Histogram(r.bytesOut, gomock.Any())
	sender.EXPECT().Histogram(r.bytesTotal, gomock.Any())
	sender.EXPECT().Timing(r.dns, gomock.Any(), "coalesced:false", "error:false")
	sender.EXPECT().Timing("test", gomock.Any(), "reused:false", "idle:false")
	sender.EXPECT().Timing(r.tls, gomock.Any(), "error:false")
	sender.EXPECT().Timing(r.wroteHeader, gomock.Any())
	sender.EXPECT().Timing(r.firstByte, gomock.Any())
	sender.EXPECT().Count(r.putIdle, gomock.Any(), "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}

func TestTransportOptionDNSName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(TransportOptionDNSName("test"))
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing(r.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(r.bytesIn, gomock.Any())
	sender.EXPECT().Histogram(r.bytesOut, gomock.Any())
	sender.EXPECT().Histogram(r.bytesTotal, gomock.Any())
	sender.EXPECT().Timing("test", gomock.Any(), "coalesced:false", "error:false")
	sender.EXPECT().Timing(r.gotConnection, gomock.Any(), "reused:false", "idle:false")
	sender.EXPECT().Timing(r.tls, gomock.Any(), "error:false")
	sender.EXPECT().Timing(r.wroteHeader, gomock.Any())
	sender.EXPECT().Timing(r.firstByte, gomock.Any())
	sender.EXPECT().Count(r.putIdle, gomock.Any(), "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}

func TestTransportOptionTLSName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(TransportOptionTLSName("test"))
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing(r.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(r.bytesIn, gomock.Any())
	sender.EXPECT().Histogram(r.bytesOut, gomock.Any())
	sender.EXPECT().Histogram(r.bytesTotal, gomock.Any())
	sender.EXPECT().Timing(r.dns, gomock.Any(), "coalesced:false", "error:false")
	sender.EXPECT().Timing(r.gotConnection, gomock.Any(), "reused:false", "idle:false")
	sender.EXPECT().Timing("test", gomock.Any(), "error:false")
	sender.EXPECT().Timing(r.wroteHeader, gomock.Any())
	sender.EXPECT().Timing(r.firstByte, gomock.Any())
	sender.EXPECT().Count(r.putIdle, gomock.Any(), "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}

func TestTransportOptionWroteHeadersName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(TransportOptionWroteHeadersName("test"))
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing(r.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(r.bytesIn, gomock.Any())
	sender.EXPECT().Histogram(r.bytesOut, gomock.Any())
	sender.EXPECT().Histogram(r.bytesTotal, gomock.Any())
	sender.EXPECT().Timing(r.dns, gomock.Any(), "coalesced:false", "error:false")
	sender.EXPECT().Timing(r.gotConnection, gomock.Any(), "reused:false", "idle:false")
	sender.EXPECT().Timing(r.tls, gomock.Any(), "error:false")
	sender.EXPECT().Timing("test", gomock.Any())
	sender.EXPECT().Timing(r.firstByte, gomock.Any())
	sender.EXPECT().Count(r.putIdle, gomock.Any(), "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}

func TestTransportOptionFirstResponseByteName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(TransportOptionFirstResponseByteName("test"))
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing(r.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(r.bytesIn, gomock.Any())
	sender.EXPECT().Histogram(r.bytesOut, gomock.Any())
	sender.EXPECT().Histogram(r.bytesTotal, gomock.Any())
	sender.EXPECT().Timing(r.dns, gomock.Any(), "coalesced:false", "error:false")
	sender.EXPECT().Timing(r.gotConnection, gomock.Any(), "reused:false", "idle:false")
	sender.EXPECT().Timing(r.tls, gomock.Any(), "error:false")
	sender.EXPECT().Timing(r.wroteHeader, gomock.Any())
	sender.EXPECT().Timing("test", gomock.Any())
	sender.EXPECT().Count(r.putIdle, gomock.Any(), "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}

func TestTransportOptionPutIdleName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(TransportOptionPutIdleName("test"))
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing(r.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(r.bytesIn, gomock.Any())
	sender.EXPECT().Histogram(r.bytesOut, gomock.Any())
	sender.EXPECT().Histogram(r.bytesTotal, gomock.Any())
	sender.EXPECT().Timing(r.dns, gomock.Any(), "coalesced:false", "error:false")
	sender.EXPECT().Timing(r.gotConnection, gomock.Any(), "reused:false", "idle:false")
	sender.EXPECT().Timing(r.tls, gomock.Any(), "error:false")
	sender.EXPECT().Timing(r.wroteHeader, gomock.Any())
	sender.EXPECT().Timing(r.firstByte, gomock.Any())
	sender.EXPECT().Count("test", gomock.Any(), "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}

func TestTransportOptionRequestTag(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(TransportOptionRequestTag(func(*http.Request) (string, string) { return "test", "test" }))
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing(r.requestTime, gomock.Any(), "test:test", "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(r.bytesIn, gomock.Any(), "test:test")
	sender.EXPECT().Histogram(r.bytesOut, gomock.Any(), "test:test")
	sender.EXPECT().Histogram(r.bytesTotal, gomock.Any(), "test:test")
	sender.EXPECT().Timing(r.dns, gomock.Any(), "test:test", "coalesced:false", "error:false")
	sender.EXPECT().Timing(r.gotConnection, gomock.Any(), "test:test", "reused:false", "idle:false")
	sender.EXPECT().Timing(r.tls, gomock.Any(), "test:test", "error:false")
	sender.EXPECT().Timing(r.wroteHeader, gomock.Any(), "test:test")
	sender.EXPECT().Timing(r.firstByte, gomock.Any(), "test:test")
	sender.EXPECT().Count(r.putIdle, gomock.Any(), "test:test", "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}
