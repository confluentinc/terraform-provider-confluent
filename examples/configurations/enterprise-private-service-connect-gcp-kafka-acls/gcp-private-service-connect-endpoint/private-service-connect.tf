locals {
  dns_domain = var.dns_domain
}

data "google_compute_network" "psc_endpoint_network" {
  name = var.customer_vpc_network
}

data "google_compute_subnetwork" "psc_endpoint_subnetwork" {
  name = var.customer_subnetwork_name
}

resource "google_compute_address" "psc_endpoint_ip" {
  name         = "ccloud-endpoint-ip-${replace(local.dns_domain, ".", "-")}"
  subnetwork   = var.customer_subnetwork_name
  address_type = "INTERNAL"
}

# Private Service Connect endpoint
resource "google_compute_forwarding_rule" "psc_endpoint_ilb" {
  name                  = "ccloud-endpoint-${replace(local.dns_domain, ".", "-")}"
  target                = var.platt_service_attachment_uri
  load_balancing_scheme = "" # need to override EXTERNAL default when target is a service attachment
  network               = var.customer_vpc_network
  ip_address            = google_compute_address.psc_endpoint_ip.id
}

# Private hosted zone for Private Service Connect endpoints
resource "google_dns_managed_zone" "psc_endpoint_hz" {
  name     = "ccloud-endpoint-zone-${replace(local.dns_domain, ".", "-")}"
  dns_name = "${local.dns_domain}."

  visibility = "private"

  private_visibility_config {
    networks {
      network_url = data.google_compute_network.psc_endpoint_network.id
    }
  }
}

resource "google_dns_record_set" "psc_endpoint_rs" {
  name = "*.${google_dns_managed_zone.psc_endpoint_hz.dns_name}"
  type = "A"
  ttl  = 60

  managed_zone = google_dns_managed_zone.psc_endpoint_hz.name
  rrdatas = [google_compute_address.psc_endpoint_ip.address]
}

resource "google_compute_firewall" "allow-https-kafka" {
  name    = "ccloud-fw-${replace(local.dns_domain, ".", "-")}"
  network = data.google_compute_network.psc_endpoint_network.id

  allow {
    protocol = "tcp"
    ports    = ["80", "443", "9092"]
  }

  direction          = "EGRESS"
  destination_ranges = [data.google_compute_subnetwork.psc_endpoint_subnetwork.ip_cidr_range]
}

output "private_service_connect_connection_id" {
  value = google_compute_forwarding_rule.psc_endpoint_ilb.psc_connection_id
}
