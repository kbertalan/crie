data "aws_iam_policy_document" "assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "lambda" {
  name               = "${var.name_prefix}-lambda"
  assume_role_policy = data.aws_iam_policy_document.assume.json
}

resource "aws_iam_role_policy_attachment" "logs" {
  role       = aws_iam_role.lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# Managed explicitly so `tofu destroy` cleans the logs up too.
resource "aws_cloudwatch_log_group" "this" {
  for_each = local.functions

  name              = "/aws/lambda/${var.name_prefix}-${each.key}"
  retention_in_days = 7
}

resource "aws_lambda_function" "this" {
  for_each = local.functions

  function_name = "${var.name_prefix}-${each.key}"
  role          = aws_iam_role.lambda.arn
  package_type  = "Image"
  # Pin to the pushed digest so the function updates whenever the image changes.
  image_uri     = "${aws_ecr_repository.this[each.key].repository_url}@${docker_registry_image.this[each.key].sha256_digest}"
  architectures = ["x86_64"]
  memory_size   = var.memory_size
  timeout       = var.timeout

  depends_on = [
    aws_iam_role_policy_attachment.logs,
    aws_cloudwatch_log_group.this,
  ]
}

# Public Function URL so the unsigned test/client can invoke the function directly.
resource "aws_lambda_function_url" "this" {
  for_each = local.functions

  function_name      = aws_lambda_function.this[each.key].function_name
  authorization_type = "NONE"
}
