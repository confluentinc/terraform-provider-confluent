# https://docs.confluent.io/cloud/current/connectors/provider-integration/index.html
resource "aws_iam_role" "s3_access_role" {
  name        = var.customer_role_name
  description = "IAM role for accessing S3 with a trust policy for Confluent Tableflow"

  assume_role_policy = jsonencode({
    Version   = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = {
          AWS = var.provider_integration_role_arn
        }
        Action    = "sts:AssumeRole"
        Condition = {
          StringEquals = {
            "sts:ExternalId" = var.provider_integration_external_id
          }
        }
      },
      {
        Effect    = "Allow"
        Principal = {
          AWS = var.provider_integration_role_arn
        }
        Action    = "sts:TagSession"
      }
    ]
  })
}

# https://docs.confluent.io/cloud/current/connectors/cc-s3-sink/cc-s3-sink.html#user-account-iam-policy
resource "aws_iam_policy" "s3_access_policy" {
  name        = "TableflowS3AccessPolicy"
  description = "IAM policy for accessing the S3 bucket for Confluent Tableflow"

  policy = jsonencode({
    Version   = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetBucketLocation",
          "s3:ListBucketMultipartUploads",
          "s3:ListBucket"
        ]
        Resource = "arn:aws:s3:::${var.s3_bucket_name}"
      },
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:PutObjectTagging",
          "s3:GetObject",
          "s3:AbortMultipartUpload",
          "s3:ListMultipartUploadParts"
        ]
        Resource = "arn:aws:s3:::${var.s3_bucket_name}/*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "s3_policy_attachment" {
  role       = aws_iam_role.s3_access_role.name
  policy_arn = aws_iam_policy.s3_access_policy.arn
}
