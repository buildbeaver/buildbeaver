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
