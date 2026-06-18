output "function_urls" {
  description = "Function URL for each deployed lambda, keyed by implementation."
  value       = { for k, v in aws_lambda_function_url.this : k => v.function_url }
}

output "go_function_url" {
  description = "Function URL of the Go lambda."
  value       = aws_lambda_function_url.this["go"].function_url
}

output "python_function_url" {
  description = "Function URL of the Python lambda."
  value       = aws_lambda_function_url.this["python"].function_url
}
