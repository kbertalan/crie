# One ECR repository per lambda. force_delete lets `tofu destroy` remove the
# repo even though it still holds the image we pushed.
resource "aws_ecr_repository" "this" {
  for_each = local.functions

  name                 = "${var.name_prefix}-${each.key}"
  image_tag_mutability = "MUTABLE"
  force_delete         = true
}
