package manage

import (
	"context"
	"time"
)

// Audit provides an interface for a service that persists audit records,
// which say who did what, to what, and whether it worked.
//
// Audit records are append-only. there is no update or remove, only a retention-driven
// Cleanup. An audit trail that the acting principal can amend is not an audit trail.
type Audit interface {
	// Get retrieves an audit record by ID.
	// It returns an `AuditRecordNotFound` error if the record does not exist.
	Get(ctx context.Context, id string) (*AuditRecord, error)

	// Save persists a new audit record.
	//
	// A record ID is stored at most once, so the trail never holds the same ID
	// twice. Which copy survives a reused ID is not specified, callers must supply a
	// unique ID per record.
	Save(ctx context.Context, record *AuditRecord) error

	// Query retrieves the audit records matching the given query, ordered
	// oldest-first by timestamp and then by ID.
	//
	// The ordering is what makes extraction resumable, a forwarder passes the
	// timestamp and ID of the last record it shipped as the query's AfterTimestamp
	// and AfterID, and receives the records that follow it. Resuming on the
	// timestamp alone would not be enough, since several records can share one and the
	// limit is applied after ordering, so a second holding more records than the
	// limit would never be read past.
	Query(ctx context.Context, query *AuditRecordQuery) ([]*AuditRecord, error)

	// Cleanup removes audit records older than the given threshold date.
	// It returns the number of records removed.
	Cleanup(ctx context.Context, thresholdDate time.Time) (int64, error)
}

// AuditRecordQuery narrows an audit trail query.
// A zero value matches every record up to the backend's default limit.
//
// AfterTimestamp and AfterID work as a pair: a timestamp alone does not identify a
// record, so resuming a listing needs both.
type AuditRecordQuery struct {
	// From bounds the query to records at or after this unix timestamp in seconds.
	// Inclusive. Zero leaves the query unbounded at this end.
	From int64
	// To bounds the query to records before this unix timestamp in seconds.
	// Exclusive. Zero leaves the query unbounded at this end.
	To int64
	// AfterTimestamp is the timestamp half of a resume cursor, which takes the
	// listing to the records following one already read. Zero ignores the cursor
	// entirely, whatever AfterID holds, and starts the listing at From.
	AfterTimestamp int64
	// AfterID is the ID half of a resume cursor. Records sharing AfterTimestamp are
	// included when their ID is greater than this; records after AfterTimestamp
	// always are.
	AfterID string
	// Action restricts the query to a single action name, as "instance.destroy".
	Action string
	// PrincipalSubject restricts the query to the actions of one principal.
	PrincipalSubject string
	// Outcome restricts the query to `OutcomeSuccess` or `OutcomeFailure`.
	Outcome string
	// Limit caps the number of records returned. A non-positive limit applies
	// the backend's default, so an unbounded query cannot be requested by
	// accident.
	Limit int
}

// AuditRecord says who did what, to what, and whether it worked.
//
// The engine emits these; policy and reporting belong to the layer above it. What the
// engine is uniquely able to say is the attribution.
//
// The principal fields are flattened rather than nested so that a record is one level
// deep and greppable, and so that the subject cannot be confused for the credential it
// was derived from.
type AuditRecord struct {
	// ID uniquely identifies the record. It is assigned when the record is
	// emitted rather than when it is stored, so that the same identifier
	// appears in every sink the record reaches.
	ID string `json:"id"`
	// Timestamp is the unix timestamp in seconds when the action was taken.
	Timestamp int64 `json:"timestamp"`
	// Action names what was done, as "instance.destroy" or "changeset.create".
	Action string `json:"action"`
	// Target identifies what the action was carried out on, where the action names one.
	Target string `json:"target,omitempty"`
	// Outcome says whether the action succeeded.
	Outcome string `json:"outcome"`
	// StatusCode is the HTTP status the request was answered with, which distinguishes a
	// rejection from a failure that the outcome alone does not.
	StatusCode int `json:"statusCode"`
	// Method is the HTTP method of the request as received, so that a record of an
	// action the engine has no name for is still usable.
	Method string `json:"method"`
	// Path is the path of the request as received. The query string is excluded.
	Path string `json:"path"`
	// PrincipalMethod is how the principal authenticated, as "oidc", "sigv1" or
	// "apikey".
	PrincipalMethod string `json:"principalMethod"`
	// PrincipalSubject identifies who acted: a fingerprint for an API key, the
	// subject claim for a JWT, the key ID for a signed request. Never the
	// credential itself.
	PrincipalSubject string `json:"principalSubject"`
	// PrincipalName is a human-readable name for the principal, where the
	// authentication method supplies one.
	PrincipalName string `json:"principalName,omitempty"`
	// OnBehalfOf is the end user a trusted fronting service asserted, once identity
	// propagation populates it.
	OnBehalfOf string `json:"onBehalfOf,omitempty"`
	// SourceIP is where the request came from, preferring what a fronting proxy
	// reported. Validated as an address before being recorded, and only as
	// trustworthy as the deployment's front door.
	SourceIP string `json:"sourceIp,omitempty"`
	// UserAgent is the user agent the request was made with.
	UserAgent string `json:"userAgent,omitempty"`
	// RequestID correlates the record with the request, taken from the caller's header
	// where one was supplied so that a trail can be followed across components.
	RequestID string `json:"requestId,omitempty"`
	// TraceID identifies the distributed trace the action was carried out under, so that
	// an audited action leads to the trace of the work it did. Empty where tracing is
	// off, or where the trace was not one a backend recorded.
	TraceID string `json:"traceId,omitempty"`
	// SpanID identifies the span within that trace, which is the unit of work the
	// action corresponds to.
	SpanID string `json:"spanId,omitempty"`
}

// GetID returns the ID of the audit record.
func (r *AuditRecord) GetID() string {
	return r.ID
}

// GetCreated returns the time the action was taken, as a Unix timestamp in seconds.
func (r *AuditRecord) GetCreated() int64 {
	return r.Timestamp
}

// Outcomes a record can report.
const (
	// OutcomeSuccess is recorded for a request the engine answered with a 2xx status.
	OutcomeSuccess = "success"
	// OutcomeFailure is recorded for anything else, including a request refused before it
	// changed anything. The status code distinguishes the two.
	OutcomeFailure = "failure"
)

// DefaultAuditQueryLimit caps a query that does not ask for a limit of its own.
const DefaultAuditQueryLimit = 100
