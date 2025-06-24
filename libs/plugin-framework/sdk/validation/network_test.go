package validation

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type NetworkValidationSuite struct {
	suite.Suite
}

func (s *NetworkValidationSuite) Test_valid_ip_addresses() {
	validIPAddresses := []string{
		"192.168.1.1",
		"10.0.0.1",
		"172.16.0.1",
		"127.0.0.1",
		"2001:0db8:85a3:0000:0000:8a2e:0370:7334", // Valid IPv6
		"::1",                    // Loopback IPv6
		"2001:db8::ff00:42:8329", // Another valid IPv6
	}

	for _, ip := range validIPAddresses {
		diagnostics := IsIPAddress()("exampleField", core.ScalarFromString(ip))
		s.Assert().Empty(diagnostics)
	}
}

func (s *NetworkValidationSuite) Test_invalid_ip_addresses() {
	invalidIPAddresses := []string{
		"256.256.256.256", // Invalid octet
		"192.168.1",       // Too short
		"::::2022:2121",   // Invalid IPv6 format
	}

	for _, ip := range invalidIPAddresses {
		diagnostics := IsIPAddress()("exampleField", core.ScalarFromString(ip))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a valid IPv4 or IPv6 address")
	}
}

func (s *NetworkValidationSuite) Test_invalid_type_for_ip_address() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(1234),
		core.ScalarFromFloat(12.34),
		core.ScalarFromBool(true),
	}

	for _, value := range invalidValues {
		diagnostics := IsIPAddress()("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *NetworkValidationSuite) Test_valid_ipv4_addresses() {
	validIPv4Addresses := []string{
		"192.168.1.1",
		"10.0.0.1",
		"172.16.0.1",
		"127.0.0.1",
	}
	for _, ip := range validIPv4Addresses {
		diagnostics := IsIPv4Address()("exampleField", core.ScalarFromString(ip))
		s.Assert().Empty(diagnostics)
	}
}

func (s *NetworkValidationSuite) Test_invalid_ipv4_addresses() {
	invalidIPv4Addresses := []string{
		"256.256.256.256", // Invalid octet
		"192.168.1",       // Too short
		"::1",             // IPv6 address
	}

	for _, ip := range invalidIPv4Addresses {
		diagnostics := IsIPv4Address()("exampleField", core.ScalarFromString(ip))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a valid IPv4 address")
	}
}

func (s *NetworkValidationSuite) Test_invalid_type_for_ipv4_address() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(1234),
		core.ScalarFromFloat(12.34),
		core.ScalarFromBool(true),
	}

	for _, value := range invalidValues {
		diagnostics := IsIPv4Address()("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *NetworkValidationSuite) Test_valid_ipv6_addresses() {
	validIPv6Addresses := []string{
		"2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		"::1",
		"2001:db8::ff00:42:8329",
	}

	for _, ip := range validIPv6Addresses {
		diagnostics := IsIPv6Address()("exampleField", core.ScalarFromString(ip))
		s.Assert().Empty(diagnostics)
	}
}

func (s *NetworkValidationSuite) Test_invalid_ipv6_addresses() {
	invalidIPv6Addresses := []string{
		"255.255.255.256",              // Invalid Ipv4 format
		"2001:db8::ff00:42:8329::",     // Invalid format
		"2001:db8::ff00:42:8329:12345", // Too long
	}

	for _, ip := range invalidIPv6Addresses {
		diagnostics := IsIPv6Address()("exampleField", core.ScalarFromString(ip))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a valid IPv6 address")
	}
}

func (s *NetworkValidationSuite) Test_invalid_type_for_ipv6_address() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(1234),
		core.ScalarFromFloat(12.34),
		core.ScalarFromBool(true),
	}

	for _, value := range invalidValues {
		diagnostics := IsIPv6Address()("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *NetworkValidationSuite) Test_valid_ipv4_ranges() {
	validRanges := []string{
		"8.0.0.0-10.0.0.0",
		"127.0.0.1-127.0.0.255",
	}

	for _, ipRange := range validRanges {
		diagnostics := IsIPv4Range()("exampleField", core.ScalarFromString(ipRange))
		s.Assert().Empty(diagnostics)
	}
}

func (s *NetworkValidationSuite) Test_invalid_ipv4_ranges() {
	invalidRanges := []string{
		"256.0.0.0-256.0.0.255",   // Invalid octet
		"192.168.1.0-192.168.0.0", // Backwards range
	}

	for _, ipRange := range invalidRanges {
		diagnostics := IsIPv4Range()("exampleField", core.ScalarFromString(ipRange))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a valid IPv4 address range")
	}
}

func (s *NetworkValidationSuite) Test_invalid_type_for_ipv4_range() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(1234),
		core.ScalarFromFloat(12.34),
		core.ScalarFromBool(true),
	}

	for _, value := range invalidValues {
		diagnostics := IsIPv4Range()("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *NetworkValidationSuite) Test_valid_mac_addresses() {
	validMACAddresses := []string{
		"00:1A:2B:3C:4D:5E",
		"01:23:45:67:89:AB",
		"AA:BB:CC:DD:EE:FF",
	}

	for _, mac := range validMACAddresses {
		diagnostics := IsMACAddress()("exampleField", core.ScalarFromString(mac))
		s.Assert().Empty(diagnostics)
	}
}

func (s *NetworkValidationSuite) Test_invalid_mac_addresses() {
	invalidMACAddresses := []string{
		"00:1A:2B:3C:4D:5G",    // Invalid character
		"01:23:45:67:89",       // Too short
		"01:23:45:67:89:AB:CD", // Too long
	}

	for _, mac := range invalidMACAddresses {
		diagnostics := IsMACAddress()("exampleField", core.ScalarFromString(mac))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a valid MAC address")
	}
}

func (s *NetworkValidationSuite) Test_invalid_type_for_mac_address() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(1234),
		core.ScalarFromFloat(12.34),
		core.ScalarFromBool(true),
	}

	for _, value := range invalidValues {
		diagnostics := IsMACAddress()("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *NetworkValidationSuite) Test_valid_cidr_notation() {
	validCIDRs := []string{
		"127.0.0.1/32",
		"192.169.1.2/24",
	}

	for _, cidr := range validCIDRs {
		diagnostics := IsCIDR()("exampleField", core.ScalarFromString(cidr))
		s.Assert().Empty(diagnostics)
	}
}

func (s *NetworkValidationSuite) Test_invalid_cidr_notation() {
	invalidCIDRs := []string{
		"256.0.0.1/24", // Invalid octet
		"255.0.0.1/33", // Invalid significant bits length
	}

	for _, cidr := range invalidCIDRs {
		diagnostics := IsCIDR()("exampleField", core.ScalarFromString(cidr))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be in valid CIDR notation")
	}
}

func (s *NetworkValidationSuite) Test_invalid_type_for_cidr() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(1234),
		core.ScalarFromFloat(12.34),
		core.ScalarFromBool(true),
	}

	for _, value := range invalidValues {
		diagnostics := IsCIDR()("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *NetworkValidationSuite) Test_valid_cidr_network_values() {
	validValues := []string{
		"127.0.0.0/8",
		"192.169.0.0/16",
	}
	for _, value := range validValues {
		diagnostics := IsCIDRNetwork(8, 16)("exampleField", core.ScalarFromString(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *NetworkValidationSuite) Test_invalid_cidr_network_values() {
	validValues := []string{
		"127.0.0.0/32",
		"192.169.0.0/24",
	}
	for _, value := range validValues {
		diagnostics := IsCIDRNetwork(8, 16)("exampleField", core.ScalarFromString(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(
			diagnostics[0].Message,
			"must contain a network value with significant bits between 8 and 16",
		)
	}
}

func (s *NetworkValidationSuite) Test_invalid_type_for_cidr_network() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(3215),
		core.ScalarFromFloat(11.34),
		core.ScalarFromBool(false),
	}

	for _, value := range invalidValues {
		diagnostics := IsCIDRNetwork(8, 16)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *NetworkValidationSuite) Test_valid_port_numbers() {
	validPorts := []int{
		80,   // HTTP
		443,  // HTTPS
		22,   // SSH
		8080, // Alternative HTTP
		3306, // MySQL
	}

	for _, port := range validPorts {
		diagnostics := IsPortNumber()("exampleField", core.ScalarFromInt(port))
		s.Assert().Empty(diagnostics)
	}
}

func (s *NetworkValidationSuite) Test_invalid_port_numbers() {
	invalidPorts := []int{
		-1,
		65536,
		70000,
	}

	for _, port := range invalidPorts {
		diagnostics := IsPortNumber()("exampleField", core.ScalarFromInt(port))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a valid port number")
	}
}

func (s *NetworkValidationSuite) Test_invalid_type_for_port_number() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromString("not-a-port"),
		core.ScalarFromFloat(80.5),
		core.ScalarFromBool(true),
	}

	for _, value := range invalidValues {
		diagnostics := IsPortNumber()("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be an integer")
	}
}

func (s *NetworkValidationSuite) Test_port_or_zero_values() {
	validPorts := []int{
		0,    // Zero is valid
		80,   // HTTP
		443,  // HTTPS
		22,   // SSH
		8080, // Alternative HTTP
	}

	for _, port := range validPorts {
		diagnostics := IsPortNumberOrZero()("exampleField", core.ScalarFromInt(port))
		s.Assert().Empty(diagnostics)
	}
}

func (s *NetworkValidationSuite) Test_invalid_port_or_zero_values() {
	invalidPorts := []int{
		-1,    // Negative port
		65536, // Above maximum port number
		70000, // Invalid port number
	}

	for _, port := range invalidPorts {
		diagnostics := IsPortNumberOrZero()("exampleField", core.ScalarFromInt(port))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a valid port number (1-65535) or zero")
	}
}

func (s *NetworkValidationSuite) Test_invalid_type_for_port_or_zero() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromString("not-a-port"),
		core.ScalarFromFloat(74.5),
		core.ScalarFromBool(true),
	}

	for _, value := range invalidValues {
		diagnostics := IsPortNumberOrZero()("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be an integer")
	}
}

func TestNetworkValidationSuite(t *testing.T) {
	suite.Run(t, new(NetworkValidationSuite))
}
