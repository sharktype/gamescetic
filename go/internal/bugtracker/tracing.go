package bugtracker

// sentry-go removed its native OpenTelemetry span processor in v0.47 in favour of
// sentryotlp, which only speaks Sentry's OTLP endpoint. GlitchTip implements the
// envelope API but not OTLP traces, so this is a local port of the removed
// processor: OTel spans are mirrored into Sentry transactions sent over the
// envelope transport GlitchTip already accepts.

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	otelTrace "go.opentelemetry.io/otel/trace"
)

const (
	httpMethodKey           = "http.request.method"
	httpMethodLegacyKey     = "http.method"
	httpStatusCodeKey       = "http.response.status_code"
	httpStatusCodeLegacyKey = "http.status_code"
	httpRouteKey            = "http.route"
	httpTargetKey           = "http.target"
	httpURLKey              = "http.url"
	urlPathKey              = "url.path"
	urlFullKey              = "url.full"
	dbSystemKey             = "db.system"
	dbStatementKey          = "db.statement"
	rpcSystemKey            = "rpc.system"
	rpcGRPCStatusCodeKey    = "rpc.grpc.status_code"
	messagingSystemKey      = "messaging.system"
	faasTriggerKey          = "faas.trigger"
)

type sentrySpanProcessor struct {
	spans *spanStore
}

func newSentrySpanProcessor() sdktrace.SpanProcessor {
	return &sentrySpanProcessor{spans: newSpanStore()}
}

func (p *sentrySpanProcessor) OnStart(parent context.Context, span sdktrace.ReadWriteSpan) {
	spanContext := span.SpanContext()
	if !spanContext.IsValid() || !spanContext.IsSampled() {
		return
	}

	traceID := spanContext.TraceID()
	spanID := spanContext.SpanID()
	parentSpanID := span.Parent().SpanID()

	if _, ok := p.spans.addChild(traceID, spanID, parentSpanID, span.Name(), span.StartTime()); ok {
		return
	}

	// The OTel sampler has already made the decision, so make Sentry honour it rather than
	// sampling a second time and producing errors that point at dropped transactions.
	transaction := sentry.StartTransaction(
		parent, span.Name(), sentry.WithSpanSampled(sentry.SampledTrue),
	)
	transaction.SpanID = sentry.SpanID(spanID)
	transaction.TraceID = sentry.TraceID(traceID)
	transaction.ParentSpanID = sentry.SpanID(parentSpanID)
	transaction.StartTime = span.StartTime()

	p.spans.set(traceID, spanID, transaction)
}

func (p *sentrySpanProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	spanContext := span.SpanContext()
	if !spanContext.IsValid() {
		return
	}

	traceID := spanContext.TraceID()
	spanID := spanContext.SpanID()

	sentrySpan, ok := p.spans.get(traceID, spanID)
	if !ok {
		return
	}

	defer p.spans.markFinished(traceID, spanID)

	// Never let an outgoing request to Sentry itself overwrite a real transaction.
	if isSentryRequestSpan(sentrySpan.Context(), span) {
		return
	}

	if sentrySpan.IsTransaction() {
		updateTransaction(sentrySpan, span)
	} else {
		updateSpan(sentrySpan, span)
	}

	sentrySpan.EndTime = span.EndTime()
	sentrySpan.Finish()
}

func (p *sentrySpanProcessor) Shutdown(ctx context.Context) error {
	p.spans.clear()

	return p.ForceFlush(ctx)
}

func (p *sentrySpanProcessor) ForceFlush(ctx context.Context) error {
	sentry.FlushWithContext(ctx)

	return nil
}

type spanStore struct {
	mu     sync.Mutex
	traces map[otelTrace.TraceID]*traceSpans
}

type traceSpans struct {
	root         *sentry.Span
	spans        map[otelTrace.SpanID]*sentry.Span
	knownSpanIDs map[otelTrace.SpanID]struct{}
}

func newSpanStore() *spanStore {
	return &spanStore{traces: make(map[otelTrace.TraceID]*traceSpans)}
}

func (s *spanStore) set(traceID otelTrace.TraceID, spanID otelTrace.SpanID, span *sentry.Span) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.traces[traceID]
	if !ok {
		entry = &traceSpans{
			root:         span,
			spans:        make(map[otelTrace.SpanID]*sentry.Span),
			knownSpanIDs: make(map[otelTrace.SpanID]struct{}),
		}
		s.traces[traceID] = entry
	}

	entry.spans[spanID] = span
	entry.knownSpanIDs[spanID] = struct{}{}
}

func (s *spanStore) addChild(
	traceID otelTrace.TraceID,
	spanID otelTrace.SpanID,
	parentSpanID otelTrace.SpanID,
	name string,
	startTime time.Time,
) (*sentry.Span, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.traces[traceID]
	if !ok {
		return nil, false
	}

	if _, known := entry.knownSpanIDs[parentSpanID]; !known {
		return nil, false
	}

	span := entry.root.StartChild(name)
	span.SpanID = sentry.SpanID(spanID)
	span.ParentSpanID = sentry.SpanID(parentSpanID)
	span.StartTime = startTime

	entry.spans[spanID] = span
	entry.knownSpanIDs[spanID] = struct{}{}

	return span, true
}

func (s *spanStore) get(traceID otelTrace.TraceID, spanID otelTrace.SpanID) (*sentry.Span, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.traces[traceID]
	if !ok {
		return nil, false
	}

	span, ok := entry.spans[spanID]

	return span, ok
}

func (s *spanStore) markFinished(traceID otelTrace.TraceID, spanID otelTrace.SpanID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.traces[traceID]
	if !ok {
		return
	}

	delete(entry.spans, spanID)

	if len(entry.spans) == 0 {
		delete(s.traces, traceID)
	}
}

func (s *spanStore) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	clear(s.traces)
}

func updateTransaction(transaction *sentry.Span, span sdktrace.ReadOnlySpan) {
	attributes := spanAttributes(span)
	parsed := parseSpanAttributes(span, attributes)

	transaction.SetContext("otel", map[string]any{
		"attributes": attributeValues(span.Attributes()),
		"resource":   attributeValues(span.Resource().Attributes()),
	})
	transaction.Status = mapStatus(span, attributes)
	transaction.Name = parsed.description
	transaction.Op = parsed.op
	transaction.Source = parsed.source
}

func updateSpan(span *sentry.Span, otelSpan sdktrace.ReadOnlySpan) {
	attributes := spanAttributes(otelSpan)
	parsed := parseSpanAttributes(otelSpan, attributes)

	span.Status = mapStatus(otelSpan, attributes)
	span.Op = parsed.op
	span.Description = parsed.description
	span.SetData("otel.kind", otelSpan.SpanKind().String())

	for _, attr := range otelSpan.Attributes() {
		span.SetData(string(attr.Key), attr.Value.AsInterface())
	}
}

type parsedSpanAttributes struct {
	op          string
	description string
	source      sentry.TransactionSource
}

func parseSpanAttributes(
	span sdktrace.ReadOnlySpan, attributes map[string]attribute.Value,
) parsedSpanAttributes {
	if _, ok := stringAttribute(attributes, httpMethodKey, httpMethodLegacyKey); ok {
		return httpSpanAttributes(span, attributes)
	}

	if _, ok := attributes[dbSystemKey]; ok {
		description, _ := stringAttribute(attributes, dbStatementKey)

		if description == "" {
			description = span.Name()
		}

		return parsedSpanAttributes{
			op:          "db",
			description: description,
			source:      sentry.SourceTask,
		}
	}

	if _, ok := attributes[rpcSystemKey]; ok {
		return parsedSpanAttributes{
			op:          "rpc",
			description: span.Name(),
			source:      sentry.SourceRoute,
		}
	}

	if _, ok := attributes[messagingSystemKey]; ok {
		return parsedSpanAttributes{
			op:          "messaging",
			description: span.Name(),
			source:      sentry.SourceRoute,
		}
	}

	if trigger, ok := attributes[faasTriggerKey]; ok {
		return parsedSpanAttributes{
			op:          trigger.AsString(),
			description: span.Name(),
			source:      sentry.SourceRoute,
		}
	}

	return parsedSpanAttributes{
		description: span.Name(),
		source:      sentry.SourceCustom,
	}
}

func httpSpanAttributes(
	span sdktrace.ReadOnlySpan, attributes map[string]attribute.Value,
) parsedSpanAttributes {
	op := "http"

	switch span.SpanKind() {
	case otelTrace.SpanKindClient:
		op = "http.client"
	case otelTrace.SpanKindServer:
		op = "http.server"
	}

	method, _ := stringAttribute(attributes, httpMethodKey, httpMethodLegacyKey)
	route, _ := stringAttribute(attributes, httpRouteKey)
	target, _ := stringAttribute(attributes, httpTargetKey)
	path, _ := stringAttribute(attributes, urlPathKey)
	fullURL, _ := stringAttribute(attributes, urlFullKey)

	switch {
	case route != "":
		path = route
	case target != "":
		path = urlPath(target)
	case path == "" && fullURL != "":
		if parsed, err := url.Parse(fullURL); err == nil {
			path = fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, parsed.Path)
		}
	}

	if path == "" {
		return parsedSpanAttributes{
			op:          op,
			description: span.Name(),
			source:      sentry.SourceCustom,
		}
	}

	source := sentry.SourceURL
	if route != "" || path == "/" {
		source = sentry.SourceRoute
	}

	return parsedSpanAttributes{
		op:          op,
		description: strings.TrimSpace(method + " " + path),
		source:      source,
	}
}

func spanAttributes(span sdktrace.ReadOnlySpan) map[string]attribute.Value {
	attributes := make(map[string]attribute.Value, len(span.Attributes()))
	for _, attr := range span.Attributes() {
		attributes[string(attr.Key)] = attr.Value
	}

	return attributes
}

func attributeValues(attributes []attribute.KeyValue) map[string]any {
	values := make(map[string]any, len(attributes))
	for _, attr := range attributes {
		values[string(attr.Key)] = attr.Value.AsInterface()
	}

	return values
}

func stringAttribute(attributes map[string]attribute.Value, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := attributes[key]; ok {
			return value.AsString(), true
		}
	}

	return "", false
}

func urlPath(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	return parsed.Path
}

var canonicalHTTPStatuses = map[string]sentry.SpanStatus{
	"400": sentry.SpanStatusFailedPrecondition,
	"401": sentry.SpanStatusUnauthenticated,
	"403": sentry.SpanStatusPermissionDenied,
	"404": sentry.SpanStatusNotFound,
	"409": sentry.SpanStatusAborted,
	"429": sentry.SpanStatusResourceExhausted,
	"499": sentry.SpanStatusCanceled,
	"500": sentry.SpanStatusInternalError,
	"501": sentry.SpanStatusUnimplemented,
	"503": sentry.SpanStatusUnavailable,
	"504": sentry.SpanStatusDeadlineExceeded,
}

var canonicalGRPCStatuses = map[string]sentry.SpanStatus{
	"1":  sentry.SpanStatusCanceled,
	"2":  sentry.SpanStatusUnknown,
	"3":  sentry.SpanStatusInvalidArgument,
	"4":  sentry.SpanStatusDeadlineExceeded,
	"5":  sentry.SpanStatusNotFound,
	"6":  sentry.SpanStatusAlreadyExists,
	"7":  sentry.SpanStatusPermissionDenied,
	"8":  sentry.SpanStatusResourceExhausted,
	"9":  sentry.SpanStatusFailedPrecondition,
	"10": sentry.SpanStatusAborted,
	"11": sentry.SpanStatusOutOfRange,
	"12": sentry.SpanStatusUnimplemented,
	"13": sentry.SpanStatusInternalError,
	"14": sentry.SpanStatusUnavailable,
	"15": sentry.SpanStatusDataLoss,
	"16": sentry.SpanStatusUnauthenticated,
}

func mapStatus(span sdktrace.ReadOnlySpan, attributes map[string]attribute.Value) sentry.SpanStatus {
	if code, ok := stringAttribute(attributes, httpStatusCodeKey, httpStatusCodeLegacyKey); ok {
		if status, found := canonicalHTTPStatuses[code]; found {
			return status
		}
	}

	if code, ok := attributes[rpcGRPCStatusCodeKey]; ok {
		if status, found := canonicalGRPCStatuses[code.Emit()]; found {
			return status
		}
	}

	switch span.Status().Code {
	case codes.Unset, codes.Ok:
		return sentry.SpanStatusOK
	case codes.Error:
		return sentry.SpanStatusInternalError
	default:
		return sentry.SpanStatusUnknown
	}
}

func isSentryRequestSpan(ctx context.Context, span sdktrace.ReadOnlySpan) bool {
	for _, attr := range span.Attributes() {
		if attr.Key == httpURLKey || attr.Key == urlFullKey {
			return isSentryURL(ctx, attr.Value.AsString())
		}
	}

	return false
}

func isSentryURL(ctx context.Context, rawURL string) bool {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	client := hub.Client()
	if client == nil {
		return false
	}

	dsn, err := sentry.NewDsn(client.Options().Dsn)
	if err != nil {
		return false
	}

	return strings.Contains(rawURL, dsn.GetAPIURL().String())
}
