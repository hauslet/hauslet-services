# Global External Application Load Balancer for Cloud Run with Custom Domain

# 1. Global Static IP Address
resource "google_compute_global_address" "lb_ip" {
  name        = "hauslet-lb-ip"
  project     = var.project_id
  description = "Global static IP for Hauslet Load Balancer"
}

# 2. Managed SSL Certificate
resource "google_compute_managed_ssl_certificate" "lb_cert" {
  name    = "hauslet-lb-cert"
  project = var.project_id

  managed {
    domains = [var.domain_name]
  }
}

# 3. Serverless Network Endpoint Group (NEG) for Cloud Run
resource "google_compute_region_network_endpoint_group" "api_neg" {
  name                  = "hauslet-api-neg"
  project               = var.project_id
  network_endpoint_type = "SERVERLESS"
  region                = var.region

  cloud_run {
    service = var.api_service_name
  }
}

# 4. Backend Service
resource "google_compute_backend_service" "lb_backend" {
  name                  = "hauslet-lb-backend"
  project               = var.project_id
  protocol              = "HTTPS"
  port_name             = "http"
  load_balancing_scheme = "EXTERNAL"
  timeout_sec           = 30

  backend {
    group = google_compute_region_network_endpoint_group.api_neg.id
  }
}

# 5. URL Map
resource "google_compute_url_map" "lb_url_map" {
  name            = "hauslet-lb-url-map"
  project         = var.project_id
  default_service = google_compute_backend_service.lb_backend.id
}

# 6. Target HTTPS Proxy
resource "google_compute_target_https_proxy" "lb_proxy" {
  name             = "hauslet-lb-proxy"
  project          = var.project_id
  url_map          = google_compute_url_map.lb_url_map.id
  ssl_certificates = [google_compute_managed_ssl_certificate.lb_cert.id]
}

# 7. Global Forwarding Rule
resource "google_compute_global_forwarding_rule" "lb_forwarding_rule" {
  name                  = "hauslet-lb-forwarding-rule"
  project               = var.project_id
  target                = google_compute_target_https_proxy.lb_proxy.id
  port_range            = "443"
  ip_address            = google_compute_global_address.lb_ip.address
  load_balancing_scheme = "EXTERNAL"
}

# Output the IP address for DNS configuration
output "load_balancer_ip" {
  description = "The static global IP address of the load balancer"
  value       = google_compute_global_address.lb_ip.address
}
