###############################################################################
# DMZ
###############################################################################

resource "aws_security_group" "dmz" {
  name   = "${var.base_resource_name}-dmz"
  vpc_id = var.vpc_id
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-dmz"
  })
}

# Allow us to have SSH access to servers in the DMZ via the internet.
# We can hop further in to the network from here
resource "aws_security_group_rule" "dmz_ssh_ingress" {
  security_group_id = aws_security_group.dmz.id
  type              = "ingress"
  from_port         = 22
  to_port           = 22
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
}

# Allow servers in the DMZ to access the outside world
resource "aws_security_group_rule" "mz_egress" {
  security_group_id = aws_security_group.dmz.id
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
}

###############################################################################
# App API Load Balancer
###############################################################################

resource "aws_security_group" "app_lb" {
  name   = "${var.base_resource_name}-app-api-lb"
  vpc_id = var.vpc_id
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-app-api-lb"
  })
}

# Allow the outside world to connect to the app API on port 80 (load balancer will redirect to HTTPS/443)
resource "aws_security_group_rule" "app_lb_http_ingress" {
  security_group_id = aws_security_group.app_lb.id
  type              = "ingress"
  from_port         = 80
  to_port           = 80
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
}

# Allow the outside world to connect to the app API on port 443
resource "aws_security_group_rule" "app_lb_https_ingress" {
  security_group_id = aws_security_group.app_lb.id
  type              = "ingress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
}

# Allow the firewall to make outbound connections to our app API servers (app API listens on port 80, load
# balancer terminates TLS)
resource "aws_security_group_rule" "app_lb_egress" {
  security_group_id = aws_security_group.app_lb.id
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
}

###############################################################################
# BB Server Container
###############################################################################

resource "aws_security_group" "bb_server_container" {
  name   = "${var.base_resource_name}-bb-server-container"
  vpc_id = var.vpc_id
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-bb-server-container"
  })
}

# Allow the App API load balancer to forward HTTP traffic to the bb server container
resource "aws_security_group_rule" "app_api_ingress" {
  security_group_id        = aws_security_group.bb_server_container.id
  type                     = "ingress"
  from_port                = 80
  to_port                  = var.app_api_listen_port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.app_lb.id
}

# Allow the Runner API load balancer to forward HTTPS traffic to the bb server container.
# The Runner API load balancer is a network lb, and it transparently exposes client connections
# to our container e.g. the source IP will be the clients, not the load balancers.
resource "aws_security_group_rule" "runner_api_ingress" {
  security_group_id = aws_security_group.bb_server_container.id
  type              = "ingress"
  from_port         = 443
  to_port           = var.runner_api_listen_port
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  ipv6_cidr_blocks  = ["::/0"]
}

# All the bb server container to talk to any external endpoint
resource "aws_security_group_rule" "world_egress" {
  security_group_id = aws_security_group.bb_server_container.id
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
}

###############################################################################
# Runner Servers
###############################################################################

resource "aws_security_group" "runner_servers" {
  name   = "${var.base_resource_name}-runner"
  vpc_id = var.vpc_id
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-runner"
  })
}

# Allow servers in the dmz to access runner servers (useful for us to perform diagnostics)
resource "aws_security_group_rule" "runner_servers_dmz_ssh_ingress" {
  security_group_id        = aws_security_group.runner_servers.id
  type                     = "ingress"
  from_port                = 22
  to_port                  = 22
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.dmz.id
}

# Runner servers should be able to access the outside world
resource "aws_security_group_rule" "runner_servers_world_egress" {
  security_group_id = aws_security_group.runner_servers.id
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
}

###############################################################################
# Database
###############################################################################

# A security group that controls access into and out of the database servers
resource "aws_security_group" "database" {
  name   = "${var.base_resource_name}-database"
  vpc_id = var.vpc_id
  tags = merge(var.resource_tags, {
    Name = "${var.base_resource_name}-database"
  })
}

# Allow servers in the DMZ to access the database
resource "aws_security_group_rule" "database_dmz_ingress" {
  security_group_id        = aws_security_group.database.id
  type                     = "ingress"
  from_port                = var.database_port
  to_port                  = var.database_port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.dmz.id
}

resource "aws_security_group_rule" "database_bb_server_container_ingress" {
  security_group_id        = aws_security_group.database.id
  type                     = "ingress"
  from_port                = var.database_port
  to_port                  = var.database_port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.bb_server_container.id
}