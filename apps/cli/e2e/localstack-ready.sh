#!/bin/sh
# Runs inside LocalStack once S3 is ready: uploads every fixture to the bucket the e2e
# scripts read blueprints from (s3://test-blueprints/...).
set -e
awslocal s3 mb s3://test-blueprints
awslocal s3 cp /fixtures/ s3://test-blueprints/ --recursive
echo "LocalStack setup complete"
