variable "resource_prefix" {
  type        = string
  description = "An arbitrary string to include in the name of the VM (and any associated resources such as disks)"
}

variable "buildbeaver_version" {
  type        = string
  description = "The standard BuildBeaver version string derived from Git"
}

variable "aws_region" {
  type        = string
  default     = "set_me_when_using_aws"
  description = "The region to build Amazon AMIs in"
}

variable "aws_source_ami" {
  type        = string
  default     = "set_me_when_using_aws"
  description = "The Amazon AMI to use as the base image. Must be in the configured Amazon Region"
}

variable "aws_root_disk_size" {
  type        = string
  default     = "set_me_when_using_aws"
  description = "The size of the root SSD disk in GB to configure the Amazon AMI with"
}

variable "qemu_headless" {
  type        = string
  default     = true
  description = "Hide the VM window when using the qemu provider"
}

variable "qemu_output_directory" {
  type        = string
  default     = "set_me_when_using_qemu"
  description = "The directory to write the image to when using the qemu provider"
}

source "amazon-ebs" "bb-runner" {
  instance_type = "t2.micro"
  communicator  = "ssh"
  ssh_pty       = true
  ssh_username  = "ubuntu"
  ami_name      = "bb-runner-${var.buildbeaver_version}${var.resource_prefix}"
  run_tags      = {
    "Name" = "packer-bb-runner-${var.buildbeaver_version}${var.resource_prefix}"
  }
  region     = var.aws_region
  source_ami = var.aws_source_ami
  ami_block_device_mappings {
    delete_on_termination = true
    volume_size           = var.aws_root_disk_size
    volume_type           = "gp2"
    device_name           = "/dev/xvdh"
  }
}

source "qemu" "bb-runner" {
  vm_name          = "bb-runner-${var.buildbeaver_version}${var.resource_prefix}"
  iso_url          = "https://buildbeaver-build.s3.us-west-2.amazonaws.com/ubuntu-20.04.5-live-server-amd64.iso"
  iso_checksum     = "sha256:5035be37a7e9abbdc09f0d257f3e33416c1a0fb322ba860d42d74aa75c3468d4"
  memory           = 2048
  disk_image       = false
  output_directory = var.qemu_output_directory
  headless         = var.qemu_headless
  accelerator      = "kvm"
  disk_size        = "20000M"
  disk_interface   = "virtio"
  disk_compression = true
#  qemuargs         = [
#    ["-smp", "2"]
#  ]
  format       = "qcow2"
  net_device   = "virtio-net"
  boot_wait    = "3s"
  boot_command = [
    " <wait><enter><wait>",
    "<f6><esc>",
    "<bs><bs><bs><bs><bs><bs><bs><bs><bs><bs>",
    "<bs><bs><bs><bs><bs><bs><bs><bs><bs><bs>",
    "<bs><bs><bs><bs><bs><bs><bs><bs><bs><bs>",
    "<bs><bs><bs><bs><bs><bs><bs><bs><bs><bs>",
    "<bs><bs><bs><bs><bs><bs><bs><bs><bs><bs>",
    "<bs><bs><bs><bs><bs><bs><bs><bs><bs><bs>",
    "<bs><bs><bs><bs><bs><bs><bs><bs><bs><bs>",
    "<bs><bs><bs><bs><bs><bs><bs><bs><bs><bs>",
    "<bs><bs><bs>",
    "/casper/vmlinuz ",
    "initrd=/casper/initrd ",
    "autoinstall ",
    "ds=nocloud-net;s=http://{{.HTTPIP}}:{{.HTTPPort}}/ubuntu-2204/ ",
    "<enter>"
  ]
  http_directory         = "./http"
  shutdown_command       = "echo 'packer' | sudo -S shutdown -P now"
  ssh_username           = "ubuntu"
  ssh_password           = "ubuntu"
  ssh_timeout            = "60m"
  ssh_handshake_attempts = 30
}

build {
  sources = [
    "source.amazon-ebs.bb-runner",
    "source.qemu.bb-runner"
  ]
  provisioner "shell" {
    only   = ["amazon-ebs.bb-runner", "qemu.bb-runner"]
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y software-properties-common",
      "sudo apt-add-repository -y ppa:ansible/ansible",
      "sudo apt-get update",
      "sudo apt-get install -y ansible python3-pip"
    ]
  }
  provisioner "file" {
    source      = "../output/go/bin/bb-runner"
    destination = "/tmp/bb-runner"
  }
  provisioner "ansible-local" {
    playbook_dir      = "../ansible/playbooks"
    playbook_file     = "../ansible/playbooks/buildbeaver-runner.yml"
    inventory_file    = "../ansible/inventory/local-buildbeaver-runner"
    group_vars        = "../ansible/inventory/group_vars"
    galaxy_file       = "../ansible/requirements.yml"
    staging_directory = "/tmp/packer-ansible-staging"
    galaxy_roles_path = "/tmp/packer-ansible-staging"
    extra_arguments   = [
      "-vvvv --extra-vars \"buildbeaver_version=${var.buildbeaver_version}\""
    ]
  }
  provisioner "shell" {
    inline = [
      "sudo rm -f /root/.bash_history || true",
      "sudo rm -f /var/log/**/*.log",
      "sudo rm -rf /home/ubuntu/.ansible",
      "sudo rm -f /home/ubuntu/.ssh/*",
      "sudo rm -f /root/.ssh/*",
      "sudo rm -f /root/.ssh/authorized_keys"
    ]
  }
  provisioner "shell" {
    only   = ["qemu.bb-runner"]
    inline = [
      "while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done"
    ]
  }
}