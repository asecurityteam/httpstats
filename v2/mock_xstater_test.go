// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/rs/xstats (interfaces: XStater,Sender)

package httpstats

import (
	time "time"

	gomock "go.uber.org/mock/gomock"
)

// Mock of XStater interface
type MockXStater struct {
	ctrl     *gomock.Controller
	recorder *_MockXStaterRecorder
}

// Recorder for MockXStater (not exported)
type _MockXStaterRecorder struct {
	mock *MockXStater
}

func NewMockXStater(ctrl *gomock.Controller) *MockXStater {
	mock := &MockXStater{ctrl: ctrl}
	mock.recorder = &_MockXStaterRecorder{mock}
	return mock
}

func (_m *MockXStater) EXPECT() *_MockXStaterRecorder {
	return _m.recorder
}

func (_m *MockXStater) AddTags(_param0 ...string) {
	_s := []interface{}{}
	for _, _x := range _param0 {
		_s = append(_s, _x)
	}
	_m.ctrl.Call(_m, "AddTags", _s...)
}

func (_mr *_MockXStaterRecorder) AddTags(arg0 ...interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "AddTags", arg0...)
}

func (_m *MockXStater) Count(_param0 string, _param1 float64, _param2 ...string) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	_m.ctrl.Call(_m, "Count", _s...)
}

func (_mr *_MockXStaterRecorder) Count(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Count", _s...)
}

func (_m *MockXStater) Gauge(_param0 string, _param1 float64, _param2 ...string) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	_m.ctrl.Call(_m, "Gauge", _s...)
}

func (_mr *_MockXStaterRecorder) Gauge(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Gauge", _s...)
}

func (_m *MockXStater) GetTags() []string {
	ret := _m.ctrl.Call(_m, "GetTags")
	ret0, _ := ret[0].([]string)
	return ret0
}

func (_mr *_MockXStaterRecorder) GetTags() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetTags")
}

func (_m *MockXStater) Histogram(_param0 string, _param1 float64, _param2 ...string) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	_m.ctrl.Call(_m, "Histogram", _s...)
}

func (_mr *_MockXStaterRecorder) Histogram(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Histogram", _s...)
}

func (_m *MockXStater) Timing(_param0 string, _param1 time.Duration, _param2 ...string) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	_m.ctrl.Call(_m, "Timing", _s...)
}

func (_mr *_MockXStaterRecorder) Timing(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Timing", _s...)
}

// Mock of Sender interface
type MockSender struct {
	ctrl     *gomock.Controller
	recorder *_MockSenderRecorder
}

// Recorder for MockSender (not exported)
type _MockSenderRecorder struct {
	mock *MockSender
}

func NewMockSender(ctrl *gomock.Controller) *MockSender {
	mock := &MockSender{ctrl: ctrl}
	mock.recorder = &_MockSenderRecorder{mock}
	return mock
}

func (_m *MockSender) EXPECT() *_MockSenderRecorder {
	return _m.recorder
}

func (_m *MockSender) Count(_param0 string, _param1 float64, _param2 ...string) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	_m.ctrl.Call(_m, "Count", _s...)
}

func (_mr *_MockSenderRecorder) Count(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Count", _s...)
}

func (_m *MockSender) Gauge(_param0 string, _param1 float64, _param2 ...string) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	_m.ctrl.Call(_m, "Gauge", _s...)
}

func (_mr *_MockSenderRecorder) Gauge(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Gauge", _s...)
}

func (_m *MockSender) Histogram(_param0 string, _param1 float64, _param2 ...string) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	_m.ctrl.Call(_m, "Histogram", _s...)
}

func (_mr *_MockSenderRecorder) Histogram(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Histogram", _s...)
}

func (_m *MockSender) Timing(_param0 string, _param1 time.Duration, _param2 ...string) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	_m.ctrl.Call(_m, "Timing", _s...)
}

func (_mr *_MockSenderRecorder) Timing(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Timing", _s...)
}
