package subengine

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type TransformMappingNodeTestSuite struct {
	suite.Suite
}

const (
	stringText = "This is a string"
)

func (s *TransformMappingNodeTestSuite) Test_transform_go_int_value_to_mapping_node() {
	intVal := 42
	mappingNode := GoValueToMappingNode(intVal)
	s.Assert().Equal(&core.MappingNode{
		Scalar: &core.ScalarValue{
			IntValue: &intVal,
		},
	}, mappingNode)
}

func (s *TransformMappingNodeTestSuite) Test_transform_go_float_value_to_mapping_node() {
	floatVal := 4092.4029
	mappingNode := GoValueToMappingNode(floatVal)
	s.Assert().Equal(&core.MappingNode{
		Scalar: &core.ScalarValue{
			FloatValue: &floatVal,
		},
	}, mappingNode)
}

func (s *TransformMappingNodeTestSuite) Test_transform_go_string_value_to_mapping_node() {
	stringVal := stringText
	mappingNode := GoValueToMappingNode(stringVal)
	s.Assert().Equal(&core.MappingNode{
		Scalar: &core.ScalarValue{
			StringValue: &stringVal,
		},
	}, mappingNode)
}

func (s *TransformMappingNodeTestSuite) Test_transform_go_bool_value_to_mapping_node() {
	boolVal := true
	mappingNode := GoValueToMappingNode(boolVal)
	s.Assert().Equal(&core.MappingNode{
		Scalar: &core.ScalarValue{
			BoolValue: &boolVal,
		},
	}, mappingNode)
}

func (s *TransformMappingNodeTestSuite) Test_transform_go_byte_slice_to_mapping_node() {
	byteSlice := []byte("Hello, World!")
	mappingNode := GoValueToMappingNode(byteSlice)
	expectedBytes := []byte("Hello, World!")
	s.Assert().Equal(&core.MappingNode{
		Scalar: &core.ScalarValue{
			BytesValue: &expectedBytes,
		},
	}, mappingNode)
}

func (s *TransformMappingNodeTestSuite) Test_transform_bytes_mapping_node_to_go_value() {
	inputBytes := []byte("Example bytes")
	mappingNode := &core.MappingNode{
		Scalar: &core.ScalarValue{
			BytesValue: &inputBytes,
		},
	}
	bytesVal := MappingNodeToGoValue(mappingNode)
	s.Assert().Equal([]byte("Example bytes"), bytesVal)
}

func (s *TransformMappingNodeTestSuite) Test_transform_go_mixed_slice_value_to_mapping_node() {
	intVal := 10000
	stringVal := "string"
	floatVal := 3.14
	boolVal := true
	sliceVal := []interface{}{intVal, stringVal, floatVal, boolVal}
	mappingNode := GoValueToMappingNode(sliceVal)
	s.Assert().Equal(&core.MappingNode{
		Items: []*core.MappingNode{
			{
				Scalar: &core.ScalarValue{
					IntValue: &intVal,
				},
			},
			{
				Scalar: &core.ScalarValue{
					StringValue: &stringVal,
				},
			},
			{
				Scalar: &core.ScalarValue{
					FloatValue: &floatVal,
				},
			},
			{
				Scalar: &core.ScalarValue{
					BoolValue: &boolVal,
				},
			},
		},
	}, mappingNode)
}

func (s *TransformMappingNodeTestSuite) Test_transform_go_typed_slice_value_to_mapping_node() {
	stringVal1 := "This is string 1"
	stringVal2 := "This is string 2"
	stringVal3 := "This is string 3"
	sliceVal := []string{stringVal1, stringVal2, stringVal3}
	mappingNode := GoValueToMappingNode(sliceVal)
	s.Assert().Equal(&core.MappingNode{
		Items: []*core.MappingNode{
			{
				Scalar: &core.ScalarValue{
					StringValue: &stringVal1,
				},
			},
			{
				Scalar: &core.ScalarValue{
					StringValue: &stringVal2,
				},
			},
			{
				Scalar: &core.ScalarValue{
					StringValue: &stringVal3,
				},
			},
		},
	}, mappingNode)
}

func (s *TransformMappingNodeTestSuite) Test_transform_go_mixed_map_to_mapping_node() {
	key1Val := 42
	key2Val := "string"
	key3Val := 7.65
	key4Val := true
	mapVal := map[string]interface{}{
		"key1": key1Val,
		"key2": key2Val,
		"key3": key3Val,
		"key4": key4Val,
	}
	mappingNode := GoValueToMappingNode(mapVal)
	s.Assert().Equal(&core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"key1": {
				Scalar: &core.ScalarValue{
					IntValue: &key1Val,
				},
			},
			"key2": {
				Scalar: &core.ScalarValue{
					StringValue: &key2Val,
				},
			},
			"key3": {
				Scalar: &core.ScalarValue{
					FloatValue: &key3Val,
				},
			},
			"key4": {
				Scalar: &core.ScalarValue{
					BoolValue: &key4Val,
				},
			},
		},
	}, mappingNode)
}

func (s *TransformMappingNodeTestSuite) Test_transform_go_typed_map_to_mapping_node() {
	key1Val := 42.589
	key2Val := 1004.30
	key3Val := 7.65
	key4Val := 9.8342
	mapVal := map[string]float64{
		"key1": key1Val,
		"key2": key2Val,
		"key3": key3Val,
		"key4": key4Val,
	}
	mappingNode := GoValueToMappingNode(mapVal)
	s.Assert().Equal(&core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"key1": {
				Scalar: &core.ScalarValue{
					FloatValue: &key1Val,
				},
			},
			"key2": {
				Scalar: &core.ScalarValue{
					FloatValue: &key2Val,
				},
			},
			"key3": {
				Scalar: &core.ScalarValue{
					FloatValue: &key3Val,
				},
			},
			"key4": {
				Scalar: &core.ScalarValue{
					FloatValue: &key4Val,
				},
			},
		},
	}, mappingNode)
}

func (s *TransformMappingNodeTestSuite) Test_transform_go_struct_to_mapping_node() {
	type testStruct struct {
		Key1 int
		Key2 string
		Key3 float64
		Key4 bool
	}
	structVal := testStruct{
		Key1: 42,
		Key2: "string",
		Key3: 7.65,
		Key4: true,
	}
	mappingNode := GoValueToMappingNode(structVal)
	s.Assert().Equal(&core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"Key1": {
				Scalar: &core.ScalarValue{
					IntValue: &structVal.Key1,
				},
			},
			"Key2": {
				Scalar: &core.ScalarValue{
					StringValue: &structVal.Key2,
				},
			},
			"Key3": {
				Scalar: &core.ScalarValue{
					FloatValue: &structVal.Key3,
				},
			},
			"Key4": {
				Scalar: &core.ScalarValue{
					BoolValue: &structVal.Key4,
				},
			},
		},
	}, mappingNode)
}

func (s *TransformMappingNodeTestSuite) Test_transform_int_mapping_node_to_go_value() {
	inputInt := 409282
	mappingNode := &core.MappingNode{
		Scalar: &core.ScalarValue{
			IntValue: &inputInt,
		},
	}
	intVal := MappingNodeToGoValue(mappingNode)
	s.Assert().Equal(409282, intVal)
}

func (s *TransformMappingNodeTestSuite) Test_transform_float_mapping_node_to_go_value() {
	inputFloat := 3.14159
	mappingNode := &core.MappingNode{
		Scalar: &core.ScalarValue{
			FloatValue: &inputFloat,
		},
	}
	floatVal := MappingNodeToGoValue(mappingNode)
	s.Assert().Equal(3.14159, floatVal)
}

func (s *TransformMappingNodeTestSuite) Test_transform_string_mapping_node_to_go_value() {
	inputString := stringText
	mappingNode := &core.MappingNode{
		Scalar: &core.ScalarValue{
			StringValue: &inputString,
		},
	}
	stringVal := MappingNodeToGoValue(mappingNode)
	s.Assert().Equal(stringText, stringVal)
}

func (s *TransformMappingNodeTestSuite) Test_transform_bool_mapping_node_to_go_value() {
	inputBool := true
	mappingNode := &core.MappingNode{
		Scalar: &core.ScalarValue{
			BoolValue: &inputBool,
		},
	}
	boolVal := MappingNodeToGoValue(mappingNode)
	s.Assert().Equal(true, boolVal)
}

func (s *TransformMappingNodeTestSuite) Test_transform_slice_mapping_node_to_go_value() {
	intVal := 10000
	stringVal := "string"
	floatVal := 3.14
	boolVal := true
	mappingNode := &core.MappingNode{
		Items: []*core.MappingNode{
			{
				Scalar: &core.ScalarValue{
					IntValue: &intVal,
				},
			},
			{
				Scalar: &core.ScalarValue{
					StringValue: &stringVal,
				},
			},
			{
				Scalar: &core.ScalarValue{
					FloatValue: &floatVal,
				},
			},
			{
				Scalar: &core.ScalarValue{
					BoolValue: &boolVal,
				},
			},
		},
	}
	sliceVal := MappingNodeToGoValue(mappingNode)
	s.Assert().Equal([]interface{}{10000, "string", 3.14, true}, sliceVal)
}

func (s *TransformMappingNodeTestSuite) Test_transform_map_mapping_node_to_go_value() {
	key1Val := 42
	key2Val := "string"
	key3Val := 7.65
	key4Val := true
	mappingNode := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"key1": {
				Scalar: &core.ScalarValue{
					IntValue: &key1Val,
				},
			},
			"key2": {
				Scalar: &core.ScalarValue{
					StringValue: &key2Val,
				},
			},
			"key3": {
				Scalar: &core.ScalarValue{
					FloatValue: &key3Val,
				},
			},
			"key4": {
				Scalar: &core.ScalarValue{
					BoolValue: &key4Val,
				},
			},
		},
	}
	mapVal := MappingNodeToGoValue(mappingNode)
	s.Assert().Equal(map[string]interface{}{
		"key1": 42,
		"key2": "string",
		"key3": 7.65,
		"key4": true,
	}, mapVal)
}

func TestTransformMappingNodeTestSuite(t *testing.T) {
	suite.Run(t, new(TransformMappingNodeTestSuite))
}
