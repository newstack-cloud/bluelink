package validation

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// IsIPAddress validates that a string is a valid IPv4 or IPv6 address.
func IsIPAddress() func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringVal := core.StringValueFromScalar(value)
		ip := net.ParseIP(stringVal)
		if ip == nil {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must be a valid IPv4 or IPv6 address, but got %q.",
						fieldName,
						ip,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}

// IsIPv4Address validates that a string is a valid IPv4 address.
func IsIPv4Address() func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringVal := core.StringValueFromScalar(value)
		ip := net.ParseIP(stringVal)
		if ip == nil || ip.To4() == nil {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must be a valid IPv4 address, but got %q.",
						fieldName,
						stringVal,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}

// IsIPv6Address validates that a string is a valid IPv6 address.
func IsIPv6Address() func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringVal := core.StringValueFromScalar(value)
		ip := net.ParseIP(stringVal)
		if ip == nil || ip.To16() == nil {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must be a valid IPv6 address, but got %q.",
						fieldName,
						stringVal,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}

// IsIPv4Range validates that a string is a valid IPv4 address range.
func IsIPv4Range() func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringVal := core.StringValueFromScalar(value)
		ips := strings.Split(stringVal, "-")
		if len(ips) != 2 {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must be a valid IPv4 address range in the format 'start-end', but got %q.",
						fieldName,
						stringVal,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		startIP := net.ParseIP(strings.TrimSpace(ips[0]))
		endIP := net.ParseIP(strings.TrimSpace(ips[1]))
		if startIP == nil || endIP == nil || bytes.Compare(startIP, endIP) > 0 {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must be a valid IPv4 address range, but got %q.",
						fieldName,
						stringVal,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}

// IsMACAddress validates that a string is a valid MAC address.
func IsMACAddress() func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringVal := core.StringValueFromScalar(value)
		if _, err := net.ParseMAC(stringVal); err != nil {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must be a valid MAC address, but got %q: %v.",
						fieldName,
						stringVal,
						err,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}

// IsCIDR validates that a string is in valid CIDR notation.
func IsCIDR() func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringVal := core.StringValueFromScalar(value)
		if _, _, err := net.ParseCIDR(stringVal); err != nil {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must be in valid CIDR notation, but got %q: %v.",
						fieldName,
						stringVal,
						err,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}

// IsCIDRNetwork validates that a string is in valid CIDR notation and
// has significant bits between minValue and maxValue (inclusive).
func IsCIDRNetwork(
	minValue int,
	maxValue int,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringVal := core.StringValueFromScalar(value)
		_, ipNet, err := net.ParseCIDR(stringVal)
		if err != nil {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must be in valid CIDR notation, but got %q: %v.",
						fieldName,
						stringVal,
						err,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		if ipNet == nil || stringVal != ipNet.String() {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"expected %q to contain a valid network value, expected %s, got %s",
						fieldName,
						ipNet,
						stringVal,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		sigbits, _ := ipNet.Mask.Size()
		if sigbits < minValue || sigbits > maxValue {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must contain a network value with significant bits between %d and %d, but got %d.",
						fieldName,
						minValue,
						maxValue,
						sigbits,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}

// IsPortNumber validates that a string is a valid port number (0-65535).
func IsPortNumber() func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarInt(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"an integer",
			)
		}

		intValue := core.IntValueFromScalar(value)
		if intValue < 1 || intValue > 65535 {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must be a valid port number (1-65535), but got %d.",
						fieldName,
						intValue,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}

// IsPortNumberOrZero validates that a string is a valid port number (0-65535) or zero.
func IsPortNumberOrZero() func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarInt(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"an integer",
			)
		}

		intValue := core.IntValueFromScalar(value)
		if intValue < 0 || intValue > 65535 {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must be a valid port number (1-65535) or zero, but got %d.",
						fieldName,
						intValue,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}
