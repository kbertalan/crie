variable "region" {
  description = "AWS region to deploy the Lambda functions into."
  type        = string
  default     = "eu-central-1"
}

variable "name_prefix" {
  description = "Prefix used for all created resource names."
  type        = string
  default     = "crie-test"
}

variable "memory_size" {
  description = "Memory (MB) allocated to each Lambda function."
  type        = number
  default     = 256
}

variable "timeout" {
  description = "Timeout (seconds) for each Lambda function."
  type        = number
  default     = 30
}
