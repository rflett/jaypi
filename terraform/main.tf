resource "aws_dynamodb_table" "jaypi" {
  name           = "jaypi"
  billing_mode   = "PROVISIONED"
  read_capacity  = 1
  write_capacity = 1
  hash_key       = "PK"
  range_key      = "SK"

  attribute {
    name = "PK"
    type = "S"
  }

  attribute {
    name = "SK"
    type = "S"
  }

  global_secondary_index {
    name            = "GSI1"
    hash_key        = "SK"
    range_key       = "PK"
    projection_type = "ALL"
    write_capacity  = 1
    read_capacity   = 1
  }
}

resource "aws_iam_role" "jaypi" {
  name = "lambda-jaypi"

  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [{
      Action = "sts:AssumeRole",
      Principal = {
        Service = "lambda.amazonaws.com"
      },
      Effect = "Allow"
    }]
  })
}

resource "aws_iam_role_policy" "jaypi" {
  role = aws_iam_role.jaypi.id

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:UpdateItem",
          "dynamodb:DeleteItem",
          "dynamodb:Scan",
          "dynamodb:Query",
          "sqs:SendMessage",
          "sqs:SendMessageBatch",
          "sqs:DeleteMessage",
          "sqs:ReceiveMessage",
          "sqs:GetQueueAttributes"
        ],
        Resource = [
          aws_sqs_queue.chune_refresh.arn,
          aws_sqs_queue.bean_counter.arn,
          aws_sqs_queue.scorer.arn,
          aws_dynamodb_table.jaypi.arn,
          "${aws_dynamodb_table.jaypi.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "logs:*"
        ],
        Resource = [
          "*"
        ]
      }
    ]
  })
}

resource "aws_sqs_queue" "chune_refresh" {
  name                       = "chune-refresh-${terraform.workspace}"
  delay_seconds              = 0
  visibility_timeout_seconds = 30
}

resource "aws_sqs_queue" "bean_counter" {
  name                       = "bean-counter-${terraform.workspace}"
  delay_seconds              = 0
  visibility_timeout_seconds = 30
}

resource "aws_sqs_queue" "scorer" {
  name                       = "scorer-${terraform.workspace}"
  delay_seconds              = 0
  visibility_timeout_seconds = 30
}
