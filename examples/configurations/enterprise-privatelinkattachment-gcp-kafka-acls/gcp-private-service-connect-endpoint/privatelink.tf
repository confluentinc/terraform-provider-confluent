locals {
  network_id = split(".", var.dns_domain)[0]
}

data "google_compute_network" "psc_endpoint_network" {
  name = var.customer_vpc_network
}

data "google_compute_subnetwork" "psc_endpoint_subnetwork" {
  name = var.customer_subnetwork_name
}

resource "google_compute_address" "psc_endpoint_ip" {
  for_each = var.subnet_name_by_zone

  name         = "ccloud-endpoint-ip-${local.network_id}-${each.key}"
  subnetwork   = var.customer_subnetwork_name
  address_type = "INTERNAL"
}

# Private Service Connect endpoint
resource "google_compute_forwarding_rule" "psc_endpoint_ilb" {
  for_each = var.subnet_name_by_zone

  name = "ccloud-endpoint-${local.network_id}-${each.key}"

  target                = lookup(confluent_network.private-service-connect.gcp[0].private_service_connect_service_attachments, each.key, "\n\nerror: ${each.key} subnet is missing from CCN's Private Service Connect service attachments")
  load_balancing_scheme = "" # need to override EXTERNAL default when target is a service attachment
  network               = var.customer_vpc_network
  ip_address            = google_compute_address.psc_endpoint_ip[each.key].id
}

# Private hosted zone for Private Service Connect endpoints
resource "google_dns_managed_zone" "psc_endpoint_hz" {
  name     = "ccloud-endpoint-zone-${local.network_id}"
  dns_name = "${var.dns_domain}."

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
  rrdatas = [
    for zone, _ in var.subnet_name_by_zone : google_compute_address.psc_endpoint_ip[zone].address
  ]
}

resource "google_dns_record_set" "psc_endpoint_zonal_rs" {
  for_each = var.subnet_name_by_zone

  name = "*.${each.key}.${google_dns_managed_zone.psc_endpoint_hz.dns_name}"
  type = "A"
  ttl  = 60

  managed_zone = google_dns_managed_zone.psc_endpoint_hz.name
  rrdatas      = [google_compute_address.psc_endpoint_ip[each.key].address]
}

resource "google_compute_firewall" "allow-https-kafka" {
  name    = "ccloud-endpoint-firewall-${local.network_id}"
  network = data.google_compute_network.psc_endpoint_network.id

  allow {
    protocol = "tcp"
    ports    = ["80", "443", "9092"]
  }

  direction          = "EGRESS"
  destination_ranges = [data.google_compute_subnetwork.psc_endpoint_subnetwork.ip_cidr_range]
}
