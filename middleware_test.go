package stridestats

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rs/xstats"
)

type fixtureHandler struct{}

func (fixtureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func middlewareOptionSender(s xstats.Sender) MiddlewareOption {
	return func(m *Middleware) (*Middleware, error) {
		m.senders = append(m.senders, s)
		return m, nil
	}
}

func TestMiddlewareOptionTag(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result, e = NewMiddleware(middlewareOptionSender(sender), MiddlewareOptionTag("test", "test"))
	if e != nil {
		t.Fatal(e.Error())
	}
	var m = result(fixtureHandler{}).(*Middleware)
	sender.EXPECT().Timing(m.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok", "test:test")
	sender.EXPECT().Histogram(m.bytesIn, gomock.Any(), "method:GET", "status_code:200", "status:ok", "test:test")
	sender.EXPECT().Histogram(m.bytesOut, gomock.Any(), "method:GET", "status_code:200", "status:ok", "test:test")
	sender.EXPECT().Histogram(m.bytesTotal, gomock.Any(), "method:GET", "status_code:200", "status:ok", "test:test")
	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestMiddlewareOptionBytesInName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result, e = NewMiddleware(middlewareOptionSender(sender), MiddlewareOptionBytesInName("bytesin"))
	if e != nil {
		t.Fatal(e.Error())
	}
	var m = result(fixtureHandler{}).(*Middleware)
	sender.EXPECT().Timing(m.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram("bytesin", gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesOut, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesTotal, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestMiddlewareOptionBytesOutName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result, e = NewMiddleware(middlewareOptionSender(sender), MiddlewareOptionBytesOutName("bytesout"))
	if e != nil {
		t.Fatal(e.Error())
	}
	var m = result(fixtureHandler{}).(*Middleware)
	sender.EXPECT().Timing(m.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesIn, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram("bytesout", gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesTotal, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestMiddlewareOptionBytesTotalName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result, e = NewMiddleware(middlewareOptionSender(sender), MiddlewareOptionBytesTotalName("bytestotal"))
	if e != nil {
		t.Fatal(e.Error())
	}
	var m = result(fixtureHandler{}).(*Middleware)
	sender.EXPECT().Timing(m.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesIn, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesOut, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram("bytestotal", gomock.Any(), "method:GET", "status_code:200", "status:ok")
	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestMiddlewareOptionRequestTimeName(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result, e = NewMiddleware(middlewareOptionSender(sender), MiddlewareOptionRequestTimeName("requesttime"))
	if e != nil {
		t.Fatal(e.Error())
	}
	var m = result(fixtureHandler{}).(*Middleware)
	sender.EXPECT().Timing("requesttime", gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesIn, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesOut, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesTotal, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestMiddlewareOptionRequestTag(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result, e = NewMiddleware(middlewareOptionSender(sender), MiddlewareOptionRequestTag(func(*http.Request) (string, string) { return "test", "test" }))
	if e != nil {
		t.Fatal(e.Error())
	}
	var m = result(fixtureHandler{}).(*Middleware)
	sender.EXPECT().Timing(m.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok", "test:test")
	sender.EXPECT().Histogram(m.bytesIn, gomock.Any(), "method:GET", "status_code:200", "status:ok", "test:test")
	sender.EXPECT().Histogram(m.bytesOut, gomock.Any(), "method:GET", "status_code:200", "status:ok", "test:test")
	sender.EXPECT().Histogram(m.bytesTotal, gomock.Any(), "method:GET", "status_code:200", "status:ok", "test:test")
	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

type fixtureConn struct{}

func (*fixtureConn) Read(b []byte) (n int, err error) {
	return len(b), nil
}
func (*fixtureConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}
func (*fixtureConn) Close() error {
	return nil
}
func (*fixtureConn) LocalAddr() net.Addr {
	var a, _ = net.ResolveUDPAddr("udp", "127.0.0.1")
	return a
}

func (*fixtureConn) RemoteAddr() net.Addr {
	var a, _ = net.ResolveUDPAddr("udp", "127.0.0.1")
	return a
}

func (*fixtureConn) SetDeadline(t time.Time) error {
	return nil
}
func (*fixtureConn) SetReadDeadline(t time.Time) error {
	return nil
}
func (*fixtureConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func fixtureDialFunc(network string, address string) (net.Conn, error) {
	return &fixtureConn{}, nil
}

func TestMiddlewareOptionUDPSender(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var result, e = NewMiddleware(middlewareOptionSender(sender), middlewareOptionUDPSenderDialer("localhost", 1, time.Second, "test", fixtureDialFunc))
	if e != nil {
		t.Fatal(e.Error())
	}
	var m = result(fixtureHandler{}).(*Middleware)
	sender.EXPECT().Timing(m.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesIn, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesOut, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesTotal, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestMiddlewareOptionUDPGlobalRollupSender(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var sender = NewMockXStater(ctrl)
	var rollupSender = NewMockSender(ctrl)
	var result, e = NewMiddleware(middlewareOptionSender(sender), middlewareOptionUDPGlobalRollupSenderDialer("localhost", 1, time.Second, "test", []string{"test"}, fixtureDialFunc))
	if e != nil {
		t.Fatal(e.Error())
	}
	var m = result(fixtureHandler{}).(*Middleware)
	m.senders[1].(*rollupStatWrapper).Sender = rollupSender
	sender.EXPECT().Timing(m.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesIn, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesOut, gomock.Any(), "method:GET", "status_code:200", "status:ok")
	sender.EXPECT().Histogram(m.bytesTotal, gomock.Any(), "method:GET", "status_code:200", "status:ok")

	rollupSender.EXPECT().Timing(m.requestTime, gomock.Any(), "method:GET", "status_code:200", "status:ok", "test:global")
	rollupSender.EXPECT().Histogram(m.bytesIn, gomock.Any(), "method:GET", "status_code:200", "status:ok", "test:global")
	rollupSender.EXPECT().Histogram(m.bytesOut, gomock.Any(), "method:GET", "status_code:200", "status:ok", "test:global")
	rollupSender.EXPECT().Histogram(m.bytesTotal, gomock.Any(), "method:GET", "status_code:200", "status:ok", "test:global")
	m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestResponseCodeStatus(t *testing.T) {
	var ctx = context.Background()
	var statusCode = 100
	if responseStatus(ctx, statusCode) != "error" {
		t.Fatal(responseStatus(ctx, statusCode))
	}
	statusCode = 200
	if responseStatus(ctx, statusCode) != "ok" {
		t.Fatal(responseStatus(ctx, statusCode))
	}
	statusCode = 500
	if responseStatus(ctx, statusCode) != "error" {
		t.Fatal(responseStatus(ctx, statusCode))
	}
	var cancel func()
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if responseStatus(ctx, statusCode) != "cancelled" {
		t.Fatal(responseStatus(ctx, statusCode))
	}
	ctx, cancel = context.WithDeadline(context.Background(), time.Time{})
	defer cancel()
	if responseStatus(ctx, statusCode) != "timeout" {
		t.Fatal(responseStatus(ctx, statusCode))
	}
}
