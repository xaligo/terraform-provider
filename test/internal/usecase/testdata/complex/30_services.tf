resource "aws_s3_bucket" "artifacts" {
  bucket = "xaligo-complex-test-artifacts"
}

resource "aws_sqs_queue" "jobs" {
  name = "complex-jobs"
}

resource "aws_iam_role" "worker" {
  name = "complex-worker"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })
}

resource "aws_lambda_function" "worker" {
  function_name = "complex-worker"
  role          = aws_iam_role.worker.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  s3_bucket     = aws_s3_bucket.artifacts.id
  s3_key        = "worker.zip"

  environment {
    variables = {
      QUEUE_URL = aws_sqs_queue.jobs.url
    }
  }
}

resource "aws_lambda_event_source_mapping" "jobs" {
  event_source_arn = aws_sqs_queue.jobs.arn
  function_name    = aws_lambda_function.worker.arn
}

resource "aws_cloudwatch_log_group" "worker" {
  name              = "/aws/lambda/${aws_lambda_function.worker.function_name}"
  retention_in_days = 14
}

module "observability" {
  source = "./modules/observability"

  log_group_name = aws_cloudwatch_log_group.worker.name
  queue_arn      = aws_sqs_queue.jobs.arn
}

resource "terraform_data" "release" {
  triggers_replace = [
    aws_lb.application.arn,
    aws_db_instance.main.id,
    module.observability.id,
  ]
}
