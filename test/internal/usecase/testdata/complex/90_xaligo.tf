variable "diagram_layout" {
  type = object({
    application = list(string)
    monitoring  = list(string)
    queue       = list(string)
  })
  default = {
    application = ["aws_lambda_function.worker", "aws_s3_bucket.artifacts"]
    monitoring  = ["aws_cloudwatch_log_group.worker"]
    queue       = ["aws_sqs_queue.jobs"]
  }
}

data "xaligo_items" "complex" {
  source_dir = abspath(path.module)
}

locals {
  items = merge(data.xaligo_items.complex.items, {
    application_services = "application-services"
    aws_cloud             = "xaligo-aws-cloud"
  })
  application_items = [for address in var.diagram_layout.application : local.items[address]]
  monitoring_items  = [for address in var.diagram_layout.monitoring : local.items[address]]
  queue_items       = [for address in var.diagram_layout.queue : local.items[address]]
}

resource "xaligo_diagram" "complex" {
  source_dir  = abspath(path.module)
  output_path = abspath("${path.module}/generated.xal")
  frame_id    = "complex"
  title       = "Complex AWS Architecture"

  paper_size  = "A3"
  orientation = "landscape"
  grid_gap    = 20

  container {
    id = "application-services"
    items    = local.application_items
    layout   = "horizontal"
    gap      = 0
    align    = "middle-spread"
    overflow = "visible"
  }

  row {
    gap      = 20
    overflow = "visible"

    col {
      span  = 3
      class = "pa-1"
      items = local.monitoring_items
    }
    col {
      span  = 6
      items = [local.items.application_services]
    }
    col {
      span  = 3
      align = "middle-center"
      items = local.queue_items
    }
  }

  layout {
    item     = local.items.aws_cloud
    align    = "top-left"
    overflow = "visible"
    row      = 2
  }
}
