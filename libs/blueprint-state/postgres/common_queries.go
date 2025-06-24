package postgres

import (
	"fmt"
	"strings"

	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

func removeMultipleQuery(table string, idParamNames []string) string {
	finalParamNames := commoncore.Map(
		idParamNames,
		func(paramName string, _ int) string {
			return fmt.Sprintf("@%s", paramName)
		},
	)
	return fmt.Sprintf(`
	DELETE FROM %s
	WHERE id IN (%s)
	`,
		table,
		strings.Join(finalParamNames, ", "),
	)
}
