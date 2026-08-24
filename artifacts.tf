# Build artifacts for Lambda code signing

resource "aws_s3_bucket" "artifacts" {
  #checkov:skip=CKV_AWS_18: build output already published as a release asset
  #checkov:skip=CKV_AWS_144: rebuildable from a release
  #checkov:skip=CKV2_AWS_62: nothing consumes bucket events
  bucket = local.aws_s3_artifacts_bucket_name

  # Rebuildable content, so a leftover object shouldn't wedge terraform destroy
  force_destroy = true
}

# Required by Signer: a signing job names its source object by version
resource "aws_s3_bucket_versioning" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_policy" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "DenyInsecureTransport"
      Effect    = "Deny"
      Principal = "*"
      Action    = "s3:*"
      Resource = [
        aws_s3_bucket.artifacts.arn,
        "${aws_s3_bucket.artifacts.arn}/*",
      ]
      Condition = {
        Bool = { "aws:SecureTransport" = "false" }
      }
    }]
  })

  depends_on = [aws_s3_bucket_public_access_block.artifacts]
}

# Signed objects are job-named, so they accumulate instead of overwriting.
# Lambda copies a package at deploy time, so expiry only costs a re-sign.
resource "aws_s3_bucket_lifecycle_configuration" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id

  rule {
    id     = "abort-incomplete-uploads"
    status = "Enabled"

    filter {}

    abort_incomplete_multipart_upload {
      days_after_initiation = 1
    }
  }

  rule {
    id     = "expire-superseded-unsigned"
    status = "Enabled"

    filter {
      prefix = "unsigned/"
    }

    noncurrent_version_expiration {
      noncurrent_days = 1
    }
  }

  rule {
    id     = "expire-old-signed"
    status = "Enabled"

    filter {
      prefix = "signed/"
    }

    expiration {
      days = 1
    }

    noncurrent_version_expiration {
      noncurrent_days = 1
    }
  }

  depends_on = [aws_s3_bucket_versioning.artifacts]
}

# Uploaded before signing because Signer only reads sources from S3. The
# depends_on is load-bearing: without versioning on, version_id below is null.
resource "aws_s3_object" "unsigned" {
  for_each = local.lambda_packages

  bucket      = aws_s3_bucket.artifacts.id
  key         = "unsigned/${each.key}.zip"
  source      = "bin/${each.key}.zip"
  source_hash = filemd5("bin/${each.key}.zip")

  depends_on = [aws_s3_bucket_versioning.artifacts]
}

resource "aws_signer_signing_job" "lambda" {
  for_each = local.lambda_packages

  profile_name = aws_signer_signing_profile.lambda_signing_profile.name

  source {
    s3 {
      bucket  = aws_s3_bucket.artifacts.id
      key     = aws_s3_object.unsigned[each.key].key
      version = aws_s3_object.unsigned[each.key].version_id
    }
  }

  destination {
    s3 {
      bucket = aws_s3_bucket.artifacts.id
      prefix = "signed/${each.key}-"
    }
  }
}
