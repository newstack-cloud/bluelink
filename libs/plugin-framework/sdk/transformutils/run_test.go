package transformutils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

type RunHelpersTestSuite struct {
	suite.Suite
}

type buildManifest struct {
	Region string
}

type modulePath string
type assetsPath string

func (s *RunHelpersTestSuite) Test_Use_returns_zero_value_and_false_when_value_not_provided() {
	run := &Run{}

	manifest, ok := Use[*buildManifest](run)

	s.Assert().Nil(manifest)
	s.Assert().False(ok)
}

func (s *RunHelpersTestSuite) Test_Provide_and_Use_round_trip_a_typed_value() {
	run := &Run{}
	manifest := &buildManifest{Region: "eu-west-1"}

	Provide(run, manifest)
	got, ok := Use[*buildManifest](run)

	s.Require().True(ok)
	s.Assert().Same(manifest, got)
}

func (s *RunHelpersTestSuite) Test_Provide_overwrites_an_earlier_value_of_the_same_type() {
	run := &Run{}

	Provide(run, &buildManifest{Region: "us-east-1"})
	Provide(run, &buildManifest{Region: "ap-south-1"})

	got, ok := Use[*buildManifest](run)
	s.Require().True(ok)
	s.Assert().Equal("ap-south-1", got.Region)
}

func (s *RunHelpersTestSuite) Test_newtype_wrappers_disambiguate_values_with_same_underlying_type() {
	run := &Run{}

	Provide(run, modulePath("/modules"))
	Provide(run, assetsPath("/assets"))

	gotModule, moduleOK := Use[modulePath](run)
	gotAssets, assetsOK := Use[assetsPath](run)

	s.Require().True(moduleOK)
	s.Require().True(assetsOK)
	s.Assert().Equal(modulePath("/modules"), gotModule)
	s.Assert().Equal(assetsPath("/assets"), gotAssets)
}

func (s *RunHelpersTestSuite) Test_MustUse_returns_value_when_one_has_been_provided() {
	run := &Run{}
	Provide(run, modulePath("/modules"))

	s.Assert().Equal(modulePath("/modules"), MustUse[modulePath](run))
}

func (s *RunHelpersTestSuite) Test_MustUse_panics_with_descriptive_message_when_value_missing() {
	run := &Run{}

	defer func() {
		recovered := recover()
		s.Require().NotNil(recovered, "expected MustUse to panic")
		msg, ok := recovered.(string)
		s.Require().True(ok, "expected string panic value")
		s.Assert().Contains(msg, "MustUse")
		s.Assert().Contains(msg, "modulePath")
	}()

	_ = MustUse[modulePath](run)
}

func (s *RunHelpersTestSuite) Test_concurrent_Provide_and_Use_calls_do_not_race() {
	run := &Run{}
	const iterations = 200
	var wg sync.WaitGroup

	for range iterations {
		wg.Add(2)
		go func() {
			defer wg.Done()
			Provide(run, modulePath("/x"))
		}()
		go func() {
			defer wg.Done()
			_, _ = Use[modulePath](run)
		}()
	}
	wg.Wait()

	got, ok := Use[modulePath](run)
	s.Require().True(ok)
	s.Assert().Equal(modulePath("/x"), got)
}

func TestRunHelpersTestSuite(t *testing.T) {
	suite.Run(t, new(RunHelpersTestSuite))
}
