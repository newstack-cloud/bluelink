package eventsv1

import (
	"context"
	"net/http"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httputils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

const (
	// An internal timeout used for the cleanup process
	// that cleans up old events.
	// 10 minutes is a reasonable time to wait for the cleanup process
	// to complete for instances of the deploy engine with a lot of use.
	eventsCleanupTimeout = 10 * time.Minute
)

// Controller handles HTTP requests
// for managing events.
type Controller struct {
	eventsRetentionPeriod time.Duration
	eventStore            manage.Events
	clock                 commoncore.Clock
	logger                core.Logger
}

// NewController creates a new events Controller
// instance with the provided dependencies.
func NewController(
	eventsRetentionPeriod time.Duration,
	deps *typesv1.Dependencies,
) *Controller {
	return &Controller{
		eventsRetentionPeriod: eventsRetentionPeriod,
		eventStore:            deps.EventStore,
		clock:                 deps.Clock,
		logger:                deps.Logger,
	}
}

// CleanupEventsHandler is the handler for the
// POST /events/cleanup endpoint that cleans up
// events that are older than the configured
// retention period.
func (c *Controller) CleanupEventsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Carry out the cleanup process in a separate goroutine
	// to avoid blocking the request,
	// general clean up should be a task that a client can trigger
	// but not need to wait for.
	go c.cleanupEvents()

	httputils.HTTPJSONResponse(
		w,
		http.StatusAccepted,
		&helpersv1.MessageResponse{
			Message: "Cleanup started",
		},
	)
}

func (c *Controller) cleanupEvents() {
	logger := c.logger.Named("eventsCleanup")

	cleanupBefore := c.clock.Now().Add(
		-c.eventsRetentionPeriod,
	)

	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		eventsCleanupTimeout,
	)
	defer cancel()

	err := c.eventStore.Cleanup(
		ctxWithTimeout,
		cleanupBefore,
	)
	if err != nil {
		logger.Error(
			"failed to clean up old events",
			core.ErrorLogField("error", err),
		)
		return
	}
}
