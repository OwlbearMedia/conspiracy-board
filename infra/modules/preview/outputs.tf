output "url" {
  value = "https://${local.host}"
}

output "database" {
  value = local.db_name
}
