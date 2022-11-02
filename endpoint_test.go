package gate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/daimaou92/gate/pb/fortest"
	"github.com/julienschmidt/httprouter"
)

// var (
// 	tstReqBody   = &testPld{}
// 	tstQueryBody = &QueryPayload{}
// )

func TestEndpointHandle(t *testing.T) {
	type tt struct {
		port           int
		name           string
		ec             EndpointConfig
		url            string
		requestPayload protoreflect.ProtoMessage
		output         protoreflect.ProtoMessage
		outStatus      int
		routerFunc     func(string, httprouter.Handle)
	}

	// var port = 2525
	router := httprouter.New()
	tsts := []tt{
		{
			port: 4444,
			name: "valid",
			ec: EndpointConfig{
				Path:               "/1/:namevalid",
				method:             http.MethodPost,
				RequestPayloadType: &fortest.TestReq{},
				Handler: func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
					return &fortest.TestRes{
						Key:   "a",
						Value: "b",
					}, nil
				},
			},
			routerFunc: router.POST,
			url:        "http://localhost:4444/1/paul?key=value",
			requestPayload: &fortest.TestReq{
				Key:   "a",
				Value: "b",
			},
			output: &fortest.TestRes{
				Key:   "a",
				Value: "b",
			},
			outStatus: StatusOK,
		}, {
			port: 2525,
			name: "Request Body Missing Error",
			ec: EndpointConfig{
				Path:               "/3/:namerbme",
				RequestPayloadType: &fortest.TestReq{},
				Handler: func(rc *RequestCtx, rd *RequestData) (protoreflect.ProtoMessage, error) {
					return nil, nil
				},
				method: http.MethodPost,
			},
			routerFunc: router.POST,
			url:        "http://localhost:2525/3/paul?key=value",
			output:     nil,
			outStatus:  StatusBadRequest,
		},
	}

	for _, tst := range tsts {
		t.Run(tst.name, func(t *testing.T) {
			ep := tst.ec.endpoint()
			ep.handle(tst.routerFunc)

			server := &http.Server{
				Addr:    fmt.Sprintf(":%d", tst.port),
				Handler: router,
			}
			go func() {
				if err := server.ListenAndServe(); err != nil {
					if err != http.ErrServerClosed {
						log.Println(wrapErr(err))
					}
				}
			}()
			var (
				req *http.Request
				err error
			)
			if tst.requestPayload != nil {
				bs, _ := proto.Marshal(tst.requestPayload)
				req, err = http.NewRequest(tst.ec.method, tst.url, bytes.NewBuffer(bs))
			} else {
				req, err = http.NewRequest(tst.ec.method, tst.url, nil)
			}
			if err != nil {
				server.Shutdown(context.TODO())
				t.Fatal(err)
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				server.Shutdown(context.TODO())
				t.Fatal(err)
			}
			if res.StatusCode != tst.outStatus {
				server.Shutdown(context.TODO())
				t.Fatalf("received code: %d. Wanted: %d", res.StatusCode, tst.outStatus)
			}

			if tst.output != nil {
				bs, err := io.ReadAll(res.Body)
				if err != nil {
					server.Shutdown(context.TODO())
					t.Fatal(err)
				}
				defer res.Body.Close()
				tbs, _ := proto.Marshal(tst.output)
				if !bytes.Equal(bs, tbs) {
					server.Shutdown(context.TODO())
					t.Fatalf("wanted: %s\nGot: %s\n", tbs, bs)
				}
			}
			server.Shutdown(context.TODO())
		})
	}
}
