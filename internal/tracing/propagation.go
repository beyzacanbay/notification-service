package tracing

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
)

const traceKeyPrefix = "trace:"
const traceTTL = 1 * time.Hour

// InjectTraceContext stores the current span's trace context in Redis,
// keyed by notification ID. Worker reads this to continue the same trace.
func InjectTraceContext(ctx context.Context, redisClient *redis.Client, notificationID string) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}

	// Store as "traceID:spanID" so worker can create a linked span
	sc := span.SpanContext()
	val := fmt.Sprintf("%s:%s", sc.TraceID().String(), sc.SpanID().String())
	redisClient.Set(ctx, traceKeyPrefix+notificationID, val, traceTTL)
}

// ExtractTraceContext reads trace context from Redis and returns a context
// with a remote span context, so worker spans join the same trace.
func ExtractTraceContext(ctx context.Context, redisClient *redis.Client, notificationID string) context.Context {
	val, err := redisClient.Get(ctx, traceKeyPrefix+notificationID).Result()
	if err != nil || val == "" {
		return ctx
	}

	// Parse "traceID:spanID"
	var traceIDStr, spanIDStr string
	if n, _ := fmt.Sscanf(val, "%32s:%16s", &traceIDStr, &spanIDStr); n != 2 {
		// Try splitting by colon
		parts := splitOnce(val, ':')
		if len(parts) != 2 {
			return ctx
		}
		traceIDStr = parts[0]
		spanIDStr = parts[1]
	}

	traceID, err := trace.TraceIDFromHex(traceIDStr)
	if err != nil {
		return ctx
	}
	spanID, err := trace.SpanIDFromHex(spanIDStr)
	if err != nil {
		return ctx
	}

	// Create remote span context so new spans become children of the API span
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})

	// Clean up — one-time use
	redisClient.Del(ctx, traceKeyPrefix+notificationID)

	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

func splitOnce(s string, sep byte) []string {
	for i := range s {
		if s[i] == sep {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
