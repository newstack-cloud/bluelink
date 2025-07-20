package core

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type UtilsTestSuite struct {
	suite.Suite
}

func (s *UtilsTestSuite) Test_converts_string_slice_to_mapping_node() {
	strings := []string{"value one", "value two", "value three"}
	mappingNode := MappingNodeFromStringSlice(strings)

	s.Assert().Equal(
		&MappingNode{
			Items: []*MappingNode{
				MappingNodeFromString("value one"),
				MappingNodeFromString("value two"),
				MappingNodeFromString("value three"),
			},
		},
		mappingNode,
	)
}

func (s *UtilsTestSuite) Test_converts_int_slice_to_mapping_node() {
	ints := []int64{504, 1024, 2048}
	mappingNode := MappingNodeFromIntSlice(ints)

	s.Assert().Equal(
		&MappingNode{
			Items: []*MappingNode{
				MappingNodeFromInt(504),
				MappingNodeFromInt(1024),
				MappingNodeFromInt(2048),
			},
		},
		mappingNode,
	)
}

func (s *UtilsTestSuite) Test_converts_float_slice_to_mapping_node() {
	floats := []float64{1.23, 4.56, 7.89}
	mappingNode := MappingNodeFromFloatSlice(floats)

	s.Assert().Equal(
		&MappingNode{
			Items: []*MappingNode{
				MappingNodeFromFloat(1.23),
				MappingNodeFromFloat(4.56),
				MappingNodeFromFloat(7.89),
			},
		},
		mappingNode,
	)
}

func (s *UtilsTestSuite) Test_converts_bool_to_mapping_node() {
	bools := []bool{true, false, true}
	mappingNode := MappingNodeFromBoolSlice(bools)

	s.Assert().Equal(
		&MappingNode{
			Items: []*MappingNode{
				MappingNodeFromBool(true),
				MappingNodeFromBool(false),
				MappingNodeFromBool(true),
			},
		},
		mappingNode,
	)
}

func (s *UtilsTestSuite) Test_converts_string_map_to_mapping_node() {
	stringMap := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	mappingNode := MappingNodeFromStringMap(stringMap)

	s.Assert().Equal(
		&MappingNode{
			Fields: map[string]*MappingNode{
				"key1": MappingNodeFromString("value1"),
				"key2": MappingNodeFromString("value2"),
				"key3": MappingNodeFromString("value3"),
			},
		},
		mappingNode,
	)
}

func (s *UtilsTestSuite) Test_converts_int_map_to_mapping_node() {
	intMap := map[string]int64{
		"key1": 100,
		"key2": 200,
		"key3": 300,
	}
	mappingNode := MappingNodeFromIntMap(intMap)

	s.Assert().Equal(
		&MappingNode{
			Fields: map[string]*MappingNode{
				"key1": MappingNodeFromInt(100),
				"key2": MappingNodeFromInt(200),
				"key3": MappingNodeFromInt(300),
			},
		},
		mappingNode,
	)
}

func (s *UtilsTestSuite) Test_converts_float_map_to_mapping_node() {
	floatMap := map[string]float64{
		"key1": 13.14039,
		"key2": 2.2102,
		"key3": 3.3,
	}
	mappingNode := MappingNodeFromFloatMap(floatMap)

	s.Assert().Equal(
		&MappingNode{
			Fields: map[string]*MappingNode{
				"key1": MappingNodeFromFloat(13.14039),
				"key2": MappingNodeFromFloat(2.2102),
				"key3": MappingNodeFromFloat(3.3),
			},
		},
		mappingNode,
	)
}

func (s *UtilsTestSuite) Test_converts_bool_map_to_mapping_node() {
	boolMap := map[string]bool{
		"key1": true,
		"key2": false,
		"key3": true,
	}
	mappingNode := MappingNodeFromBoolMap(boolMap)

	s.Assert().Equal(
		&MappingNode{
			Fields: map[string]*MappingNode{
				"key1": MappingNodeFromBool(true),
				"key2": MappingNodeFromBool(false),
				"key3": MappingNodeFromBool(true),
			},
		},
		mappingNode,
	)
}

func (s *UtilsTestSuite) Test_converts_mapping_node_to_string_slice() {
	mappingNode := &MappingNode{
		Items: []*MappingNode{
			MappingNodeFromString("value one"),
			MappingNodeFromString("value two"),
			MappingNodeFromString("value three"),
		},
	}
	strings := StringSliceValue(mappingNode)

	s.Assert().Equal([]string{"value one", "value two", "value three"}, strings)
}

func (s *UtilsTestSuite) Test_converts_empty_mapping_node_to_empty_string_slice() {
	mappingNode := &MappingNode{}
	strings := StringSliceValue(mappingNode)

	s.Assert().Equal([]string{}, strings)
}

func (s *UtilsTestSuite) Test_converts_mapping_node_to_int_slice() {
	mappingNode := &MappingNode{
		Items: []*MappingNode{
			MappingNodeFromInt(504),
			MappingNodeFromInt(1024),
			MappingNodeFromInt(2048),
		},
	}
	ints := IntSliceValue(mappingNode)

	s.Assert().Equal([]int{504, 1024, 2048}, ints)
}

func (s *UtilsTestSuite) Test_converts_empty_mapping_node_to_empty_int_slice() {
	mappingNode := &MappingNode{}
	ints := IntSliceValue(mappingNode)

	s.Assert().Equal([]int{}, ints)
}

func (s *UtilsTestSuite) Test_converts_mapping_node_to_float_slice() {
	mappingNode := &MappingNode{
		Items: []*MappingNode{
			MappingNodeFromFloat(1.23),
			MappingNodeFromFloat(4.56),
			MappingNodeFromFloat(7.89),
		},
	}
	floats := FloatSliceValue(mappingNode)

	s.Assert().Equal([]float64{1.23, 4.56, 7.89}, floats)
}

func (s *UtilsTestSuite) Test_converts_empty_mapping_node_to_empty_float_slice() {
	mappingNode := &MappingNode{}
	floats := FloatSliceValue(mappingNode)

	s.Assert().Equal([]float64{}, floats)
}

func (s *UtilsTestSuite) Test_converts_mapping_node_to_bool_slice() {
	mappingNode := &MappingNode{
		Items: []*MappingNode{
			MappingNodeFromBool(true),
			MappingNodeFromBool(false),
			MappingNodeFromBool(true),
		},
	}
	bools := BoolSliceValue(mappingNode)

	s.Assert().Equal([]bool{true, false, true}, bools)
}

func (s *UtilsTestSuite) Test_converts_empty_mapping_node_to_empty_bool_slice() {
	mappingNode := &MappingNode{}
	bools := BoolSliceValue(mappingNode)

	s.Assert().Equal([]bool{}, bools)
}

func (s *UtilsTestSuite) Test_converts_mapping_node_to_string_map() {
	mappingNode := &MappingNode{
		Fields: map[string]*MappingNode{
			"key1": MappingNodeFromString("value1"),
			"key2": MappingNodeFromString("value2"),
			"key3": MappingNodeFromString("value3"),
		},
	}
	stringMap := StringMapValue(mappingNode)

	s.Assert().Equal(
		map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
		stringMap,
	)
}

func (s *UtilsTestSuite) Test_converts_empty_mapping_node_to_empty_string_map() {
	mappingNode := &MappingNode{}
	stringMap := StringMapValue(mappingNode)

	s.Assert().Equal(map[string]string{}, stringMap)
}

func (s *UtilsTestSuite) Test_converts_mapping_node_to_int_map() {
	mappingNode := &MappingNode{
		Fields: map[string]*MappingNode{
			"key1": MappingNodeFromInt(100),
			"key2": MappingNodeFromInt(200),
			"key3": MappingNodeFromInt(300),
		},
	}
	intMap := IntMapValue(mappingNode)

	s.Assert().Equal(
		map[string]int{
			"key1": 100,
			"key2": 200,
			"key3": 300,
		},
		intMap,
	)
}

func (s *UtilsTestSuite) Test_converts_empty_mapping_node_to_empty_int_map() {
	mappingNode := &MappingNode{}
	intMap := IntMapValue(mappingNode)

	s.Assert().Equal(map[string]int{}, intMap)
}

func (s *UtilsTestSuite) Test_converts_mapping_node_to_float_map() {
	mappingNode := &MappingNode{
		Fields: map[string]*MappingNode{
			"key1": MappingNodeFromFloat(13.14039),
			"key2": MappingNodeFromFloat(2.2102),
			"key3": MappingNodeFromFloat(3.3),
		},
	}
	floatMap := FloatMapValue(mappingNode)

	s.Assert().Equal(
		map[string]float64{
			"key1": 13.14039,
			"key2": 2.2102,
			"key3": 3.3,
		},
		floatMap,
	)
}

func (s *UtilsTestSuite) Test_converts_empty_mapping_node_to_empty_float_map() {
	mappingNode := &MappingNode{}
	floatMap := FloatMapValue(mappingNode)

	s.Assert().Equal(map[string]float64{}, floatMap)
}

func (s *UtilsTestSuite) Test_converts_mapping_node_to_bool_map() {
	mappingNode := &MappingNode{
		Fields: map[string]*MappingNode{
			"key1": MappingNodeFromBool(true),
			"key2": MappingNodeFromBool(false),
			"key3": MappingNodeFromBool(true),
		},
	}
	boolMap := BoolMapValue(mappingNode)

	s.Assert().Equal(
		map[string]bool{
			"key1": true,
			"key2": false,
			"key3": true,
		},
		boolMap,
	)
}

func (s *UtilsTestSuite) Test_converts_empty_mapping_node_to_empty_bool_map() {
	mappingNode := &MappingNode{}
	boolMap := BoolMapValue(mappingNode)

	s.Assert().Equal(map[string]bool{}, boolMap)
}

func (s *UtilsTestSuite) Test_mapping_node_fields() {
	mappingNode := MappingNodeFields(
		"key1", MappingNodeFromString("value1"),
		"key2", MappingNodeFromInt(200),
		"key3", MappingNodeFromFloat(3.3),
		"key4", MappingNodeFromBool(true),
	)

	s.Assert().Equal(
		map[string]*MappingNode{
			"key1": MappingNodeFromString("value1"),
			"key2": MappingNodeFromInt(200),
			"key3": MappingNodeFromFloat(3.3),
			"key4": MappingNodeFromBool(true),
		},
		mappingNode.Fields,
	)
}

func (s *UtilsTestSuite) Test_mapping_node_fields_empty_for_odd_amount_of_args() {
	mappingNode := MappingNodeFields(
		"key1",
		MappingNodeFromString("value1"),
		"key2",
	)

	s.Assert().Equal(
		map[string]*MappingNode{},
		mappingNode.Fields,
	)
}

func (s *UtilsTestSuite) Test_mapping_node_fields_skips_fields_with_invalid_types() {
	mappingNode := MappingNodeFields(
		"key1", MappingNodeFromString("value1"),
		"key2", "not a mapping node",
		"key3", MappingNodeFromFloat(3.3),
	)

	s.Assert().Equal(
		map[string]*MappingNode{
			"key1": MappingNodeFromString("value1"),
			"key3": MappingNodeFromFloat(3.3),
		},
		mappingNode.Fields,
	)
}

func (s *UtilsTestSuite) Test_mapping_node_items() {
	mappingNode := MappingNodeItems(
		MappingNodeFromString("value1"),
		MappingNodeFromInt(200),
		MappingNodeFromFloat(3.3),
		MappingNodeFromBool(true),
	)

	s.Assert().Equal(
		[]*MappingNode{
			MappingNodeFromString("value1"),
			MappingNodeFromInt(200),
			MappingNodeFromFloat(3.3),
			MappingNodeFromBool(true),
		},
		mappingNode.Items,
	)
}

func TestUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(UtilsTestSuite))
}
