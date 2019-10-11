
output user_pool_id {
  value = aws_cognito_user_pool._.id
}

output user_pool_arn {
  value = aws_cognito_user_pool._.arn
}

output client_id {
  value = aws_cognito_user_pool_client._.id
}

output user_policy_arn {
  value = aws_iam_policy.user.arn
}

output user_role_arn {
  value = aws_iam_role.user.arn
}

output admin_policy_arn {
  value = aws_iam_policy.admin.arn
}

output admin_role_arn {
  value = aws_iam_role.admin.arn
}
