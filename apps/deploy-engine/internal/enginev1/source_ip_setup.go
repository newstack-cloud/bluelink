package enginev1

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/httpmw"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// Builds the middleware that decides what address a request is attributed to.
//
// A source the middleware does not implement is refused here rather than at the first
// request. It would otherwise resolve no address at all, and an audit trail that quietly
// stopped saying where actions came from can go unnoticed.
func setupSourceIPMiddleware(
	config *core.Config,
	logger bpcore.Logger,
) (*httpmw.ExtractIPMiddleware, error) {
	source := httpmw.IPExtractionSource(config.IPExtractionSource)
	if !httpmw.IsKnownSource(source) {
		return nil, fmt.Errorf(
			"%q is not a supported ip_extraction_source, expected one of %v",
			config.IPExtractionSource,
			httpmw.KnownSources(),
		)
	}

	if source == httpmw.DirectExtractionSource {
		logger.Info(
			"attributing requests to the address they connect from, ignoring forwarded " +
				"headers; set ip_extraction_source to match a proxy in front of this " +
				"engine for it to attribute them to the original caller",
		)
	} else {
		logger.Info(
			"attributing requests to the address reported by a known proxy",
			bpcore.StringLogField("ipExtractionSource", string(source)),
		)
	}

	return httpmw.NewExtractIPMiddleware(
		httpmw.WithExtractIPMiddlewareSource(source),
	), nil
}
