package enginev1

import (
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/audit"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// Opens the sink the audit trail is written to, and reports how to close it.
func setupAuditSink(
	config *core.AuditConfig,
	auditStore manage.Audit,
	logger bpcore.Logger,
) (audit.Sink, func(), error) {
	if !config.Enabled {
		logger.Warn(
			"no audit trail is being recorded, so there will be no record of who carried " +
				"out the actions taken through this engine; set audit.enabled to true to " +
				"record one",
		)

		return &audit.NopSink{}, func() {}, nil
	}

	sinks := []audit.Sink{audit.NewStateSink(auditStore)}
	if config.Stdout {
		sinks = append(sinks, audit.NewStdoutSink())
	}

	logger.Info(
		"recording an audit trail of the actions taken through this engine",
		bpcore.BoolLogField("toProcessOutput", config.Stdout),
	)

	sink := audit.NewMultiSink(sinks...)

	return sink, func() {
		if err := sink.Close(); err != nil {
			logger.Error(
				"failed to close the audit sink cleanly",
				bpcore.ErrorLogField("error", err),
			)
		}
	}, nil
}
