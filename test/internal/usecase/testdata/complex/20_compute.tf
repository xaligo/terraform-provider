data "aws_ami" "linux" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }
}

resource "aws_instance" "web" {
  for_each = local.web_nodes

  ami                    = data.aws_ami.linux.id
  instance_type          = each.value
  subnet_id              = aws_subnet.public_a.id
  vpc_security_group_ids = [aws_security_group.application.id]

  tags = {
    Name = "web-fleet"
  }
}

resource "aws_instance" "worker" {
  count = 2

  ami                    = data.aws_ami.linux.id
  instance_type          = "t3.micro"
  subnet_id              = aws_subnet.private_a.id
  vpc_security_group_ids = [aws_security_group.application.id]

  tags = {
    Name = "worker-fleet"
  }
}

resource "aws_lb" "application" {
  name               = "complex-application"
  load_balancer_type = "application"
  security_groups    = [aws_security_group.application.id]
  subnets = [
    aws_subnet.public_a.id,
    aws_subnet.public_b.id,
  ]
}

resource "aws_lb_target_group" "application" {
  name     = "complex-application"
  port     = 8080
  protocol = "HTTP"
  vpc_id   = aws_vpc.main.id
}

resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.application.arn
  port              = 443
  protocol          = "HTTPS"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.application.arn
  }
}

resource "aws_db_subnet_group" "main" {
  name = "complex-main"
  subnet_ids = [
    aws_subnet.private_a.id,
    aws_subnet.private_b.id,
  ]
}

resource "aws_db_instance" "main" {
  identifier             = "complex-main"
  engine                 = "postgres"
  instance_class         = "db.t4g.micro"
  allocated_storage      = 20
  username               = "example"
  password               = var.database_password
  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.application.id]
  skip_final_snapshot    = true
}
