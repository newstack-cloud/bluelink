package substitutions

import (
	"testing"

	"github.com/coreos/go-json"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/stretchr/testify/suite"
)

type UtilsTestSuite struct {
	suite.Suite
}

func (s *UtilsTestSuite) Test_accurately_calculates_json_string_or_sub_start_position_1() {
	// End of node is on the same line as the end of the key for a mapping.
	// Reference input:
	// {
	//   "key": "${variables.myVar}"
	// }
	node := &json.Node{
		// Start of the string value.
		Start:    11,
		End:      31,
		KeyStart: 0,
		// KeyEnd as calculated by coreos/go-json
		// is the offset of the ":" character.
		KeyEnd: 9,
		Value:  "${variables.myVar}",
	}

	linePositions := []int{0, 1, 31, 33}

	sourceMeta := DetermineJSONSourceStartMeta(
		node,
		"${variables.myVar}",
		linePositions,
	)
	s.Assert().Equal(
		&source.Meta{
			Position: source.Position{
				Line: 2,
				// The column of the opening quote
				// for the string. (counting from 1)
				Column: 10,
			},
			EndPosition: &source.Position{
				Line:   2,
				Column: 30,
			},
		},
		sourceMeta,
	)
}

func (s *UtilsTestSuite) Test_accurately_calculates_json_string_or_sub_start_position_2() {
	// End of node is on the line after the key for a mapping.
	// {
	//   "key": "${variables.myVar}"
	// }
	node := &json.Node{
		// Start of the string value.
		Start: 11,
		// End of node is calculated differently here
		// and is considered to be on the next line.
		End:      33,
		KeyStart: 0,
		// KeyEnd as calculated by coreos/go-json
		// is the offset of the ":" character.
		KeyEnd: 9,
		Value:  "${variables.myVar}",
	}
	linePositions := []int{0, 1, 31, 33}

	sourceMeta := DetermineJSONSourceStartMeta(
		node,
		"${variables.myVar}",
		linePositions,
	)
	s.Assert().Equal(
		&source.Meta{
			Position: source.Position{
				Line: 2,
				// The column of the opening quote
				// for the string. (counting from 1)
				Column: 10,
			},
			EndPosition: &source.Position{
				Line:   2,
				Column: 30,
			},
		},
		sourceMeta,
	)
}

func (s *UtilsTestSuite) Test_accurately_calculates_json_string_or_sub_start_position_3() {
	// End of node is on the same line as the start of the value for a sequence.
	// {
	//   "key": [
	//     "${variables.myVar}"
	// 	 ]
	// }
	node := &json.Node{
		Start: 17,
		End:   38,
		Value: "${variables.myVar}",
	}
	linePositions := []int{0, 1, 12, 36, 40, 42}

	sourceMeta := DetermineJSONSourceStartMeta(
		node,
		"${variables.myVar}",
		linePositions,
	)
	s.Assert().Equal(
		&source.Meta{
			Position: source.Position{
				Line: 3,
				// The column of the opening quote
				// for the string. (counting from 1)
				Column: 5,
			},
			EndPosition: &source.Position{
				Line:   3,
				Column: 24,
			},
		},
		sourceMeta,
	)
}

func (s *UtilsTestSuite) Test_accurately_calculates_json_string_or_sub_start_position_4() {
	// End of node is on the line after the string value in the sequence.
	// {
	//   "key": [
	//     "${variables.myVar}"
	// 	 ]
	// }
	node := &json.Node{
		Start: 17,
		// End of node is calculated differently here
		// and is considered to be on the next line.
		End:   41,
		Value: "${variables.myVar}",
	}
	linePositions := []int{0, 1, 12, 36, 40, 42}

	sourceMeta := DetermineJSONSourceStartMeta(
		node,
		"${variables.myVar}",
		linePositions,
	)
	s.Assert().Equal(
		&source.Meta{
			Position: source.Position{
				Line: 3,
				// The column of the opening quote
				// for the string. (counting from 1)
				Column: 5,
			},
			EndPosition: &source.Position{
				Line:   3,
				Column: 24,
			},
		},
		sourceMeta,
	)
}

func TestUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(UtilsTestSuite))
}
