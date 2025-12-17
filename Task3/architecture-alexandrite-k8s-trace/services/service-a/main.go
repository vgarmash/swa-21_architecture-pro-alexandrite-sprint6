package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

var tracer = otel.Tracer("service-a")

func main() {
    // Настройка zerolog (опционально)
    zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

    ctx := context.Background()

    tp, err := initTracer(ctx)
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to initialize tracer")
    }
    defer func() {
        if err := tp.Shutdown(ctx); err != nil {
            log.Error().Err(err).Msg("Error shutting down tracer provider")
        }
    }()

    handler := otelhttp.NewHandler(
        http.HandlerFunc(handleRequest),
        "service-a",
        otelhttp.WithTracerProvider(tp),
    )

    http.Handle("/", handler)

    log.Info().Msg("Service A starting on :8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal().Err(err).Msg("Failed to start server")
    }
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithInsecure(),
        otlptracegrpc.WithEndpoint("simplest-collector:4317"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create exporter: %w", err)
    }

    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName("service-a"),
            semconv.ServiceVersion("1.0.0"),
            semconv.DeploymentEnvironmentNameKey.String("development"),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
    )

    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    return tp, nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    ctx, span := tracer.Start(r.Context(), "service-a-handler")
    defer span.End()

    log.Info().Msg("Service A: Received request")
    time.Sleep(50 * time.Millisecond)
    callServiceB(ctx)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status": "ok", "service": "a", "message": "Hello from Service A!"}`))
}

func callServiceB(ctx context.Context) {
    ctx, span := tracer.Start(ctx, "call-service-b")
    defer span.End()

    log.Info().Msg("Service A: Calling Service B...")

    client := http.Client{
        Transport: otelhttp.NewTransport(http.DefaultTransport),
        Timeout:   5 * time.Second,
    }

    req, err := http.NewRequestWithContext(ctx, "GET", "http://service-b:8080", nil)
    if err != nil {
        log.Error().Err(err).Msg("Failed to create request")
        return
    }

    resp, err := client.Do(req)
    if err != nil {
        log.Error().Err(err).Msg("Failed to call Service B")
        return
    }
    defer resp.Body.Close()

    log.Info().Str("status", resp.Status).Msg("Service B response status")
}