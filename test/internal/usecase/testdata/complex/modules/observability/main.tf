variable "log_group_name" {
  type = string
}

variable "queue_arn" {
  type = string
}

resource "aws_cloudwatch_metric_alarm" "queue_depth" {
  alarm_name          = "complex-queue-depth"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  period              = 60
  statistic           = "Average"
  threshold           = 10

  dimensions = {
    QueueArn = var.queue_arn
  }
}

output "id" {
  value = aws_cloudwatch_metric_alarm.queue_depth.id
}
