package schema

import (
	"os"

	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	. "gopkg.in/check.v1"
)

type TreeTestSuite struct {
	specFixtures map[string]fixture
}

var _ = Suite(&TreeTestSuite{})

func (s *TreeTestSuite) SetUpSuite(c *C) {
	s.specFixtures = make(map[string]fixture)
	fixturesToLoad := map[string]string{
		"partial": "__testdata/tree/blueprint-partial.yml",
		"full":    "__testdata/tree/blueprint-full.yml",
	}

	for name, filePath := range fixturesToLoad {
		specBytes, err := os.ReadFile(filePath)
		if err != nil {
			c.Error(err)
			c.FailNow()
		}
		s.specFixtures[name] = fixture{
			filePath:  filePath,
			stringVal: string(specBytes),
		}
	}
}

func (s *TreeTestSuite) Test_generates_tree_from_partial_blueprint(c *C) {
	// Partial meaning a subset of elements in the blueprint are populated.
	blueprint, err := Load(s.specFixtures["partial"].filePath, YAMLSpecFormat)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	tree := SchemaToTree(blueprint)
	if tree == nil {
		c.Error("SchemaToTree returned nil")
		c.FailNow()
	}

	err = testhelpers.Snapshot(tree)
	if err != nil {
		c.Error(err)
	}
}

func (s *TreeTestSuite) Test_generates_tree_from_full_blueprint(c *C) {
	blueprint, err := Load(s.specFixtures["full"].filePath, YAMLSpecFormat)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	tree := SchemaToTree(blueprint)
	if tree == nil {
		c.Error("SchemaToTree returned nil")
		c.FailNow()
	}

	err = testhelpers.Snapshot(tree)
	if err != nil {
		c.Error(err)
	}
}
