package auditv1

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/cleanup"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

const (
	// The largest page a single query will return, so that one request cannot
	// ask the engine to materialise an entire trail.
	maxQueryLimit = 1000
)

// Controller handles HTTP requests for reading the audit trail.
//
// Read and clean up only. There is no endpoint that writes or amends a record.
// Records are made by the middleware that observes the actions, and a trail the
// callers being audited can write to is not one.
//
// The engine authenticates but does not authorise, so these endpoints are open to
// any principal it accepts, and the trail names every other principal along with
// their source addresses. Of everything the engine serves, this is the surface the
// authorisation layer above it most needs to gate.
type Controller struct {
	auditRecordRetentionPeriod time.Duration
	cleanupRunner              *cleanup.Runner
	auditStore                 manage.Audit
	cleanupOperationsStore     manage.CleanupOperations
	logger                     core.Logger
}

// NewController creates a new audit Controller
// instance with the provided dependencies.
func NewController(
	auditRecordRetentionPeriod time.Duration,
	deps *typesv1.Dependencies,
) *Controller {
	return &Controller{
		auditRecordRetentionPeriod: auditRecordRetentionPeriod,
		cleanupRunner:              deps.CleanupRunner,
		auditStore:                 deps.AuditStore,
		cleanupOperationsStore:     deps.CleanupOperationsStore,
		logger:                     deps.Logger,
	}
}

// QueryAuditRecordsResponse holds a page of the audit trail.
type QueryAuditRecordsResponse struct {
	// Records are the matching records, oldest first.
	Records []*manage.AuditRecord `json:"records"`
	// HasMore says whether the trail holds further records matching the query beyond
	// this page.
	//
	// Reported rather than left to be inferred. A page of a trail is easily mistaken
	// for the whole answer to "what did this principal do", and a partial trail read
	// as a complete one is the one failure this endpoint must not have.
	HasMore bool `json:"hasMore"`
	// NextCursor resumes the listing immediately after this page, and is absent when
	// nothing follows it.
	//
	// Supplied ready to use rather than left for a caller to assemble out of the last
	// record, so that how this trail is paged is something the response states.
	NextCursor *AuditRecordCursor `json:"nextCursor,omitempty"`
}

// AuditRecordCursor resumes an audit trail listing after a record.
type AuditRecordCursor struct {
	AfterTimestamp int64  `json:"afterTimestamp"`
	AfterID        string `json:"afterId"`
}

// QueryAuditRecordsHandler is the handler for the
// GET /audit/records endpoint that reads the audit trail, narrowed by the
// `from`, `to`, `afterTimestamp`, `afterId`, `action`, `principalSubject`,
// `outcome` and `limit` query parameters.
//
// This is what makes the trail extractable without reaching into the state
// backend.
func (c *Controller) QueryAuditRecordsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	query, err := queryFromRequest(r)
	if err != nil {
		httputils.HTTPError(w, http.StatusBadRequest, err.Error())
		return
	}

	// One more than the page is asked for, so that whether anything follows it is known
	// without counting the matching set. A count would mean reading the whole trail on
	// the backends that cannot answer it from a listing, and would be out of date by the
	// time it was returned on a trail that is still being appended to.
	pageSize := query.Limit
	query.Limit = pageSize + 1

	records, err := c.auditStore.Query(r.Context(), query)
	if err != nil {
		c.requestLogger(r).Debug(
			"failed to query audit records",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	httputils.HTTPJSONResponse(w, http.StatusOK, pageOf(records, pageSize))
}

// GetAuditRecordHandler is the handler for the
// GET /audit/records/{id} endpoint that retrieves a single audit record.
func (c *Controller) GetAuditRecordHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	recordID := mux.Vars(r)["id"]

	record, err := c.auditStore.Get(r.Context(), recordID)
	if err != nil {
		notFoundErr := &manage.AuditRecordNotFound{}
		if errors.As(err, &notFoundErr) {
			httputils.HTTPError(w, http.StatusNotFound, notFoundErr.Error())
			return
		}
		c.requestLogger(r).Debug(
			"failed to get audit record",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	httputils.HTTPJSONResponse(w, http.StatusOK, record)
}

// CleanupAuditRecordsHandler is the handler for the
// POST /audit/records/cleanup endpoint that cleans up
// audit records that are older than the configured
// retention period.
func (c *Controller) CleanupAuditRecordsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	helpersv1.StartCleanup(
		w,
		r,
		c.cleanupRunner,
		cleanup.AuditRecordsTask(c.auditStore, c.auditRecordRetentionPeriod),
		c.requestLogger(r),
	)
}

// GetCleanupStatusHandler is the handler for the
// GET /audit/records/cleanup/{id} endpoint that retrieves the
// status of a cleanup operation.
func (c *Controller) GetCleanupStatusHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	operationID := mux.Vars(r)["id"]

	operation, err := c.cleanupOperationsStore.Get(r.Context(), operationID)
	if err != nil {
		notFoundErr := &manage.CleanupOperationNotFound{}
		if errors.As(err, &notFoundErr) {
			httputils.HTTPError(w, http.StatusNotFound, notFoundErr.Error())
			return
		}
		c.requestLogger(r).Debug(
			"failed to get cleanup operation",
			core.ErrorLogField("error", err),
		)
		httputils.HTTPError(w, http.StatusInternalServerError, utils.UnexpectedErrorMessage)
		return
	}

	httputils.HTTPJSONResponse(w, http.StatusOK, operation)
}

// Trims the extra record that was asked for, and says what follows the page.
func pageOf(records []*manage.AuditRecord, pageSize int) *QueryAuditRecordsResponse {
	hasMore := len(records) > pageSize
	if hasMore {
		records = records[:pageSize]
	}

	response := &QueryAuditRecordsResponse{Records: records, HasMore: hasMore}
	if hasMore {
		last := records[len(records)-1]
		response.NextCursor = &AuditRecordCursor{
			AfterTimestamp: last.Timestamp,
			AfterID:        last.ID,
		}
	}

	return response
}

// A malformed filter is rejected rather than ignored: a query that silently
// drops a bound would answer a different question from the one asked, and the
// caller would read the result as the whole trail for that window.
func queryFromRequest(r *http.Request) (*manage.AuditRecordQuery, error) {
	params := r.URL.Query()

	from, err := timestampParam(params.Get("from"), "from")
	if err != nil {
		return nil, err
	}

	to, err := timestampParam(params.Get("to"), "to")
	if err != nil {
		return nil, err
	}

	afterTimestamp, err := timestampParam(params.Get("afterTimestamp"), "afterTimestamp")
	if err != nil {
		return nil, err
	}

	limit, err := limitParam(params.Get("limit"))
	if err != nil {
		return nil, err
	}

	return &manage.AuditRecordQuery{
		From:             from,
		To:               to,
		AfterTimestamp:   afterTimestamp,
		AfterID:          params.Get("afterId"),
		Action:           params.Get("action"),
		PrincipalSubject: params.Get("principalSubject"),
		Outcome:          params.Get("outcome"),
		Limit:            limit,
	}, nil
}

func timestampParam(value string, name string) (int64, error) {
	if value == "" {
		return 0, nil
	}

	timestamp, err := strconv.ParseInt(value, 10, 64)
	if err != nil || timestamp < 0 {
		return 0, errors.New(
			"the \"" + name + "\" query parameter must be a unix timestamp in seconds",
		)
	}

	return timestamp, nil
}

func limitParam(value string) (int, error) {
	if value == "" {
		return manage.DefaultAuditQueryLimit, nil
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 {
		return 0, errors.New("the \"limit\" query parameter must be a positive integer")
	}

	if limit > maxQueryLimit {
		return 0, errors.New(
			"the \"limit\" query parameter must not exceed " + strconv.Itoa(maxQueryLimit),
		)
	}

	return limit, nil
}

// The logger for the request being answered, carrying its correlation ID, source
// address and who it was authenticated for.
//
// Falls back to the process logger when the request-context middleware has not run, so
// that a handler never has to decide whether logging is worth attempting.
func (c *Controller) requestLogger(r *http.Request) core.Logger {
	return httpmw.LoggerFromContext(r.Context(), c.logger)
}
