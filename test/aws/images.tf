# Build each container image from the repo root (so the Dockerfiles can COPY
# both the crie sources and the handler code), then push it to ECR.
resource "docker_image" "this" {
  for_each = local.functions

  name = "${aws_ecr_repository.this[each.key].repository_url}:latest"

  build {
    context    = abspath(local.repo_root)
    dockerfile = each.value.dockerfile
    platform   = "linux/amd64"
  }

  # Rebuild whenever the Dockerfile, the handler sources, or the crie sources change.
  triggers = {
    sources = sha1(join(",", [
      for f in sort(setunion(local.crie_sources, each.value.sources, toset([each.value.dockerfile]))) :
      filemd5("${local.repo_root}/${f}")
    ]))
  }
}

resource "docker_registry_image" "this" {
  for_each = local.functions

  name          = docker_image.this[each.key].name
  keep_remotely = false

  triggers = {
    digest = docker_image.this[each.key].image_id
  }
}
