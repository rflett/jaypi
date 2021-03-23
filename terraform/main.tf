data "aws_caller_identity" "current" {}

data "aws_secretsmanager_secret" "jwt_signing_key" {
  name = "jaypi-private-key-${var.environment}"
}

data "aws_ssm_parameter" "gcm_token" {
  name            = "/${var.environment}/sns/fcm-delegator-countdown-test-token"
  with_decryption = true
}

data "aws_ssm_parameter" "apn_cert" {
  name            = "/${var.environment}/sns/apple-apn-notifications-cert"
  with_decryption = true
}

data "aws_ssm_parameter" "apn_key" {
  name            = "/${var.environment}/sns/apple-apn-notifications-key"
  with_decryption = true
}

data "aws_route53_zone" "jaypi" {
  name = var.environment == "production" ? "jaypi.online." : "${var.environment}.jaypi.online."
}

resource "aws_dynamodb_table" "jaypi" {
  name           = "jaypi-${var.environment}"
  billing_mode   = "PROVISIONED"
  hash_key       = "PK"
  range_key      = "SK"
  read_capacity  = 1
  write_capacity = 1

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

  tags = {
    Environment = var.environment
  }
}

resource "aws_iam_role" "jaypi" {
  name = "lambda-jaypi-${var.environment}"

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
  name = aws_iam_role.jaypi.name
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
          "secretsmanager:GetSecretValue",
          "s3:PutObject"
        ],
        Resource = [
          aws_sqs_queue.chune_refresh.arn,
          aws_sqs_queue.bean_counter.arn,
          aws_sqs_queue.scorer.arn,
          aws_sqs_queue.town_crier.arn,
          aws_sqs_queue.town_crier_dlq.arn,
          aws_dynamodb_table.jaypi.arn,
          "${aws_dynamodb_table.jaypi.arn}/*",
          "${aws_s3_bucket.assets.arn}/*",
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
  name                       = "chune-refresh-${var.environment}"
  delay_seconds              = 0
  visibility_timeout_seconds = 30

  tags = {
    Environment = var.environment
    Component   = "music"
  }
}

resource "aws_sqs_queue" "bean_counter" {
  name                       = "bean-counter-${var.environment}"
  delay_seconds              = 0
  visibility_timeout_seconds = 30

  tags = {
    Environment = var.environment
    Component   = "scoring"
  }
}

resource "aws_sqs_queue" "scorer" {
  name                       = "scorer-${var.environment}"
  delay_seconds              = 0
  visibility_timeout_seconds = 30

  tags = {
    Environment = var.environment
    Component   = "scoring"
  }
}

resource "aws_sqs_queue" "town_crier_dlq" {
  name                       = "town-crier-dlq-${var.environment}"
  delay_seconds              = 0
  visibility_timeout_seconds = 30
  message_retention_seconds  = 604800
  receive_wait_time_seconds  = 20

  tags = {
    Environment = var.environment
    Component   = "notifications"
  }
}

resource "aws_sqs_queue" "town_crier" {
  name                       = "town-crier-${var.environment}"
  delay_seconds              = 0
  visibility_timeout_seconds = 30

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.town_crier_dlq.arn
    maxReceiveCount     = 1
  })

  tags = {
    Environment = var.environment
    Component   = "notifications"
  }
}

resource "aws_sns_platform_application" "gcm_application" {
  name                         = "google-fcm-notifications-${var.environment}"
  platform                     = "GCM"
  success_feedback_sample_rate = 100
  platform_credential          = data.aws_ssm_parameter.gcm_token.value
}

resource "aws_sns_platform_application" "apn_application" {
  name                         = "apple-apn-notifications-${var.environment}"
  success_feedback_sample_rate = 100
  platform                     = var.environment == "production" ? "APNS" : "APNS_SANDBOX"
  platform_credential          = base64decode(data.aws_ssm_parameter.apn_key.value)
  platform_principal           = base64decode(data.aws_ssm_parameter.apn_cert.value)
}

resource "aws_cloudfront_origin_access_identity" "main" {
  comment = "jaypi-${var.environment}"
}

resource "aws_s3_bucket" "assets" {
  bucket = "jaypi-assets-${var.environment}"
  acl    = "private"
}

resource "aws_s3_bucket_public_access_block" "assets" {
  bucket                  = aws_s3_bucket.assets.bucket
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_policy" "assets" {
  bucket = aws_s3_bucket.assets.bucket
  policy = jsonencode({
    Version = "2008-10-17",
    Statement = [{
      Action   = "s3:GetObject",
      Effect   = "Allow",
      Resource = "${aws_s3_bucket.assets.arn}/*",
      Principal = {
        AWS = aws_cloudfront_origin_access_identity.main.iam_arn
      }
    }]
  })
}

resource "aws_acm_certificate" "assets" {
  provider = aws.north_virginia

  domain_name       = var.environment == "production" ? "assets.jaypi.online" : "assets.${var.environment}.jaypi.online"
  validation_method = "DNS"

  tags = {
    Environment = var.environment
  }
}

resource "aws_route53_record" "assets_validation" {
  for_each = {
    for dvo in aws_acm_certificate.assets.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  }

  allow_overwrite = true
  ttl             = 60
  name            = each.value.name
  type            = each.value.type
  records         = [each.value.record]
  zone_id         = data.aws_route53_zone.jaypi.zone_id
}

resource "aws_acm_certificate_validation" "assets" {
  certificate_arn         = aws_acm_certificate.assets.arn
  validation_record_fqdns = [for record in aws_route53_record.assets_validation : record.fqdn]
}

resource "aws_cloudfront_distribution" "assets" {
  enabled             = true
  comment             = "Web Assets ${var.environment}"
  default_root_object = "index.html"
  price_class         = "PriceClass_All"
  http_version        = "http2"
  is_ipv6_enabled     = true
  aliases             = [aws_s3_bucket.assets.bucket]

  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "S3Origin"
    viewer_protocol_policy = "redirect-to-https"
    max_ttl                = 604800
    min_ttl                = 0
    default_ttl            = 86400
    compress               = true

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }
  }

  origin {
    domain_name = aws_s3_bucket.assets.bucket_domain_name
    origin_id   = "S3Origin"

    s3_origin_config {
      origin_access_identity = aws_cloudfront_origin_access_identity.main.cloudfront_access_identity_path
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.assets.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.1_2016"
  }
}

resource "aws_route53_record" "assets" {
  type    = "A"
  name    = aws_acm_certificate.assets.domain_name
  zone_id = data.aws_route53_zone.jaypi.zone_id

  alias {
    evaluate_target_health = false
    name                   = aws_cloudfront_distribution.assets.domain_name
    zone_id                = aws_cloudfront_distribution.assets.hosted_zone_id
  }
}
