package auditv1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

// Saturday, 3 May 2025 14:27:22 UTC
var paginationTestTime = time.Unix(1746282442, 0).UTC()

// Whether a caller can tell a page of the trail from the whole answer.
//
// A page mistaken for the whole answer is what this protects against.
// A caller asking what a principal did yesterday, getting the default page, and concluding that
// was all of it would be incorrect.
type PaginationSuite struct {
	suite.Suite
	router *mux.Router
}

func (s *PaginationSuite) withRecords(count int) {
	records := map[string]*manage.AuditRecord{}
	for i := 1; i <= count; i++ {
		id := fmt.Sprintf("record-%03d", i)
		records[id] = &manage.AuditRecord{
			ID:               id,
			Timestamp:        int64(1000 + i),
			Action:           "instance.create",
			Outcome:          manage.OutcomeSuccess,
			StatusCode:       200,
			Method:           "POST",
			Path:             "/deployments/instances",
			PrincipalMethod:  "apikey",
			PrincipalSubject: "user-a",
		}
	}

	controller := NewController(10*time.Second, &typesv1.Dependencies{
		AuditStore: testutils.NewMockAuditStore(records),
		CleanupOperationsStore: testutils.NewMockCleanupOperationsStore(
			map[string]*manage.CleanupOperation{},
		),
		IDGenerator: core.NewUUIDGenerator(),
		Clock:       &testutils.MockClock{StaticTime: paginationTestTime},
		Logger:      core.NewNopLogger(),
	})

	s.router = mux.NewRouter()
	s.router.HandleFunc("/audit/records", controller.QueryAuditRecordsHandler).Methods("GET")
}

func (s *PaginationSuite) Test_reports_more_when_the_trail_holds_more_than_the_page() {
	s.withRecords(5)
	page := s.query("/audit/records?limit=2")

	s.Assert().Equal([]string{"record-001", "record-002"}, recordIDs(page.Records))
	s.Assert().True(page.HasMore)
	s.Require().NotNil(page.NextCursor)
	s.Assert().Equal(int64(1002), page.NextCursor.AfterTimestamp)
	s.Assert().Equal("record-002", page.NextCursor.AfterID)
}

func (s *PaginationSuite) Test_reports_no_more_when_the_page_covers_all_available_results() {
	s.withRecords(2)
	page := s.query("/audit/records?limit=2")

	s.Assert().Len(page.Records, 2)
	s.Assert().False(page.HasMore, "a full page that is also the end must say so")
	s.Assert().Nil(page.NextCursor)
}

func (s *PaginationSuite) Test_reports_no_more_when_the_page_is_short() {
	s.withRecords(1)
	page := s.query("/audit/records?limit=2")

	s.Assert().Len(page.Records, 1)
	s.Assert().False(page.HasMore)
	s.Assert().Nil(page.NextCursor)
}

func (s *PaginationSuite) Test_reports_no_more_for_an_empty_trail() {
	s.withRecords(0)
	page := s.query("/audit/records?limit=2")

	s.Assert().Empty(page.Records)
	s.Assert().False(page.HasMore)
	s.Assert().Nil(page.NextCursor)
}

// Paging the whole trail with the supplied cursor has to return every record once, which
// is what a forwarder shipping the trail onwards depends on.
func (s *PaginationSuite) Test_the_cursor_pages_the_whole_trail_exactly_once() {
	s.withRecords(7)

	seen := []string{}
	target := "/audit/records?limit=3"
	for range 10 {
		page := s.query(target)
		seen = append(seen, recordIDs(page.Records)...)
		if !page.HasMore {
			s.Assert().Nil(page.NextCursor)
			break
		}
		s.Require().NotNil(page.NextCursor)
		target = fmt.Sprintf(
			"/audit/records?limit=3&afterTimestamp=%d&afterId=%s",
			page.NextCursor.AfterTimestamp,
			page.NextCursor.AfterID,
		)
	}

	expected := []string{}
	for i := 1; i <= 7; i++ {
		expected = append(expected, fmt.Sprintf("record-%03d", i))
	}
	s.Assert().Equal(expected, seen)
}

// The page the caller gets when it asks for no particular size is still a page, and has
// to say so.
func (s *PaginationSuite) Test_the_default_page_reports_more() {
	s.withRecords(manage.DefaultAuditQueryLimit + 1)
	page := s.query("/audit/records")

	s.Assert().Len(page.Records, manage.DefaultAuditQueryLimit)
	s.Assert().True(page.HasMore, "the default page size must not look like the whole trail")
}

func (s *PaginationSuite) query(target string) *QueryAuditRecordsResponse {
	request := httptest.NewRequest("GET", target, nil)
	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, request)

	result := recorder.Result()
	defer result.Body.Close()
	s.Require().Equal(http.StatusOK, result.StatusCode)

	data, err := io.ReadAll(result.Body)
	s.Require().NoError(err)
	page := &QueryAuditRecordsResponse{}
	s.Require().NoError(json.Unmarshal(data, page))

	return page
}

func TestPaginationSuite(t *testing.T) {
	suite.Run(t, new(PaginationSuite))
}
