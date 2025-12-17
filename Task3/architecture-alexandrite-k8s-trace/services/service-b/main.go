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

var tracer = otel.Tracer("service-b")

func main() {
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
        "service-b",
        otelhttp.WithTracerProvider(tp),
    )

    http.Handle("/", handler)

    log.Info().Msg("Service B starting on :8080")
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
            semconv.ServiceName("service-b"),
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
    // Используем _ вместо ctx, если он не нужен
    _, span := tracer.Start(r.Context(), "service-b-handler")
    defer span.End()

    log.Info().Msg("Service B: Received request")
    time.Sleep(30 * time.Millisecond) // Имитация работы

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status": "ok", "service": "b", "message": "Hello from Service B!"}`))

    log.Info().Msg("Service B: Response sent")
}