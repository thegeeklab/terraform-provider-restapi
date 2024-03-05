provider "restapi" {
  endpoint             = "https://example.com"
  write_returns_object = true

  headers = {
    "X-Auth-Token" = var.AUTH_TOKEN,
    "Content-Type" = "application/json"
  }

  create_method  = "POST"
  update_method  = "PUT"
  destroy_method = "DELETE"
}
