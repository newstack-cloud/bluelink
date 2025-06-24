package substitutions

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type ParseSubstitutionsTestSuite struct{}

var _ = Suite(&ParseSubstitutionsTestSuite{})

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_multiple_substitutions(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		"${replace(datasources.host.domain, \"${}\", \"\")}/${variables.version}/app",
		// Emulate the inner substitution starting on line 200, column 100,
		// outer column is 98.
		// Source meta values of substitution components are offset from the start
		// of the input string.
		&source.Meta{Position: source.Position{
			Line:   200,
			Column: 100,
		}},
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 4)

	arg2 := "${}"
	arg3 := ""
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			Function: &SubstitutionFunctionExpr{
				FunctionName: "replace",
				Arguments: []*SubstitutionFunctionArg{
					{
						Value: &Substitution{
							DataSourceProperty: &SubstitutionDataSourceProperty{
								DataSourceName: "host",
								FieldName:      "domain",
								SourceMeta: &source.Meta{
									Position: source.Position{
										Line:   200,
										Column: 110,
									},
									EndPosition: &source.Position{
										Line:   200,
										Column: 133,
									},
								},
							},
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   200,
									Column: 110,
								},
								EndPosition: &source.Position{
									Line:   200,
									Column: 133,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   200,
								Column: 110,
							},
							EndPosition: &source.Position{
								Line:   200,
								Column: 133,
							},
						},
					},
					{
						Value: &Substitution{
							StringValue: &arg2,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   200,
									Column: 135,
								},
								EndPosition: &source.Position{
									Line:   200,
									Column: 140,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   200,
								Column: 135,
							},
							EndPosition: &source.Position{
								Line:   200,
								Column: 140,
							},
						},
					},
					{
						Value: &Substitution{
							StringValue: &arg3,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   200,
									Column: 142,
								},
								EndPosition: &source.Position{
									Line:   200,
									Column: 144,
								}},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   200,
								Column: 142,
							},
							EndPosition: &source.Position{
								Line:   200,
								Column: 144,
							},
						},
					},
				},
				Path: []*SubstitutionPathItem{},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   200,
						Column: 102,
					},
					EndPosition: &source.Position{
						Line:   200,
						Column: 145,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   200,
					Column: 102,
				},
				EndPosition: &source.Position{
					Line:   200,
					Column: 145,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   200,
				Column: 100,
			},
			EndPosition: &source.Position{
				Line:   200,
				Column: 146,
			},
		},
	})

	pathSeparator := "/"
	c.Assert(parsed[1], DeepEquals, &StringOrSubstitution{
		StringValue: &pathSeparator,
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   200,
				Column: 146,
			},
			EndPosition: &source.Position{
				Line:   200,
				Column: 147,
			},
		},
	})

	c.Assert(parsed[2], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			Variable: &SubstitutionVariable{
				VariableName: "version",
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   200,
						Column: 149,
					},
					EndPosition: &source.Position{
						Line:   200,
						Column: 166,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   200,
					Column: 149,
				},
				EndPosition: &source.Position{
					Line:   200,
					Column: 166,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   200,
				Column: 147,
			},
			EndPosition: &source.Position{
				Line:   200,
				Column: 167,
			},
		},
	})

	pathSuffix := "/app"
	c.Assert(parsed[3], DeepEquals, &StringOrSubstitution{
		StringValue: &pathSuffix,
		SourceMeta: &source.Meta{Position: source.Position{
			Line:   200,
			Column: 167,
		}},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_a_data_source_ref_sub_1(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${datasources["coreInfra.v1"]["topic.v2"][0]}`,
		nil,
		true,  // outputLineInfo
		false, // ignoreParentColumn
		0,     // parentContextPrecedingCharCount
	)
	index := int64(0)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			DataSourceProperty: &SubstitutionDataSourceProperty{
				DataSourceName:    "coreInfra.v1",
				FieldName:         "topic.v2",
				PrimitiveArrIndex: &index,
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 45,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 3,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 45,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 45,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_a_data_source_ref_sub_2(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		"${datasources.coreInfra1.topics[1]}",
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	index := int64(1)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			DataSourceProperty: &SubstitutionDataSourceProperty{
				DataSourceName:    "coreInfra1",
				FieldName:         "topics",
				PrimitiveArrIndex: &index,
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 35,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 3,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 35,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 35,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_a_child_ref_sub_1(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${children["core-infrastructure.v1"].cacheNodes[].host}`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	index := int64(0)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			Child: &SubstitutionChild{
				ChildName: "core-infrastructure.v1",
				Path: []*SubstitutionPathItem{
					{
						FieldName: "cacheNodes",
					},
					{
						ArrayIndex: &index,
					},
					{
						FieldName: "host",
					},
				},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 55,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 3,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 55,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 55,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_a_resource_ref_sub_1(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${resources.saveOrderFunction.metadata.annotations["annotationKey.v1"]}`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			ResourceProperty: &SubstitutionResourceProperty{
				ResourceName: "saveOrderFunction",
				Path: []*SubstitutionPathItem{
					{
						FieldName: "metadata",
					},
					{
						FieldName: "annotations",
					},
					{
						FieldName: "annotationKey.v1",
					},
				},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 71,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 3,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 71,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 71,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_a_resource_ref_sub_2(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${resources["save-order-function.v1"].spec.functionArn}`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			ResourceProperty: &SubstitutionResourceProperty{
				ResourceName: "save-order-function.v1",
				Path: []*SubstitutionPathItem{
					{
						FieldName: "spec",
					},
					{
						FieldName: "functionArn",
					},
				},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 55,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 3,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 55,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 55,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_a_resource_ref_sub_3(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${saveOrderFunction}`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			ResourceProperty: &SubstitutionResourceProperty{
				ResourceName: "saveOrderFunction",
				Path:         []*SubstitutionPathItem{},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 20,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 3,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 20,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 20,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_a_resource_ref_sub_4(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${resources.contentBuckets[2].spec.bucketArn}`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	templateIndex := int64(2)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			ResourceProperty: &SubstitutionResourceProperty{
				ResourceName:              "contentBuckets",
				ResourceEachTemplateIndex: &templateIndex,
				Path: []*SubstitutionPathItem{
					{
						FieldName: "spec",
					},
					{
						FieldName: "bucketArn",
					},
				},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 45,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 3,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 45,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 45,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_a_value_ref_sub_1(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${values.s3Bucket.info["objectConfig"][3]}`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	arrIndex := int64(3)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			ValueReference: &SubstitutionValueReference{
				ValueName: "s3Bucket",
				Path: []*SubstitutionPathItem{
					{
						FieldName: "info",
					},
					{
						FieldName: "objectConfig",
					},
					{
						ArrayIndex: &arrIndex,
					},
				},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 42,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 3,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 42,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 42,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_a_value_ref_sub_2(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${values.googleCloudBuckets[1].name}`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	arrIndex := int64(1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			ValueReference: &SubstitutionValueReference{
				ValueName: "googleCloudBuckets",
				Path: []*SubstitutionPathItem{
					{
						ArrayIndex: &arrIndex,
					},
					{
						FieldName: "name",
					},
				},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 36,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 3,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 36,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 36,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_a_value_ref_sub_3(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${values.queueUrl}`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			ValueReference: &SubstitutionValueReference{
				ValueName: "queueUrl",
				Path:      []*SubstitutionPathItem{},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 18,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 3,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 18,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 18,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_current_elem_ref(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${elem.queueUrl}`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			ElemReference: &SubstitutionElemReference{
				Path: []*SubstitutionPathItem{
					{
						FieldName: "queueUrl",
					},
				},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 16,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 3,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 16,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 16,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_string_with_current_elem_index_ref(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${i}`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			ElemIndexReference: &SubstitutionElemIndexReference{
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 3,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 4,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 3,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 4,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 4,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_sub_string_with_a_string_literal(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		// Includes 4-byte character to ensure runes are being counted
		// for columns and not bytes.
		`${  "This is a \"string\" literal𒀁"    }`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	expectedStrVal := "This is a \"string\" literal𒀁"
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			StringValue: &expectedStrVal,
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 5,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 34,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 40,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_sub_string_with_a_func_call_1(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${  substr(trim("This is a \"string\" literal"), 0, 3)    }`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	trimArg := "This is a \"string\" literal"
	arg2 := int64(0)
	arg3 := int64(3)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			Function: &SubstitutionFunctionExpr{
				FunctionName: "substr",
				Arguments: []*SubstitutionFunctionArg{
					{
						Value: &Substitution{
							Function: &SubstitutionFunctionExpr{
								FunctionName: "trim",
								Arguments: []*SubstitutionFunctionArg{
									{
										Value: &Substitution{
											StringValue: &trimArg,
											SourceMeta: &source.Meta{
												Position: source.Position{
													Line:   1,
													Column: 17,
												},
												EndPosition: &source.Position{
													Line:   1,
													Column: 47,
												},
											},
										},
										SourceMeta: &source.Meta{
											Position: source.Position{
												Line:   1,
												Column: 17,
											},
											EndPosition: &source.Position{
												Line:   1,
												Column: 47,
											},
										},
									},
								},
								Path: []*SubstitutionPathItem{},
								SourceMeta: &source.Meta{
									Position: source.Position{
										Line:   1,
										Column: 12,
									},
									EndPosition: &source.Position{
										Line:   1,
										Column: 48,
									},
								},
							},
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   1,
									Column: 12,
								},
								EndPosition: &source.Position{
									Line:   1,
									Column: 48,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   1,
								Column: 12,
							},
							EndPosition: &source.Position{
								Line:   1,
								Column: 48,
							},
						},
					},
					{
						Value: &Substitution{
							IntValue: &arg2,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   1,
									Column: 50,
								},
								EndPosition: &source.Position{
									Line:   1,
									Column: 51,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   1,
								Column: 50,
							},
							EndPosition: &source.Position{
								Line:   1,
								Column: 51,
							},
						},
					},
					{
						Value: &Substitution{
							IntValue: &arg3,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   1,
									Column: 53,
								},
								EndPosition: &source.Position{
									Line:   1,
									Column: 54,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   1,
								Column: 53,
							},
							EndPosition: &source.Position{
								Line:   1,
								Column: 54,
							},
						},
					},
				},
				Path: []*SubstitutionPathItem{},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 5,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 55,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 5,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 55,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 59,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_sub_string_with_a_func_call_2(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${  trim("This is a \"string\" literal", true)    }`,
		nil,
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	arg1 := "This is a \"string\" literal"
	arg2 := true
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			Function: &SubstitutionFunctionExpr{
				FunctionName: "trim",
				Arguments: []*SubstitutionFunctionArg{
					{
						Value: &Substitution{
							StringValue: &arg1,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   1,
									Column: 10,
								},
								EndPosition: &source.Position{
									Line:   1,
									Column: 40,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   1,
								Column: 10,
							},
							EndPosition: &source.Position{
								Line:   1,
								Column: 40,
							},
						},
					},
					{
						Value: &Substitution{
							BoolValue: &arg2,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   1,
									Column: 42,
								},
								EndPosition: &source.Position{
									Line:   1,
									Column: 46,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   1,
								Column: 42,
							},
							EndPosition: &source.Position{
								Line:   1,
								Column: 46,
							},
						},
					},
				},
				Path: []*SubstitutionPathItem{},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 5,
					},
					EndPosition: &source.Position{
						Line:   1,
						Column: 47,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 5,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 47,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 3,
			},
			EndPosition: &source.Position{
				Line:   1,
				Column: 51,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_sub_string_with_a_func_call_3(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${  format(25.40932102)   }`,
		// Emulate this substitution starting on line 100, column 50.
		// Source meta values of substitution components are offset from the start
		// of the input string.
		&source.Meta{Position: source.Position{
			Line:   100,
			Column: 50,
		}},
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	arg := 25.40932102
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			Function: &SubstitutionFunctionExpr{
				FunctionName: "format",
				Arguments: []*SubstitutionFunctionArg{
					{
						Value: &Substitution{
							FloatValue: &arg,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   100,
									Column: 61,
								},
								EndPosition: &source.Position{
									Line:   100,
									Column: 72,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   100,
								Column: 61,
							},
							EndPosition: &source.Position{
								Line:   100,
								Column: 72,
							},
						},
					},
				},
				Path: []*SubstitutionPathItem{},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   100,
						Column: 54,
					},
					EndPosition: &source.Position{
						Line:   100,
						Column: 73,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   100,
					Column: 54,
				},
				EndPosition: &source.Position{
					Line:   100,
					Column: 73,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   100,
				Column: 50,
			},
			EndPosition: &source.Position{
				Line:   100,
				Column: 77,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_sub_string_with_a_func_call_4(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${  object(total = 25.40932102, avg = 10.492, label = "Label")   }`,
		// Emulate this substitution starting on line 100, column 50.
		// Source meta values of substitution components are offset from the start
		// of the input string.
		&source.Meta{Position: source.Position{
			Line:   100,
			Column: 50,
		}},
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	arg1 := 25.40932102
	arg2 := 10.492
	arg3 := "Label"
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			Function: &SubstitutionFunctionExpr{
				FunctionName: "object",
				Arguments: []*SubstitutionFunctionArg{
					{
						Name: "total",
						Value: &Substitution{
							FloatValue: &arg1,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   100,
									Column: 69,
								},
								EndPosition: &source.Position{
									Line:   100,
									Column: 80,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   100,
								Column: 61,
							},
							EndPosition: &source.Position{
								Line:   100,
								Column: 80,
							},
						},
					},
					{
						Name: "avg",
						Value: &Substitution{
							FloatValue: &arg2,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   100,
									Column: 88,
								},
								EndPosition: &source.Position{
									Line:   100,
									Column: 94,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   100,
								Column: 82,
							},
							EndPosition: &source.Position{
								Line:   100,
								Column: 94,
							},
						},
					},
					{
						Name: "label",
						Value: &Substitution{
							StringValue: &arg3,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   100,
									Column: 104,
								},
								EndPosition: &source.Position{
									Line:   100,
									Column: 111,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   100,
								Column: 96,
							},
							EndPosition: &source.Position{
								Line:   100,
								Column: 111,
							},
						},
					},
				},
				Path: []*SubstitutionPathItem{},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   100,
						Column: 54,
					},
					EndPosition: &source.Position{
						Line:   100,
						Column: 112,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   100,
					Column: 54,
				},
				EndPosition: &source.Position{
					Line:   100,
					Column: 112,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   100,
				Column: 50,
			},
			EndPosition: &source.Position{
				Line:   100,
				Column: 116,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_sub_string_with_a_func_call_5(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${  extract_config(resources.network.spec)[0]["details"].id  }`,
		// Emulate this substitution starting on line 100, column 50.
		// Source meta values of substitution components are offset from the start
		// of the input string.
		&source.Meta{Position: source.Position{
			Line:   100,
			Column: 50,
		}},
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	rootIndex := int64(0)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			Function: &SubstitutionFunctionExpr{
				FunctionName: "extract_config",
				Arguments: []*SubstitutionFunctionArg{
					{
						Value: &Substitution{
							ResourceProperty: &SubstitutionResourceProperty{
								ResourceName: "network",
								Path: []*SubstitutionPathItem{
									{
										FieldName: "spec",
									},
								},
								SourceMeta: &source.Meta{
									Position: source.Position{
										Line:   100,
										Column: 69,
									},
									EndPosition: &source.Position{
										Line:   100,
										Column: 91,
									},
								},
							},
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   100,
									Column: 69,
								},
								EndPosition: &source.Position{
									Line:   100,
									Column: 91,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   100,
								Column: 69,
							},
							EndPosition: &source.Position{
								Line:   100,
								Column: 91,
							},
						},
					},
				},
				Path: []*SubstitutionPathItem{
					{
						ArrayIndex: &rootIndex,
					},
					{
						FieldName: "details",
					},
					{
						FieldName: "id",
					},
				},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   100,
						Column: 54,
					},
					EndPosition: &source.Position{
						Line:   100,
						Column: 109,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   100,
					Column: 54,
				},
				EndPosition: &source.Position{
					Line:   100,
					Column: 109,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   100,
				Column: 50,
			},
			EndPosition: &source.Position{
				Line:   100,
				Column: 112,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_correctly_parses_a_sub_string_with_a_func_call_6(c *C) {
	parsed, err := ParseSubstitutionValues(
		"",
		`${  object(total = -25, avg = -11, label = "Label")   }`,
		// Emulate this substitution starting on line 100, column 50.
		// Source meta values of substitution components are offset from the start
		// of the input string.
		&source.Meta{Position: source.Position{
			Line:   100,
			Column: 50,
		}},
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	arg1 := int64(-25)
	arg2 := int64(-11)
	arg3 := "Label"
	c.Assert(err, IsNil)
	c.Assert(len(parsed), Equals, 1)
	c.Assert(parsed[0], DeepEquals, &StringOrSubstitution{
		SubstitutionValue: &Substitution{
			Function: &SubstitutionFunctionExpr{
				FunctionName: "object",
				Arguments: []*SubstitutionFunctionArg{
					{
						Name: "total",
						Value: &Substitution{
							IntValue: &arg1,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   100,
									Column: 69,
								},
								EndPosition: &source.Position{
									Line:   100,
									Column: 72,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   100,
								Column: 61,
							},
							EndPosition: &source.Position{
								Line:   100,
								Column: 72,
							},
						},
					},
					{
						Name: "avg",
						Value: &Substitution{
							IntValue: &arg2,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   100,
									Column: 80,
								},
								EndPosition: &source.Position{
									Line:   100,
									Column: 83,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   100,
								Column: 74,
							},
							EndPosition: &source.Position{
								Line:   100,
								Column: 83,
							},
						},
					},
					{
						Name: "label",
						Value: &Substitution{
							StringValue: &arg3,
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   100,
									Column: 93,
								},
								EndPosition: &source.Position{
									Line:   100,
									Column: 100,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   100,
								Column: 85,
							},
							EndPosition: &source.Position{
								Line:   100,
								Column: 100,
							},
						},
					},
				},
				Path: []*SubstitutionPathItem{},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   100,
						Column: 54,
					},
					EndPosition: &source.Position{
						Line:   100,
						Column: 101,
					},
				},
			},
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   100,
					Column: 54,
				},
				EndPosition: &source.Position{
					Line:   100,
					Column: 101,
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   100,
				Column: 50,
			},
			EndPosition: &source.Position{
				Line:   100,
				Column: 105,
			},
		},
	})
}

func (s *ParseSubstitutionsTestSuite) Test_fails_to_parse_susbstitution_reporting_correct_position(c *C) {
	_, err := ParseSubstitutionValues(
		"",
		// hex numbers are not supported in the substitution language.
		`${  format(0x23)   }`,
		// Emulate this substitution starting on line 100, column 50.
		// Source meta values of substitution components are offset from the start
		// of the input string.
		&source.Meta{Position: source.Position{
			Line:   100,
			Column: 50,
		}},
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	c.Assert(err, NotNil)

	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	// Top-level error corresponds to the outer start point
	// of the substitution (the location of the "${").
	c.Assert(*loadErr.Line, Equals, 100)
	c.Assert(*loadErr.Column, Equals, 50)

	parseErrs, isParseErrs := loadErr.ChildErrors[0].(*ParseErrors)
	c.Assert(isParseErrs, Equals, true)
	c.Assert(parseErrs.ChildErrors, HasLen, 1)

	parseErr, isParseErr := parseErrs.ChildErrors[0].(*ParseError)
	c.Assert(isParseErr, Equals, true)
	// The parse error corresponds to the "x" in "0x23"
	// which is not expected after the "0".
	c.Assert(parseErr.Line, Equals, 100)
	c.Assert(parseErr.Column, Equals, 62)
	c.Assert(
		parseErr.Error(),
		Equals,
		"parse error at column 62 with token type identifier: "+
			"expected a comma after function argument 0",
	)
}

func (s *ParseSubstitutionsTestSuite) Test_fails_to_parse_susbstitution_reporting_correct_position_for_lex_error(c *C) {
	_, err := ParseSubstitutionValues(
		"",
		// "!" is an unexpected punctuation mark in the substitution language,
		// this should lead to a lex error.
		`${  "start of string literal"!  }`,
		// Emulate this substitution starting on line 150, column 70.
		// Source meta values of substitution components are offset from the start
		// of the input string.
		&source.Meta{Position: source.Position{
			Line:   150,
			Column: 70,
		}},
		true,
		false,
		/* parentContextPrecedingCharCount */ 0,
	)
	c.Assert(err, NotNil)

	loadErr, isLoadErr := err.(*errors.LoadError)
	c.Assert(isLoadErr, Equals, true)
	// Top-level error corresponds to the outer start point
	// of the substitution (the location of the "${").
	c.Assert(*loadErr.Line, Equals, 150)
	c.Assert(*loadErr.Column, Equals, 70)

	lexErrs, isLexErrs := loadErr.ChildErrors[0].(*LexErrors)
	c.Assert(isLexErrs, Equals, true)
	c.Assert(lexErrs.ChildErrors, HasLen, 1)

	lexErr, isLexErr := lexErrs.ChildErrors[0].(*LexError)
	c.Assert(isLexErr, Equals, true)
	// The lex error corresponds to the "!" after the string literal.
	c.Assert(lexErr.Line, Equals, 150)
	c.Assert(lexErr.Column, Equals, 99)
	c.Assert(
		lexErr.Error(),
		Equals,
		"lex error at column 99: validation failed due to an unexpected"+
			" character \"!\" having been encountered in a reference substitution",
	)
}
