# Build artifacts for Lambda code signing. AWS Signer only signs S3 objects, so
# deployment packages are uploaded here, signed in place, and deployed from the
# signed copy.
#
# Terraform owns this bucket, unlike the state and email buckets: nothing has to
# exist before init, and everything in it is rebuildable from a release. The
# email bucket can't double as this one - Signer reads its source by version, so
# the whole bucket has to be versioned, and emails are stored and deleted by
# message ID at the root, where versioning would retain every deleted message.

resource "aws_s3_bucket" "artifacts" {
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

# SSE-S3, not a CMK: Signer reads the source object on the caller's behalf, and
# a customer key would need a grant for it in return for control this project
# doesn't need over artifacts that are public code anyway.
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

# Signed objects are named after the signing job that produced them, so they
# accumulate instead of overwriting. Lambda copies a package at deploy time and
# never reads it again, so expiry only costs a re-sign on the way back to an
# older build.
resource "aws_s3_bucket_lifecycle_configuration" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id

  rule {
    id     = "abort-incomplete-uploads"
    status = "Enabled"

    filter {}

    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }

  rule {
    id     = "expire-superseded-unsigned"
    status = "Enabled"

    filter {
      prefix = "unsigned/"
    }

    noncurrent_version_expiration {
      noncurrent_days = 30
    }
  }

  rule {
    id     = "expire-old-signed"
    status = "Enabled"

    filter {
      prefix = "signed/"
    }

    expiration {
      days = 90
    }
  }

  # Expiring noncurrent versions needs versioning to be on first
  depends_on = [aws_s3_bucket_versioning.artifacts]
}
