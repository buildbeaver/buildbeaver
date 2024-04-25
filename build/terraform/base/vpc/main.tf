data "aws_availability_zones" "available" {}

resource "aws_vpc" "vpc" {
  cidr_block           = "10.30.0.0/16"
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
  cidr_block              = "10.30.${10 + count.index}.0/24"
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
