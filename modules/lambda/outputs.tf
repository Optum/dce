output "arn" {
  value = aws_lambda_function.fn.arn
}

output "name" {
  value = aws_lambda_function.fn.function_name
}

output "invoke_arn" {
  value = aws_lambda_function.fn.invoke_arn
}

output "execution_role_name" {
  value = aws_iam_role.lambda_execution.name
}
output "execution_role_arn" {
  value = aws_iam_role.lambda_execution.arn
}
output "private_subnet_ids" {
  value = data.aws_subnets.private.ids
}

output "lambda_security_group_ids" {
  value = data.aws_security_groups.lambda_sg.ids
}