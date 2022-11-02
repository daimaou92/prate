package prate

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/daimaou92/prate/pb/fortest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestStart(t *testing.T) {
	type tt struct {
		name               string
		ao                 AppOptions
		method             string
		url                string
		path               string
		reqBody            protoreflect.ProtoMessage
		handler            Handler
		resBody            protoreflect.ProtoMessage
		statusCode         int
		middlewares        []*Middleware
		excludeMiddlewares []string
		testHeaderKey      string
		testHeaderVal      []string
	}

	tsts := []tt{
		{
			name:   http.MethodGet,
			method: http.MethodGet,
			url:    "http://localhost:5050/sarkar",
			path:   "/:name",
			ao: AppOptions{
				Addr: ":5050",
			},
			resBody: &fortest.TestRes{
				Key:   "name",
				Value: "sarkar",
			},
			handler: func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
				n := rd.Params.ByName("name")
				return &fortest.TestRes{
					Key:   "name",
					Value: n,
				}, nil
			},
			statusCode: StatusOK,
		}, {
			name:   http.MethodPost,
			method: http.MethodPost,
			url:    "http://localhost:4747/sarkar",
			path:   "/:name",
			ao: AppOptions{
				Addr: ":4747",
			},
			reqBody: &fortest.TestReq{
				Key:   "Key",
				Value: "Value",
			},
			resBody: &fortest.TestRes{
				Key:   "Key",
				Value: "Value - sarkar",
			},
			handler: func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
				j, ok := rd.Body.(*fortest.TestReq)
				if !ok {
					return nil, ErrBadRequest
				}
				n := rd.Params.ByName("name")
				tres := &fortest.TestRes{
					Key:   j.Key,
					Value: fmt.Sprintf("%s - %s", j.Value, n),
				}
				return tres, nil
			},
			statusCode: StatusOK,
		}, {
			name:   http.MethodPatch,
			method: http.MethodPatch,
			url:    "http://localhost:2987/sarkar",
			path:   "/:name",
			ao: AppOptions{
				Addr: ":2987",
			},
			reqBody: &fortest.TestReq{
				Key:   "Key",
				Value: "Value",
			},
			resBody: &fortest.TestRes{
				Key:   "Key",
				Value: "Value - sarkar",
			},
			handler: func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
				j, ok := rd.Body.(*fortest.TestReq)
				if !ok {
					return nil, ErrBadRequest
				}
				n := rd.Params.ByName("name")
				tres := &fortest.TestRes{
					Key:   j.Key,
					Value: fmt.Sprintf("%s - %s", j.Value, n),
				}
				return tres, nil
			},
			statusCode: StatusOK,
		}, {
			name:   http.MethodPut,
			method: http.MethodPut,
			url:    "http://localhost:8678/sarkar",
			path:   "/:name",
			ao: AppOptions{
				Addr: ":8678",
			},
			reqBody: &fortest.TestReq{
				Key:   "Key",
				Value: "Value",
			},
			resBody: &fortest.TestRes{
				Key:   "Key",
				Value: "Value - sarkar",
			},
			handler: func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
				j, ok := rd.Body.(*fortest.TestReq)
				if !ok {
					return nil, ErrBadRequest
				}
				n := rd.Params.ByName("name")
				tres := &fortest.TestRes{
					Key:   j.Key,
					Value: fmt.Sprintf("%s - %s", j.Value, n),
				}
				return tres, nil
			},
			statusCode: StatusOK,
		}, {
			name:   http.MethodDelete,
			method: http.MethodDelete,
			url:    "http://localhost:9999/sarkar",
			path:   "/:name",
			ao: AppOptions{
				Addr: ":9999",
			},
			reqBody: &fortest.TestReq{
				Key:   "Key",
				Value: "Value",
			},
			resBody: &fortest.TestRes{
				Key:   "Key",
				Value: "Value - sarkar",
			},
			handler: func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
				j, ok := rd.Body.(*fortest.TestReq)
				if !ok {
					return nil, ErrBadRequest
				}
				n := rd.Params.ByName("name")
				tres := &fortest.TestRes{
					Key:   j.Key,
					Value: fmt.Sprintf("%s - %s", j.Value, n),
				}
				return tres, nil
			},
			statusCode: StatusOK,
		}, {
			name:   "testMiddleware",
			method: http.MethodGet,
			url:    "http://localhost:12345/sarkar",
			path:   "/:name",
			ao: AppOptions{
				Addr: ":12345",
			},
			handler: func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
				return &fortest.TestRes{}, nil
			},
			statusCode: StatusOK,
			middlewares: []*Middleware{
				{
					ID: "setPinacolada",
					Handler: func(h Handler) Handler {
						return func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
							res, err := h(rc, rd)
							rc.ResponseWriter.Header().Set("pina", "colada")
							return res, err
						}
					},
				},
			},
			testHeaderKey: "pina",
			testHeaderVal: []string{"colada"},
		}, {
			name:   "testMultiMiddleware",
			method: http.MethodGet,
			url:    "http://localhost:23456/sarkar",
			path:   "/:name",
			ao: AppOptions{
				Addr: ":23456",
			},
			handler: func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
				return &fortest.TestRes{}, nil
			},
			statusCode: StatusOK,
			middlewares: []*Middleware{
				{
					ID: "addPinaColada",
					Handler: func(h Handler) Handler {
						return func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
							res, err := h(rc, rd)
							rc.ResponseWriter.Header().Add("pina", "colada")
							return res, err
						}
					},
				}, {
					ID: "addPinaFire",
					Handler: func(h Handler) Handler {
						return func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
							res, err := h(rc, rd)
							rc.ResponseWriter.Header().Add("pina", "fire")
							return res, err
						}
					},
				},
			},
			testHeaderKey: "pina",
			testHeaderVal: []string{"fire", "colada"},
		}, {
			name:   "testExcludeMiddleware",
			method: http.MethodGet,
			url:    "http://localhost:34567/sarkar",
			path:   "/:name",
			ao: AppOptions{
				Addr: ":34567",
			},
			handler: func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
				return &fortest.TestRes{}, nil
			},
			statusCode: StatusOK,
			middlewares: []*Middleware{
				{
					ID: "addPinaColada",
					Handler: func(h Handler) Handler {
						return func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
							res, err := h(rc, rd)
							rc.ResponseWriter.Header().Add("pina", "colada")
							return res, err
						}
					},
				}, {
					ID: "addPinaFire",
					Handler: func(h Handler) Handler {
						return func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
							res, err := h(rc, rd)
							rc.ResponseWriter.Header().Add("pina", "fire")
							return res, err
						}
					},
				},
			},
			excludeMiddlewares: []string{"addPinaFire"},
			testHeaderKey:      "pina",
			testHeaderVal:      []string{"colada"},
		},
	}

	type HandleFuncType func(EndpointConfig)
	for _, tst := range tsts {
		t.Run(tst.name, func(t *testing.T) {
			app, err := New(tst.ao)
			if err != nil {
				t.Fatal(err)
			}
			var f HandleFuncType
			switch tst.method {
			case http.MethodGet:
				f = app.GET
			case http.MethodPost:
				f = app.POST
			case http.MethodDelete:
				f = app.DELETE
			case http.MethodHead:
				f = app.HEAD
			case http.MethodOptions:
				f = app.OPTIONS
			case http.MethodPut:
				f = app.PUT
			case http.MethodPatch:
				f = app.PATCH
			default:
				log.Fatalf("invalid method: %s", tst.method)
			}
			var rpt protoreflect.ProtoMessage
			if tst.reqBody != nil {
				rpt = tst.reqBody
			}
			f(EndpointConfig{
				Path:               tst.path,
				RequestPayloadType: rpt,
				Handler:            tst.handler,
				ExcludeMiddlewares: tst.excludeMiddlewares,
				method:             tst.method,
			})

			app.Apply(tst.middlewares...)

			// Start server
			go func() {
				if err := app.Start(); err != nil {
					log.Println(wrapErr(err, "listen failed"))
				}
			}()

			var (
				r *http.Request
			)
			if tst.reqBody != nil {
				bs, err := proto.Marshal(tst.reqBody)
				if err != nil {
					t.Fatalf("marshal failed: %s", err.Error())
				}
				r, err = http.NewRequest(tst.method, tst.url, bytes.NewBuffer(bs))
				if err != nil {
					t.Fatalf("newrequest failed: %s", err.Error())
				}
			} else {
				var err error
				r, err = http.NewRequest(tst.method, tst.url, nil)
				if err != nil {
					t.Fatalf("newrequest, emptybody failed: %s", err.Error())
				}
			}
			res, err := http.DefaultClient.Do(r)
			if err != nil {
				log.Printf("making client request faild: %s", err.Error())
			}
			defer res.Body.Close()

			if res.StatusCode != tst.statusCode {
				t.Fatalf("statuscode wanted: %d. got %d", tst.statusCode, res.StatusCode)
			}
			if tst.resBody != nil {
				bs, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("res body readall failed: %s", err.Error())
				}
				bsw, err := proto.Marshal(tst.resBody)
				if err != nil {
					t.Fatalf("tst.resBody marshal failed: %s", err.Error())
				}

				if !bytes.Equal(bsw, bs) {
					t.Fatalf("wanted %s. got %s", bsw, bs)
				}
			}

			if tst.testHeaderKey != "" {
				if _, ok := res.Header[http.CanonicalHeaderKey(tst.testHeaderKey)]; !ok {
					t.Fatalf("header key not present")
				}
			}

			if len(tst.testHeaderVal) > 0 {
				vs := res.Header[http.CanonicalHeaderKey(tst.testHeaderKey)]
				if len(vs) != len(tst.testHeaderVal) {
					t.Fatalf(
						"lengths don't match. wanted: %d. Got %d",
						len(tst.testHeaderVal), len(vs),
					)
				}
				for i, v := range vs {
					if tst.testHeaderVal[i] != v {
						t.Fatalf(
							"header val at index: %d did not match. wanted: %s. Got: %s\n",
							i, tst.testHeaderVal[i], v,
						)
					}
				}
			}
		})
	}
}
