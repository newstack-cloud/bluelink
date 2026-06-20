package lang_test

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/lang"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type EmitSuite struct {
	suite.Suite
}

func (s *EmitSuite) requireRoundTrip(src string) {
	first, err := lang.ParseString(src)
	s.Require().NoError(err)
	s.requireRoundTripFrom(first, src)
}

func (s *EmitSuite) requireRoundTripFrom(first *schema.Blueprint, src string) {
	emitted, err := lang.Emit(first)
	s.Require().NoErrorf(err, "emitting:\n%s", src)

	second, err := lang.ParseString(emitted)
	s.Require().NoErrorf(err, "re-parsing emitted source:\n%s", emitted)

	s.Require().Equalf(
		s.yamlString(first),
		s.yamlString(second),
		"round trip mismatch.\nemitted:\n%s",
		emitted,
	)
}

func (s *EmitSuite) yamlString(v any) string {
	data, err := yaml.Marshal(v)
	s.Require().NoError(err)
	return string(data)
}

func (s *EmitSuite) Test_emits_basic_resource() {
	s.requireRoundTrip(`version "2025-11-02"

resource myQueue: aws/sqs/queue {
    metadata {
        displayName = "My Basic SQS Queue"
    }
    spec {
        queueName = "my-basic-queue"
    }
}
`)
}

func (s *EmitSuite) Test_emits_nested_objects_arrays_and_scalars() {
	s.requireRoundTrip(`version "2025-11-02"

resource completeQueue: aws/sqs/queue {
    metadata {
        displayName = "Complete SQS Queue"
        labels = {
            team = "platform"
        }
    }
    spec {
        queueName = "complete-queue"
        delaySeconds = 30
        fifoQueue = false
        redrivePolicy = {
            deadLetterTargetArn = "arn:aws:sqs:us-west-2:123456789012:my-dlq",
            maxReceiveCount = 5
        }
        tags = [
            {
                key = "Environment",
                value = "Production"
            },
            {
                key = "Owner",
                value = "admin@example.com"
            }
        ]
        sourceQueueArns = [
            "arn:aws:sqs:us-west-2:123456789012:source-1",
            "arn:aws:sqs:us-west-2:123456789012:source-2"
        ]
    }
}
`)
}

func TestEmitSuite(t *testing.T) {
	suite.Run(t, new(EmitSuite))
}
