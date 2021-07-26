// GENERATED BY 'T'ransport 'G'enerator. DO NOT EDIT.
package transport

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	goUUID "github.com/google/uuid"
	otg "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	zipkinTracer "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"github.com/openzipkin/zipkin-go"
	httpReporter "github.com/openzipkin/zipkin-go/reporter/http"
	"github.com/rs/zerolog"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
)

func (srv *Server) TraceJaeger(serviceName string) *Server {

	environment, _ := os.LookupEnv("ENV")

	cfg, err := config.FromEnv()
	ExitOnError(srv.log, err, "jaeger config err")

	if cfg.ServiceName == "" {
		cfg.ServiceName = environment + serviceName
	}

	var trace otg.Tracer
	trace, srv.reporterCloser, err = cfg.NewTracer(config.Logger(log.NullLogger), config.Metrics(metrics.NullFactory))

	ExitOnError(srv.log, err, "could not create jaeger tracer")

	otg.SetGlobalTracer(trace)
	return srv
}

func (srv *Server) TraceZipkin(serviceName string, zipkinUrl string) *Server {

	reporter := httpReporter.NewReporter(zipkinUrl)
	srv.reporterCloser = reporter

	environment, envExists := os.LookupEnv("ENV")

	if envExists {
		serviceName = environment + serviceName
	}

	endpoint, err := zipkin.NewEndpoint(serviceName, "")
	ExitOnError(srv.log, err, "could not create endpoint")

	nativeTracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint))
	ExitOnError(srv.log, err, "could not create tracer")

	trace := zipkinTracer.Wrap(nativeTracer)
	otg.SetGlobalTracer(trace)

	return srv
}

func injectSpan(log zerolog.Logger, span otg.Span, ctx *fiber.Ctx) {
	headers := make(http.Header)
	if err := otg.GlobalTracer().Inject(span.Context(), otg.HTTPHeaders, otg.HTTPHeadersCarrier(headers)); err != nil {
		log.Debug().Err(err).Msg("inject span to HTTP headers")
	}
	for key, values := range headers {
		ctx.Response().Header.Set(key, strings.Join(values, ";"))
	}
	ctx.Response().Header.SetBytesV(headerRequestID, ctx.Request().Header.Peek(headerRequestID))
}

func extractSpan(log zerolog.Logger, opName string, ctx *fiber.Ctx) (span otg.Span) {
	headers := make(http.Header)
	requestID := string(ctx.Request().Header.Peek(headerRequestID))
	if requestID == "" {
		requestID = goUUID.New().String()
	}
	ctx.Request().Header.VisitAll(func(key, value []byte) {
		headers.Set(string(key), string(value))
	})
	var opts []otg.StartSpanOption
	wireContext, err := otg.GlobalTracer().Extract(otg.HTTPHeaders, otg.HTTPHeadersCarrier(headers))
	if err != nil {
		log.Debug().Err(err).Msg("extract span from HTTP headers")
	} else {
		opts = append(opts, otg.ChildOf(wireContext))
	}
	span = otg.GlobalTracer().StartSpan(opName, opts...)
	ext.HTTPUrl.Set(span, ctx.OriginalURL())
	ext.HTTPMethod.Set(span, ctx.Method())
	span.SetTag("requestID", requestID)
	ctx.Request().Header.Set(headerRequestID, requestID)
	ctx.Context().SetUserValue(headerRequestID, requestID)
	return
}

func toString(value interface{}) string {
	data, _ := json.Marshal(value)
	return string(data)
}
