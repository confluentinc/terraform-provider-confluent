locals {
  hosted_zone = length(regexall(".glb", var.bootstrap)) > 0 ? replace(regex("^[^.]+-([0-9a-zA-Z]+[.].*):[0-9]+$", var.bootstrap)[0], "glb.", "") : regex("[.]([0-9a-zA-Z]+[.].*):[0-9]+$", var.bootstrap)[0]
  network_id  = regex("^([^.]+)[.].*", local.hosted_zone)[0]
}


data "azurerm_resource_group" "rg" {
  name = var.resource_group
}

data "azurerm_virtual_network" "vnet" {
  name                = var.vnet_name
  resource_group_name = data.azurerm_resource_group.rg.name
}

data "azurerm_subnet" "subnet" {
  for_each = var.subnet_name_by_zone

  name                 = each.value
  virtual_network_name = data.azurerm_virtual_network.vnet.name
  resource_group_name  = data.azurerm_resource_group.rg.name
}

resource "azurerm_private_dns_zone" "hz" {
  resource_group_name = data.azurerm_resource_group.rg.name

  name = local.hosted_zone
}

resource "azurerm_private_endpoint" "endpoint" {
  name                = "confluent-${local.network_id}-1"
  location            = var.vnet_region
  resource_group_name = data.azurerm_resource_group.rg.name

  subnet_id = data.azurerm_subnet.subnet[1].id

  private_service_connection {
    name                              = "confluent-${local.network_id}-1"
    is_manual_connection              = true
    private_connection_resource_alias = var.private_link_service_alias
    request_message                   = "PL"
  }
}

resource "azurerm_private_dns_zone_virtual_network_link" "hz" {
  name                  = data.azurerm_virtual_network.vnet.name
  resource_group_name   = data.azurerm_resource_group.rg.name
  private_dns_zone_name = azurerm_private_dns_zone.hz.name
  virtual_network_id    = data.azurerm_virtual_network.vnet.id
}

resource "azurerm_private_dns_a_record" "rr" {
  name                = "*"
  zone_name           = azurerm_private_dns_zone.hz.name
  resource_group_name = data.azurerm_resource_group.rg.name
  ttl                 = 60
  records = [
    azurerm_private_endpoint.endpoint.private_service_connection[0].private_ip_address
  ]
}

output "vpc_endpoint_id" {
  value = azurerm_private_endpoint.endpoint.id
}
