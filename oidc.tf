# GitHub Actions OIDC: CI assumes a short-lived role instead of holding a
# long-lived access key. Opt-in - a fresh clone creates nothing.

locals {
  # The OIDC provider is account-global and shared with other projects, so it
  # is created out of band and referenced by ARN.
  oidc_enabled = var.github_repository != "" && var.github_oidc_provider_arn != ""
}

resource "aws_iam_role" "github_plan" {
  count = local.oidc_enabled ? 1 : 0

  name = "${local.project_name_env}-github-plan"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Action    = "sts:AssumeRoleWithWebIdentity"
      Principal = { Federated = var.github_oidc_provider_arn }
      Condition = {
        StringEquals = {
          "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          # push to main, and pull requests from branches on this repo
          "token.actions.githubusercontent.com:sub" = [
            "repo:${var.github_repository}:ref:refs/heads/main",
            "repo:${var.github_repository}:pull_request",
          ]
        }
      }
    }]
  })
}

# Read-only, except on the state bucket: native locking writes and deletes
# a .tflock object, so a plan cannot run without S3 write there.
resource "aws_iam_policy" "github_plan" {
  count = local.oidc_enabled ? 1 : 0

  name        = "${local.project_name_env}-github-plan-policy"
  description = "Permissions for terraform plan from GitHub Actions"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = concat(
      var.tf_state_bucket != "" ? [
        {
          Effect   = "Allow"
          Action   = ["s3:ListBucket"]
          Resource = "arn:aws:s3:::${var.tf_state_bucket}"
        },
        {
          Effect   = "Allow"
          Action   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
          Resource = "arn:aws:s3:::${var.tf_state_bucket}/*"
        },
      ] : [],
      [
        {
          # Read-only and scoped to the one bucket, rather than naming each of
          # the GetBucket* calls a refresh makes across its sub-resources.
          Effect   = "Allow"
          Action   = ["s3:Get*", "s3:ListBucket"]
          Resource = "arn:aws:s3:::${local.aws_s3_artifacts_bucket_name}"
        },
        {
          Effect   = "Allow"
          Action   = ["s3:GetObject", "s3:GetObjectVersion", "s3:GetObjectTagging"]
          Resource = "arn:aws:s3:::${local.aws_s3_artifacts_bucket_name}/*"
        },
        {
          Effect   = "Allow"
          Action   = ["apigateway:GET"]
          Resource = "arn:aws:apigateway:${var.aws_region}::/apis/*"
        },
        {
          Effect = "Allow"
          Action = [
            "lambda:GetFunction",
            "lambda:GetFunctionCodeSigningConfig",
            "lambda:GetPolicy",
            "lambda:ListTags",
            "lambda:ListVersionsByFunction",
          ]
          Resource = "arn:aws:lambda:${var.aws_region}:${data.aws_caller_identity.current.account_id}:function:${local.project_name_env}-*"
        },
        {
          Effect   = "Allow"
          Action   = ["lambda:GetCodeSigningConfig", "lambda:ListTags"]
          Resource = "arn:aws:lambda:${var.aws_region}:${data.aws_caller_identity.current.account_id}:code-signing-config:*"
        },
        {
          Effect   = "Allow"
          Action   = ["signer:GetSigningProfile"]
          Resource = "arn:aws:signer:${var.aws_region}:${data.aws_caller_identity.current.account_id}:/signing-profiles/*"
        },
        {
          # no resource-level permissions for this action
          Effect   = "Allow"
          Action   = ["signer:DescribeSigningJob"]
          Resource = "*"
        },
        {
          Effect   = "Allow"
          Action   = ["iam:GetRole", "iam:ListRolePolicies", "iam:ListAttachedRolePolicies"]
          Resource = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${local.project_name_env}-*"
        },
        {
          Effect   = "Allow"
          Action   = ["iam:GetPolicy", "iam:GetPolicyVersion"]
          Resource = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:policy/${local.project_name_env}-*"
        },
        {
          Effect   = "Allow"
          Action   = ["logs:DescribeLogGroups", "logs:ListTagsForResource"]
          Resource = "arn:aws:logs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:log-group:*"
        },
        {
          Effect   = "Allow"
          Action   = ["sqs:GetQueueAttributes", "sqs:ListQueueTags"]
          Resource = "arn:aws:sqs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:${local.aws_sqs_queue_name}"
        },
        {
          # no resource-level permissions for this action
          Effect   = "Allow"
          Action   = ["ses:DescribeReceiptRule"]
          Resource = "*"
        },
      ]
    )
  })
}

resource "aws_iam_role_policy_attachment" "github_plan" {
  count = local.oidc_enabled ? 1 : 0

  role       = aws_iam_role.github_plan[0].name
  policy_arn = aws_iam_policy.github_plan[0].arn
}
