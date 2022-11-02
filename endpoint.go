package prate

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/julienschmidt/httprouter"
)

var requestDataPool = sync.Pool{
	New: func() interface{} {
		return new(RequestData)
	},
}

type endpoint struct {
	method         string
	path           string
	handler        Handler
	requestPayload protoreflect.ProtoMessage
	// responsePayload protoreflect.ProtoMessage
	mexclusions []string
	requestPool sync.Pool
}

func (ep *endpoint) initPools() {
	if ep.requestPayload != nil {
		ep.requestPool = sync.Pool{
			New: func() interface{} {
				inpt := reflect.TypeOf(ep.requestPayload)
				switch inpt.Kind() {
				case reflect.Array, reflect.Chan,
					reflect.Map, reflect.Ptr, reflect.Slice:
					inpt = inpt.Elem()
				}
				val := reflect.ValueOf(ep.requestPayload)
				v := reflect.New(inpt).Elem()
				v.Set(val.Elem())
				return v.Addr()
			},
		}
	}
}

// func (ep endpoint) pathDetails() (string, []string) {
// 	// qps := queryParams(ep.Payload)
// 	params := pathParams(ep.path)
// 	if len(params) == 0 {
// 		return ep.path, nil
// 	}
// 	r := ep.path
// 	for i, param := range params {
// 		fr := fmt.Sprintf("{%s}", strings.Trim(strings.Replace(param, ":", "", 1), "/"))
// 		r = strings.Replace(r, param, fr, 1)
// 		pname := strings.Trim(strings.Replace(param, ":", "", 1), "/")
// 		params[i] = pname
// 	}
// 	return r, params
// }

// func (ep endpoint) requestSchema() (openapi3.Schema, error) {
// 	s, err := schemaFromType(reflect.TypeOf(ep.handler).In(1))
// 	if err != nil {
// 		return *openapi3.NewSchema(), wrapErr(err)
// 	}
// 	return s, nil
// }

// func (ep endpoint) responseSchema() (openapi3.Schema, error) {
// 	t := reflect.TypeOf(ep.handler)
// 	if t.Kind() != reflect.Func {
// 		return openapi3.Schema{}, wrapErr(fmt.Errorf("type if not a func"))
// 	}
// 	s, err := schemaFromType(t.Out(0))
// 	if err != nil {
// 		return openapi3.Schema{}, wrapErr(err)
// 	}
// 	return s, nil
// }

// func (ep endpoint) generatePathItem() {
// 	op := openapi3.NewOperation()
// 	op.OperationID = ep.route
// 	formattedRoute, params, queryParams := ep.routeDetails()

// }

func (ep *endpoint) handle(f func(string, httprouter.Handle)) {
	f(ep.path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		rd, ok := requestDataPool.Get().(*RequestData)
		if !ok {
			panic(wrapErr(fmt.Errorf("requestDataPool returned not *RequestData.... aaaaaaa")))
		}
		defer func() {
			rd.Custom = nil
			requestDataPool.Put(rd)
		}()
		rd.Custom = map[string]interface{}{}
		rd.Params = params

		badrequest := func(msg string) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(StatusBadRequest)
			w.Write([]byte(msg))
		}

		// Request Payload
		if ep.requestPayload != nil {
			pv := ep.requestPool.Get()
			if pv == nil {
				panic(wrapErr(fmt.Errorf("requestPayload Pool returned nil....aaaaaaaa")))
			}
			defer ep.requestPool.Put(pv)

			v, ok := pv.(reflect.Value)
			if !ok {
				panic(wrapErr(fmt.Errorf("requestBodyPool returned value of type not equal to reflect.Value")))
			}
			reflect.ValueOf(rd).Elem().FieldByName("Body").Set(v)

			bs, err := io.ReadAll(r.Body)
			if err != nil {
				// log.Println(wrapErr(err, "readall failed"))
				if err != io.EOF {
					log.Println(wrapErr(err))
					badrequest("connection error")
					return
				}
			}

			if len(bs) > 0 {
				if err := proto.Unmarshal(bs, rd.Body); err != nil {
					log.Println(wrapErr(err, "request unmarshal failed"))
					badrequest("invalid payload")
					return
				}
			} else {
				badrequest("empty payload")
				return
			}
		}

		rc, ok := rcPool.Get().(*RequestCtx)
		if !ok {
			panic(`rcpool returned something thats not a RequestCtx... aaaaaaaaa!!`)
		}
		defer func() {
			rc.reset()
			rcPool.Put(rc)
		}()
		rc.update(w, r)

		resp, err := ep.handler(rc, rd)
		if err != nil {
			if err := errorHandler(rc, err); err != nil {
				log.Println(wrapErr(err))
			}
			return
		}

		if rc.ResponseWriter.written {
			return
		}

		var resBody []byte
		err = nil
		if resp != nil {
			resBody, err = proto.Marshal(resp)
			if err != nil {
				log.Println(wrapErr(err))
				if err := errorHandler(rc, NewError(StatusInternalServerError)); err != nil {
					log.Println(wrapErr(err))
				}
				return
			}
			rc.ResponseWriter.Header().Set("Content-Type", ContentTypePROTO.String())
		}
		rc.ResponseWriter.WriteHeader(StatusOK)
		rc.ResponseWriter.Write(resBody)
	})
}

// func (ep *endpoint) pathItem() (*openapi3.PathItem, error) {
// 	// TODO
// 	return &openapi3.PathItem{}, nil
// }

// type EndpointPayload struct {
// 	RequestPayload  protoreflect.ProtoMessage
// }

// func NewEndpointPayload(ps ...protoreflect.ProtoMessage) EndpointPayload {
// 	var ep EndpointPayload
// 	if len(ps) > 0 {
// 		ep.RequestPayload = ps[0]
// 	}

// 	if len(ps) > 1 {
// 		ep.ResponsePayload = ps[2]
// 	}
// 	return ep
// }

type EndpointConfig struct {
	Path               string
	Handler            Handler
	RequestPayloadType protoreflect.ProtoMessage
	ExcludeMiddlewares []string
	method             string
}

func NewEndpointConfig(path string, handler Handler) EndpointConfig {
	return EndpointConfig{
		Path:    path,
		Handler: handler,
	}
}

func (ec EndpointConfig) WithExclude(ms ...string) EndpointConfig {
	ec.ExcludeMiddlewares = append(ec.ExcludeMiddlewares, ms...)
	return ec
}

func (ec EndpointConfig) WithHandler(h Handler) EndpointConfig {
	ec.Handler = h
	return ec
}

func (ec EndpointConfig) WithRequestPayloadType(pt protoreflect.ProtoMessage) EndpointConfig {
	ec.RequestPayloadType = pt
	return ec
}

func (ec EndpointConfig) WithPath(p string) EndpointConfig {
	ec.Path = p
	return ec
}

func (ec *EndpointConfig) applyMiddlerwares(ms []*Middleware) {
	exm := map[string]bool{}
	for _, s := range ec.ExcludeMiddlewares {
		exm[s] = true
	}

	for i := len(ms) - 1; i >= 0; i-- {
		m := ms[i]
		if _, ok := exm[m.ID]; ok {
			continue
		}
		ec.Handler = m.Handler(ec.Handler)
	}
}

func (ec EndpointConfig) endpoint() *endpoint {
	ep := &endpoint{
		method:         ec.method,
		path:           ec.Path,
		handler:        ec.Handler,
		requestPayload: ec.RequestPayloadType,
		mexclusions:    ec.ExcludeMiddlewares,
	}
	ep.initPools()
	return ep
}
