data "aws_caller_identity" "current" {}

data "aws_secretsmanager_secret" "jwt_signing_key" {
  name = "jjj-api-private-signing-key"
}

data "aws_ssm_parameter" "gcm_token" {
  name            = "/${terraform.workspace}/sns/google-fcm-notifications-server-token"
  with_decryption = true
}

data "aws_ssm_parameter" "apn_cert" {
  name            = "/${terraform.workspace}/sns/apple-apn-notifications-cert"
  with_decryption = true
}

data "aws_ssm_parameter" "apn_key" {
  name            = "/${terraform.workspace}/sns/apple-apn-notifications-key"
  with_decryption = true
}

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
          "sns:Publish",
          "sqs:SendMessage",
          "sqs:SendMessageBatch",
          "sqs:DeleteMessage",
          "sqs:ReceiveMessage",
          "sqs:GetQueueAttributes",
          "secretsmanager:GetSecretValue"
        ],
        Resource = [
          aws_sqs_queue.chune_refresh.arn,
          aws_sqs_queue.bean_counter.arn,
          aws_sqs_queue.scorer.arn,
          aws_sqs_queue.town_crier.arn,
          aws_sqs_queue.town_crier_dlq.arn,
          aws_dynamodb_table.jaypi.arn,
          "${aws_dynamodb_table.jaypi.arn}/*",
          data.aws_secretsmanager_secret.jwt_signing_key.arn,
          "arn:aws:sns:ap-southeast-2:${data.aws_caller_identity.current.account_id}:endpoint/*",
          "arn:aws:sns:ap-southeast-2:${data.aws_caller_identity.current.account_id}:app/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "logs:*",
          "sns:SetEndpointAttributes",
          "sns:GetEndpointAttributes",
          "sns:GetPlatformApplicationAttributes",
          "sns:CreatePlatformEndpoint",
          "sns:DeleteEndpoint",
          "sns:ListPlatformApplications",
          "sns:ListEndpointsByPlatformApplication"
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

resource "aws_sqs_queue" "town_crier_dlq" {
  name                       = "town-crier-dlq-${terraform.workspace}"
  delay_seconds              = 0
  visibility_timeout_seconds = 30
  message_retention_seconds  = 604800
  receive_wait_time_seconds  = 20
}

resource "aws_sqs_queue" "town_crier" {
  name                       = "town-crier-${terraform.workspace}"
  delay_seconds              = 0
  visibility_timeout_seconds = 30

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.town_crier_dlq.arn
    maxReceiveCount     = 1
  })
}

resource "aws_sns_platform_application" "gcm_application" {
  name                         = "google-fcm-notifications-${terraform.workspace}"
  platform                     = "GCM"
  success_feedback_sample_rate = 100
  platform_credential          = data.aws_ssm_parameter.gcm_token.value
}

resource "aws_sns_platform_application" "apn_application" {
  name                         = "apple-apn-notifications-${terraform.workspace}"
  platform                     = "APNS"
  success_feedback_sample_rate = 100
  platform_credential          = base64decode(data.aws_ssm_parameter.apn_key.value)
  platform_principal           = base64decode(data.aws_ssm_parameter.apn_cert.value)
}
