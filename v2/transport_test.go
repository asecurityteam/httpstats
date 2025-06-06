package httpstats

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

	"github.com/rs/xstats"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestTraceStats(t *testing.T) {
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

	stat.EXPECT().Timing("dns", gomock.Any(), "test:test", "coalesced:false", "error:false")
	trace.DNSStart(httptrace.DNSStartInfo{})
	trace.DNSDone(httptrace.DNSDoneInfo{})

	stat.EXPECT().Timing("dns", gomock.Any(), "test:test", "coalesced:true", "error:true")
	trace.DNSStart(httptrace.DNSStartInfo{})
	trace.DNSDone(httptrace.DNSDoneInfo{Coalesced: true, Err: errors.New("")})

	stat.EXPECT().Timing("tls", gomock.Any(), "test:test", "error:false")
	trace.TLSHandshakeStart()
	trace.TLSHandshakeDone(tls.ConnectionState{}, nil)

	stat.EXPECT().Timing("tls", gomock.Any(), "test:test", "error:true")
	trace.TLSHandshakeStart()
	trace.TLSHandshakeDone(tls.ConnectionState{}, errors.New(""))

	stat.EXPECT().Timing("wroteheader", gomock.Any(), "test:test")
	trace.WroteHeaders()

	stat.EXPECT().Timing("firstbyte", gomock.Any(), "test:test")
	trace.GotFirstResponseByte()

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

func TestTransportOptions(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result = NewTransport(
		TransportOptionTag(testName, testName),
		TransportOptionRequestTag(func(*http.Request) (string, string) { return test2Name, test2Name }),
		TransportOptionRequestTimeName("requesttime"),
		TransportOptionBytesInName("bytesin"),
		TransportOptionBytesOutName("bytesout"),
		TransportOptionBytesTotalName("bytestotal"),
		TransportOptionDNSName("dns"),
		TransportOptionGotConnectionName("gotcon"),
		TransportOptionTLSName("tls"),
		TransportOptionWroteHeadersName("wroteheaders"),
		TransportOptionFirstResponseByteName("firstbyte"),
		TransportOptionPutIdleName("putidle"),
	)
	var r = result(&fixtureTransport{
		response: &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
		},
		err: nil,
	}).(*Transport)

	var req = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(xstats.NewContext(context.Background(), sender))
	sender.EXPECT().Timing("requesttime", gomock.Any(), "test2:test2", "test:test", "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram("bytesin", gomock.Any(), "test2:test2", "test:test")
	sender.EXPECT().Histogram("bytesout", gomock.Any(), "test2:test2", "test:test")
	sender.EXPECT().Histogram("bytestotal", gomock.Any(), "test2:test2", "test:test")
	sender.EXPECT().Timing("dns", gomock.Any(), "test2:test2", "test:test", "coalesced:false", "error:false")
	sender.EXPECT().Timing("gotcon", gomock.Any(), "test2:test2", "test:test", "reused:false", "idle:false")
	sender.EXPECT().Timing("tls", gomock.Any(), "test2:test2", "test:test", "error:false")
	sender.EXPECT().Timing("wroteheaders", gomock.Any(), "test2:test2", "test:test")
	sender.EXPECT().Timing("firstbyte", gomock.Any(), "test2:test2", "test:test")
	sender.EXPECT().Count("putidle", gomock.Any(), "test2:test2", "test:test", "error:false")
	var resp, _ = r.RoundTrip(req)
	resp.Body.Close()
}

func TestTransportNoPanicWhenBodyNil(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var result = NewTransport(
		TransportOptionTag(testName, testName),
		TransportOptionRequestTag(func(*http.Request) (string, string) { return test2Name, test2Name }),
		TransportOptionRequestTimeName("requesttime"),
		TransportOptionBytesInName("bytesin"),
		TransportOptionBytesOutName("bytesout"),
		TransportOptionBytesTotalName("bytestotal"),
		TransportOptionDNSName("dns"),
		TransportOptionGotConnectionName("gotcon"),
		TransportOptionTLSName("tls"),
		TransportOptionWroteHeadersName("wroteheaders"),
		TransportOptionFirstResponseByteName("firstbyte"),
		TransportOptionPutIdleName("putidle"),
	)
	var r = result(http.DefaultTransport)
	var req, _ = http.NewRequest(http.MethodGet, "https://localhost/asdfasdfadsf", nil)
	_, _ = r.RoundTrip(req)
}

type instanceStoreTransport struct {
	instance xstats.XStater
}

func (t *instanceStoreTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.instance = xstats.FromRequest(r)
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(``)),
	}, nil
}

func TestNewStatTransport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	var sender = NewMockXStater(ctrl)
	wrapped := &instanceStoreTransport{}
	stats := xstats.New(sender)
	transport := NewStatTransport(stats)(wrapped)
	_, _ = transport.RoundTrip(httptest.NewRequest(http.MethodGet, "/", nil))
	assert.NotNil(t, wrapped.instance)
}
