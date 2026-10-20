package audit

import "github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"

// Record says who did what, to what, and whether it worked.
//
// Aliased to the state entity rather than converted into it. A record that travels
// through a sink and a record that is stored are the same record, and a conversion
// between two copies of the same field list is somewhere a field can be dropped in
// silence.
type Record = manage.AuditRecord

// Outcomes a record can report.
const (
	OutcomeSuccess = manage.OutcomeSuccess
	OutcomeFailure = manage.OutcomeFailure
)
