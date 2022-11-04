package prate

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"google.golang.org/protobuf/proto"
)

type epInit struct {
	ec EndpointConfig
	f  func(string, httprouter.Handle)
}

// The gate App type
type App struct {
	http.Server
	router      *httprouter.Router
	middlewares []*Middleware
	mwareIndex  map[string]int
	epCache     []epInit
}

// Conforms with the type accepted by the panic handler of httprouter
type AppPanicHandler func(http.ResponseWriter, *http.Request, interface{})

type AppOptions struct {
	Addr              string
	TLSConfig         *tls.Config
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	TLSNextProto      map[string]func(*App, *tls.Conn)
	ConnState         func(net.Conn, http.ConnState)
	ErrorLog          *log.Logger
	BaseContext       func(net.Listener) context.Context
	ConnContext       func(ctx context.Context, c net.Conn) context.Context
}

func (ao AppOptions) server() *http.Server {
	server := http.Server{
		Addr: ":0",
	}
	if ao.Addr != "" {
		server.Addr = ao.Addr
	}
	if ao.TLSConfig != nil {
		server.TLSConfig = ao.TLSConfig
	}

	if int64(ao.ReadTimeout) > 0 {
		server.ReadTimeout = ao.ReadTimeout
	}

	if int64(ao.ReadHeaderTimeout) > 0 {
		server.ReadHeaderTimeout = ao.ReadHeaderTimeout
	}

	if int64(ao.WriteTimeout) > 0 {
		server.WriteTimeout = ao.WriteTimeout
	}

	if int64(ao.IdleTimeout) > 0 {
		server.IdleTimeout = ao.IdleTimeout
	}

	if ao.MaxHeaderBytes > 0 {
		server.MaxHeaderBytes = ao.MaxHeaderBytes
	}

	if len(ao.TLSNextProto) > 0 {
		v := map[string]func(*http.Server, *tls.Conn, http.Handler){}
		for k, f := range ao.TLSNextProto {
			v[k] = func(s *http.Server, c *tls.Conn, h http.Handler) {
				a := new(App)
				a.FromServer(s)
				a.router = newRouter()
				a.Handler = a.router
				f(a, c)
			}
		}
	}
	return &server
}

// This is used to define custom behaviour when a route is requested
// with an incorrect method type. At least one registration for any method
// needs to exist for the given path; otherwise a 404 will be triggered.
func (a *App) SetMethodNotAllowedHandler(h http.Handler) error {
	if a == nil || a.router == nil {
		return wrapErr(fmt.Errorf("app not initialized"))
	}

	a.router.HandleMethodNotAllowed = true
	a.router.MethodNotAllowed = h
	return nil
}

// If called with `true` the app will start replying with
// automatically with error code 405 when a configured route
// exists but not for the method type requested. This behaviour
// can be changed by calling SetMethodNotAllowed with an appropriate
// http.Handler
func (a *App) HandleMethodNotAllowed(b bool) error {
	if a == nil || a.router == nil {
		return wrapErr(fmt.Errorf("app not initialized"))
	}

	a.router.HandleMethodNotAllowed = b
	return nil
}

// This determines whether the app should intercept OPTIONS
// requests and handle them automatically using a pre-set handler.
// Handlers can be pre-set using SetOptionsHandler and
// SetGlobalOptionsHandler
func (a *App) HandleOPTIONS(b bool) error {
	if a == nil || a.router == nil {
		return wrapErr(fmt.Errorf("app not initialized"))
	}

	a.router.HandleOPTIONS = b
	return nil
}

// This and SetGlobalOptionsHandler differ in the handler type
// they accept. This accepts a http.Handler as opposed to gate.Handler
func (a *App) SetOptionsHandler(h http.Handler) error {
	if a == nil || a.router == nil {
		return wrapErr(fmt.Errorf("app not initialized"))
	}
	a.router.HandleOPTIONS = true
	a.router.GlobalOPTIONS = h
	return nil
}

// Sets a CustomHandler whenever the server encounters a panic
// Default behaviour is to respond back with a generic
// 500 Internal Server Error. The type AppPanicHandler mirrors
// the type of the argument required by httprouter.
func (a *App) SetPanicHandler(h AppPanicHandler) error {
	if a == nil || a.router == nil {
		return wrapErr(fmt.Errorf("app not initialized"))
	}
	a.router.PanicHandler = h
	return nil
}

func newRouter() *httprouter.Router {
	r := httprouter.New()
	r.HandleMethodNotAllowed = true

	r.HandleOPTIONS = true
	return r
}

// Create a new gate.App. The internal AppOptions.Info
// attribute cannot be empty. Infact AppOptions.Info must
// have valid values for attributes `Title` and `Version`
func New(ao AppOptions) (*App, error) {
	server := ao.server()
	app := &App{}
	app.router = newRouter()
	app.Handler = app.router
	app.mwareIndex = map[string]int{}
	app.FromServer(server)
	return app, nil
}

// Populates attributes from a *http.Server
func (a *App) FromServer(server *http.Server) {
	a.Addr = server.Addr
	a.TLSConfig = server.TLSConfig
	a.ReadTimeout = server.ReadTimeout
	a.ReadHeaderTimeout = server.ReadHeaderTimeout
	a.WriteTimeout = server.WriteTimeout
	a.IdleTimeout = server.IdleTimeout
	a.MaxHeaderBytes = server.MaxHeaderBytes
	a.TLSNextProto = server.TLSNextProto
}

// Implements http.Handler interface
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

func errorHandler(rc *RequestCtx, err error) error {
	code := StatusInternalServerError
	if e, ok := err.(*Error); ok {
		code = e.Code
	}
	rc.ResponseWriter.WriteHeader(code)
	if _, err := rc.ResponseWriter.Write([]byte(err.Error())); err != nil {
		return err
	}
	return nil
}

// Called before listen
func (app *App) mountEndpoints() {
	for _, v := range app.epCache {
		v.ec.applyMiddlerwares(app.middlewares)
		ep := v.ec.endpoint()
		ep.handle(v.f)
	}
}

func (app *App) registerEndpoint(
	ec EndpointConfig,
	f func(string, httprouter.Handle),
) {
	app.epCache = append(app.epCache, epInit{
		ec: ec,
		f:  f,
	})
}

// Add a GET endpoint
func (app *App) GET(ec EndpointConfig) {
	ec.method = http.MethodGet
	app.registerEndpoint(
		ec, app.router.GET,
	)
}

// Add a POST endpoint
func (app *App) POST(ec EndpointConfig) {
	ec.method = http.MethodPost
	app.registerEndpoint(
		ec, app.router.POST,
	)
}

// Add a DELETE endpoint
func (app *App) DELETE(ec EndpointConfig) {
	ec.method = http.MethodDelete
	app.registerEndpoint(
		ec, app.router.DELETE,
	)
}

// Add a PUT endpoint
func (app *App) PUT(ec EndpointConfig) {
	ec.method = http.MethodPut
	app.registerEndpoint(
		ec, app.router.PUT,
	)
}

// Add a PATCH endpoint
func (app *App) PATCH(ec EndpointConfig) {
	ec.method = http.MethodPatch
	app.registerEndpoint(
		ec, app.router.PATCH,
	)
}

// Add a OPTIONS endpoint
func (app *App) OPTIONS(ec EndpointConfig) {
	ec.method = http.MethodOptions
	app.registerEndpoint(
		ec, app.router.OPTIONS,
	)
}

// Add a HEAD endpoint
func (app *App) HEAD(ec EndpointConfig) {
	ec.method = http.MethodHead
	app.registerEndpoint(
		ec, app.router.HEAD,
	)
}

func (app *App) addMiddleware(m *Middleware) error {
	if !m.valid(app) {
		return wrapErr(fmt.Errorf("invalid middleware"))
	}
	app.middlewares = append(app.middlewares, m)
	if app.mwareIndex == nil {
		app.mwareIndex = map[string]int{}
	}
	app.mwareIndex[m.ID] = len(app.middlewares) - 1
	return nil
}

// This function is used to add middlewares.
// The order in which middlewares are added is important.
// The first middleware added ("Apply"-ed) will be called first
// and so on.
func (app *App) Apply(ms ...*Middleware) error {
	for _, m := range ms {
		if err := app.addMiddleware(m); err != nil {
			return wrapErr(err)
		}
	}
	return nil
}

// Used to set a global handler for the HTTP Method of type OPTIONS.
func (app *App) SetGlobalOptionsHandler(h Handler) {
	app.router.GlobalOPTIONS = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rc := RequestCtx{
			ResponseWriter: &ResponseWriter{
				rw: rw,
			},
			Request: r,
		}

		rd := RequestData{
			Params: httprouter.ParamsFromContext(r.Context()),
		}
		// if len(r.URL.RawQuery) > 0 {
		// 	qp := QueryPayload(r.URL.Query())
		// 	rd.QueryParams = &qp
		// }

		res, err := h(&rc, &rd)
		if err != nil {
			if err := errorHandler(&rc, err); err != nil {
				log.Println(wrapErr(err))
			}
			return
		}

		if rc.ResponseWriter.written {
			return
		}

		var bs []byte
		if res != nil {
			bs, err = proto.Marshal(res)
			if err != nil {
				if err := errorHandler(&rc, err); err != nil {
					log.Println(wrapErr(err))
				}
				return
			}
		}
		rc.ResponseWriter.WriteHeader(StatusOK)
		rc.ResponseWriter.Write(bs)
	})
}

// Starts the server
func (app *App) Start() error {
	if app == nil || app.router == nil {
		return wrapErr(fmt.Errorf("app not initialized"))
	}

	app.mountEndpoints()
	conn, err := net.Listen("tcp", app.Addr)
	if err != nil {
		return wrapErr(err)
	}
	tlsListener := tls.NewListener(conn, app.TLSConfig)
	log.Println("Listening at: ", tlsListener.Addr())
	if err := app.Serve(tlsListener); err != nil {
		return wrapErr(err)
	}
	return nil
}
