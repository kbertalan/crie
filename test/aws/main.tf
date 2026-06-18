terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 6.51"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.9.0"
    }
  }
}

provider "aws" {
  region = var.region
}

# Authenticate the docker provider against ECR so it can push the built images.
provider "docker" {
  registry_auth {
    address  = local.ecr_registry
    username = data.aws_ecr_authorization_token.token.user_name
    password = data.aws_ecr_authorization_token.token.password
  }
}

data "aws_caller_identity" "current" {}

data "aws_ecr_authorization_token" "token" {}

locals {
  repo_root    = "${path.module}/../.."
  ecr_registry = "${data.aws_caller_identity.current.account_id}.dkr.ecr.${var.region}.amazonaws.com"

  # crie sources are baked into every image, so a change to them must rebuild both.
  crie_sources = setunion(
    fileset(local.repo_root, "cmd/**"),
    fileset(local.repo_root, "internal/**"),
    toset(["go.mod", "go.sum"]),
  )

  # The two lambdas we deploy. Each runs its handler wrapped by crie (delegate
  # mode on AWS), exactly like the local Dockerfiles in test/go and test/python.
  functions = {
    go = {
      dockerfile = "test/go/Dockerfile"
      sources    = fileset(local.repo_root, "test/go/**")
    }
    python = {
      dockerfile = "test/python/Dockerfile"
      sources    = fileset(local.repo_root, "test/python/**")
    }
  }
}
