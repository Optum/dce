locals {
  user_assume_role_policy = templatefile("${path.module}/fixtures/iam/assume-role.json", {
    cognito_identity_pool_id = aws_cognito_identity_pool._.id
  })

  user_policy = templatefile("${path.module}/fixtures/iam/user-policy.json", {
    api_gateway_arn = var.api_gateway_arn
  })

  admin_assume_role_policy = templatefile("${path.module}/fixtures/iam/assume-role.json", {
    cognito_identity_pool_id = aws_cognito_identity_pool._.id
  })

  admin_policy = templatefile("${path.module}/fixtures/iam/admin-policy.json", {
    api_gateway_arn = var.api_gateway_arn
  })
}

resource "aws_iam_role" "user" {
  name = "${var.name}-user-${var.namespace}"

  assume_role_policy = local.user_assume_role_policy
}

resource "aws_iam_policy" "user" {
  name = "${var.name}-user-${var.namespace}"

  policy = local.user_policy
}
resource "aws_iam_policy_attachment" "user" {
  name = "${var.name}-user-${var.namespace}"

  policy_arn = aws_iam_policy.user.arn
  roles      = [aws_iam_role.user.name]
}

resource "aws_iam_role" "admin" {
  name = "${var.name}-admin-${var.namespace}"

  assume_role_policy = local.admin_assume_role_policy
}

resource "aws_iam_policy" "admin" {
  name = "${var.name}-admin-${var.namespace}"

  policy = local.admin_policy
}
resource "aws_iam_policy_attachment" "admin" {
  name = "${var.name}-admin-${var.namespace}"

  policy_arn = aws_iam_policy.admin.arn
  roles      = [aws_iam_role.admin.name]
}
