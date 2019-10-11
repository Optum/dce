
output user_pool_id {
  value = aws_cognito_user_pool._.id
}

output user_pool_arn {
  value = aws_cognito_user_pool._.arn
}

output client_id {
  value = aws_cognito_user_pool_client._.id
}

