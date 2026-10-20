package enginev1

import (
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/auditv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/deploymentsv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/eventsv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/validationv1"
)

// What the rest of the wire-up needs from the registered routes.
type registeredHandlers struct {
	// The deployment controller, whose in-flight operations are drained at shutdown.
	deployments *deploymentsv1.Controller
	// The routes served without authenticating, which is the health check alone. It has
	// to answer for a load balancer that holds no credentials.
	unauthenticated []*mux.Route
}

// Registers every route the API serves.
//
// Before the middleware, which needs to be told the routes it must not authenticate.
func setupHandlers(
	router *mux.Router,
	dependencies *typesv1.Dependencies,
	config *core.Config,
) *registeredHandlers {
	healthHandler := setupHealthHandler(router)

	helpersv1.SetupRequestBodyValidator()

	setupValidationHandlers(router, dependencies, config)
	deploymentCtrl := setupDeploymentHandlers(router, dependencies, config)
	setupEventManagementHandlers(router, dependencies, config)
	setupAuditHandlers(router, dependencies, config)

	return &registeredHandlers{
		deployments:     deploymentCtrl,
		unauthenticated: []*mux.Route{healthHandler},
	}
}

func setupHealthHandler(
	router *mux.Router,
) *mux.Route {
	return router.HandleFunc("/health", HealthHandler).Methods("GET")
}

func setupValidationHandlers(
	router *mux.Router,
	dependencies *typesv1.Dependencies,
	config *core.Config,
) {
	retentionPeriod := time.Duration(
		config.Maintenance.BlueprintValidationRetentionPeriod,
	) * time.Second

	validationCtrl := validationv1.NewController(
		retentionPeriod,
		dependencies,
	)

	router.HandleFunc(
		"/validations",
		validationCtrl.CreateBlueprintValidationHandler,
	).Methods("POST")

	router.HandleFunc(
		"/validations/{id}",
		validationCtrl.GetBlueprintValidationHandler,
	).Methods("GET")

	router.HandleFunc(
		"/validations/{id}/stream",
		validationCtrl.StreamEventsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/validations/cleanup",
		validationCtrl.CleanupBlueprintValidationsHandler,
	).Methods("POST")

	router.HandleFunc(
		"/validations/cleanup/{id}",
		validationCtrl.GetCleanupStatusHandler,
	).Methods("GET")
}

func setupDeploymentHandlers(
	router *mux.Router,
	dependencies *typesv1.Dependencies,
	config *core.Config,
) *deploymentsv1.Controller {
	changesetRetentionPeriod := time.Duration(
		config.Maintenance.ChangesetRetentionPeriod,
	) * time.Second

	reconciliationResultsRetentionPeriod := time.Duration(
		config.Maintenance.ReconciliationResultsRetentionPeriod,
	) * time.Second

	deployTimeout := time.Duration(
		config.Blueprints.DeploymentTimeout,
	) * time.Second

	drainTimeout := config.GetDrainTimeout()

	deploymentCtrl := deploymentsv1.NewController(
		changesetRetentionPeriod,
		reconciliationResultsRetentionPeriod,
		deployTimeout,
		drainTimeout,
		dependencies,
	)

	setupChangesetHandlers(router, deploymentCtrl)
	setupInstanceHandlers(router, deploymentCtrl)
	setupReconciliationHandlers(router, deploymentCtrl)

	return deploymentCtrl
}

func setupChangesetHandlers(router *mux.Router, ctrl *deploymentsv1.Controller) {
	router.HandleFunc(
		"/deployments/changes",
		ctrl.CreateChangesetHandler,
	).Methods("POST")

	router.HandleFunc(
		"/deployments/changes/{id}/stream",
		ctrl.StreamChangesetEventsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/deployments/changes/{id}",
		ctrl.GetChangesetHandler,
	).Methods("GET")

	router.HandleFunc(
		"/deployments/changes/cleanup",
		ctrl.CleanupChangesetsHandler,
	).Methods("POST")

	router.HandleFunc(
		"/deployments/changes/cleanup/{id}",
		ctrl.GetChangesetsCleanupStatusHandler,
	).Methods("GET")
}

func setupInstanceHandlers(router *mux.Router, ctrl *deploymentsv1.Controller) {
	router.HandleFunc(
		"/deployments/instances",
		ctrl.ListBlueprintInstancesHandler,
	).Methods("GET")

	router.HandleFunc(
		"/deployments/instances",
		ctrl.CreateBlueprintInstanceHandler,
	).Methods("POST")

	router.HandleFunc(
		"/deployments/instances/{id}/stream",
		ctrl.StreamDeploymentEventsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/deployments/instances/{id}",
		ctrl.GetBlueprintInstanceHandler,
	).Methods("GET")

	router.HandleFunc(
		"/deployments/instances/{id}",
		ctrl.UpdateBlueprintInstanceHandler,
	).Methods("PATCH")

	router.HandleFunc(
		"/deployments/instances/{id}/exports",
		ctrl.GetBlueprintInstanceExportsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/deployments/instances/{id}/destroy",
		ctrl.DestroyBlueprintInstanceHandler,
	).Methods("POST")
}

func setupReconciliationHandlers(router *mux.Router, ctrl *deploymentsv1.Controller) {
	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/check",
		ctrl.CheckReconciliationHandler,
	).Methods("POST")

	router.HandleFunc(
		"/deployments/instances/{id}/reconciliation/apply",
		ctrl.ApplyReconciliationHandler,
	).Methods("POST")

	router.HandleFunc(
		"/deployments/reconciliation-results/cleanup",
		ctrl.CleanupReconciliationResultsHandler,
	).Methods("POST")

	router.HandleFunc(
		"/deployments/reconciliation-results/cleanup/{id}",
		ctrl.GetReconciliationResultsCleanupStatusHandler,
	).Methods("GET")
}

func setupAuditHandlers(
	router *mux.Router,
	dependencies *typesv1.Dependencies,
	config *core.Config,
) {
	retentionPeriod := time.Duration(
		config.Maintenance.AuditRecordRetentionPeriod,
	) * time.Second

	auditCtrl := auditv1.NewController(
		retentionPeriod,
		dependencies,
	)

	// Registered before the parameterised record route so that "cleanup" is not
	// taken for a record ID.
	router.HandleFunc(
		"/audit/records/cleanup",
		auditCtrl.CleanupAuditRecordsHandler,
	).Methods("POST")

	router.HandleFunc(
		"/audit/records/cleanup/{id}",
		auditCtrl.GetCleanupStatusHandler,
	).Methods("GET")

	router.HandleFunc(
		"/audit/records",
		auditCtrl.QueryAuditRecordsHandler,
	).Methods("GET")

	router.HandleFunc(
		"/audit/records/{id}",
		auditCtrl.GetAuditRecordHandler,
	).Methods("GET")
}

func setupEventManagementHandlers(
	router *mux.Router,
	dependencies *typesv1.Dependencies,
	config *core.Config,
) {
	retentionPeriod := time.Duration(
		config.Maintenance.EventsRetentionPeriod,
	) * time.Second

	eventsCtrl := eventsv1.NewController(
		retentionPeriod,
		dependencies,
	)

	router.HandleFunc(
		"/events/cleanup",
		eventsCtrl.CleanupEventsHandler,
	).Methods("POST")

	router.HandleFunc(
		"/events/cleanup/{id}",
		eventsCtrl.GetCleanupStatusHandler,
	).Methods("GET")
}
