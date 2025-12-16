package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

var tracer = otel.Tracer("service-b")

func main() {
	ctx := context.Background()

	// Инициализация трассировки
	tp, err := initTracer(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// HTTP сервер с OpenTelemetry инструментацией
	handler := otelhttp.NewHandler(
		http.HandlerFunc(handleRequest),
		"service-b",
		otelhttp.WithTracerProvider(tp),
	)

	http.Handle("/", handler)

	log.Println("Service B starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	// Jaeger через OTLP gRPC
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint("simplest-collector:4317"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Ресурс с атрибутами сервиса
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("service-b"),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironment("development"),
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
	ctx, span := tracer.Start(r.Context(), "service-b-handler")
	defer span.End()

	log.Println("Service B: Received request")

	// Имитация обработки в сервисе B
	time.Sleep(100 * time.Millisecond)

	// Дополнительная операция в сервисе B
	processData(ctx)

	// Ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "service": "b", "message": "Hello from Service B!"}`))
}

func processData(ctx context.Context) {
	ctx, span := tracer.Start(ctx, "process-data")
	defer span.End()

	log.Println("Service B: Processing data...")
	time.Sleep(30 * time.Millisecond)
}