data "aws_availability_zones" "available" {}

resource "aws_vpc" "vpc" {
  cidr_block           = "10.20.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags                 = var.resource_tags
}

###############################################################################
# Public Subnets
###############################################################################

resource "aws_subnet" "public" {
  count                   = length(data.aws_availability_zones.available.names)
  vpc_id                  = aws_vpc.vpc.id
  cidr_block              = "10.20.${10 + count.index}.0/24"
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = true
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-public-${data.aws_availability_zones.available.names[count.index]}"
  })
}

# Add an internet gateway to the VPC to allow access to/from internet
resource "aws_internet_gateway" "public" {
  vpc_id = aws_vpc.vpc.id
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-public"
  })
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.vpc.id
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-public"
  })
}

resource "aws_route" "public" {
  route_table_id         = aws_route_table.public.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.public.id
}

resource "aws_route_table_association" "public" {
  count          = length(aws_subnet.public.*)
  subnet_id      = element(aws_subnet.public.*.id, count.index)
  route_table_id = aws_route_table.public.id
}

###############################################################################
# Private Subnets
###############################################################################

resource "aws_subnet" "private" {
  count                   = length(data.aws_availability_zones.available.names)
  vpc_id                  = aws_vpc.vpc.id
  cidr_block              = "10.20.${20 + count.index}.0/24"
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = false
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-private-${data.aws_availability_zones.available.names[count.index]}"
  })
}

resource "aws_eip" "nat_eip" {
  count = length(aws_subnet.private.*)
  vpc   = true
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-private-${data.aws_availability_zones.available.names[count.index]}"
  })
}

resource "aws_nat_gateway" "private" {
  count         = length(aws_subnet.private.*)
  allocation_id = element(aws_eip.nat_eip.*.id, count.index)
  subnet_id     = element(aws_subnet.public.*.id, count.index)
  depends_on    = [aws_internet_gateway.public]
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-private-${data.aws_availability_zones.available.names[count.index]}"
  })
}

resource "aws_route_table" "private" {
  count  = length(aws_subnet.private.*)
  vpc_id = aws_vpc.vpc.id
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-private-${data.aws_availability_zones.available.names[count.index]}"
  })
}

resource "aws_route" "private" {
  count                  = length(aws_subnet.private.*)
  route_table_id         = element(aws_route_table.private.*.id, count.index)
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = element(aws_nat_gateway.private.*.id, count.index)
}

resource "aws_route_table_association" "private" {
  count          = length(aws_subnet.private.*)
  subnet_id      = element(aws_subnet.private.*.id, count.index)
  route_table_id = element(aws_route_table.private.*.id, count.index)
}