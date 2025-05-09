# Proxmox API Wrapper

A lightweight Golang API and CLI for managing Proxmox VE resources.

## Features

- REST API for Proxmox VE management
- Command-line interface for creating VMs from JSON configuration
- VM and Container management capabilities
- Resource listing (storage, networks, ISOs)
- Simple API token authentication
- Custom API token for secure access

## Requirements

- Go 1.18+
- Proxmox VE 7.0+
- API Token with appropriate permissions

## Setup

1. Clone this repository
2. Create an `env/.env` file with your Proxmox credentials:

```
APIURL=https://your-proxmox-server:8006/api2/json
NODE=your-node-name
PROXMOX_TOKEN_ID=your-user@pve\!your-token-id
PROXMOX_TOKEN_SECRET=your-token-secret

# API Authentication
API_TOKEN=your-api-token

# VM Template Configuration
# Templates are defined in env/templates.json
```

3. Build the application:

```bash
go build -o main
```

## Usage

### API Server

Run the application without arguments to start the API server:

```bash
./main
```

The API will be available at http://localhost:8080 (or the port specified in the PORT environment variable).

### CLI Mode

Create a VM from a JSON configuration file:

```bash
./main -input vm_config.json
```

Example JSON configuration:

```json
{
    "vmid": "200",
    "name": "test-vm",
    "cores": 2,
    "memory": 4096,
    "disk": "local-lvm:20G",
    "net": "vmbr0",
    "iso": "local:iso/debian-12.5.0-amd64-netinst.iso",
    "ostype": "l26",
    "cpu": "host",
    "sockets": 1
}
```

Using a template:

```json
{
    "name": "vm-from-template",
    "template": "debian",
    "memory": 4096,
    "cores": 2,
    "disk": "local-lvm:20G",
    "net": "vmbr0"
}
```

Using a template with CloudInit:

```json
{
    "name": "cloudinit-vm",
    "template": "debian",
    "memory": 4096,
    "cores": 2,
    "disk": "local-lvm:20G",
    "net": "vmbr0",
    "cloudinit": true,
    "ipconfig": {
        "0": "ip=192.168.1.100/24,gw=192.168.1.1"
    },
    "sshkeys": "ssh-rsa AAAAB3NzaC1yc2EAAA... user@example.com",
    "nameserver": "1.1.1.1 8.8.8.8",
    "searchdomain": "example.com",
    "ciuser": "clouduser",
    "cipassword": "cloudpassword"
}
```

## API Endpoints

### Proxmox API Endpoints (All Protected by API Token)

- `GET /api/v1/vms` - List all VMs
- `POST /api/v1/vms` - Create a new VM
- `POST /api/v1/vms/template` - Create a VM from template
- `POST /api/v1/vms/clone` - Clone a VM
- `GET /api/v1/vms/:vmid` - Get VM details
- `DELETE /api/v1/vms/:vmid` - Delete a VM
- `POST /api/v1/vms/:vmid/start` - Start a VM
- `POST /api/v1/vms/:vmid/stop` - Stop a VM
- `POST /api/v1/vms/:vmid/reboot` - Reboot a VM
- `GET /api/v1/resources` - Get resource information
- `GET /api/v1/nodes` - List nodes
- `GET /api/v1/storages` - List storage
- `GET /api/v1/networks` - List networks
- `GET /api/v1/isos` - List available ISOs
- `GET /api/v1/templates` - List available VM templates

## API Usage

### Authentication

All API endpoints require a valid API token in the Authorization header:

```
Authorization: Bearer your-api-token
```

The API token is configured in your `.env` file. For security purposes, use a strong, randomly generated token in production environments.

### Response Format

All API endpoints use a consistent response format:

```json
{
  "success": true|false,
  "data": [result object or array],
  "error": "Error message if success is false"
}
```

### VM Management

> **Note:** 
> - Templates are defined in the `env/templates.json` file with the format `{"templates": {"debian": "9000", "ubuntu": "9001"}}`
> - The VMID in the templates.json file refers to an existing VM that will be cloned when creating a new VM from that template
> - For CloudInit support, your template VM should be prepared with cloud-init packages and configured properly
> - When using CloudInit, the ISO parameter is ignored as the ide2 device is used for CloudInit
> - CloudInit can set user credentials with `ciuser` and `cipassword` parameters

#### List VMs
```
GET /api/v1/vms?node=node-name
```

Response:
```json
{
  "success": true,
  "data": [
    {
      "vmid": 100,
      "name": "vm-name",
      "status": "running"
    }
  ]
}
```

#### Create VM
```
POST /api/v1/vms
```

Request Body (standard ISO boot):
```json
{
  "vmid": "200",
  "name": "test-vm",
  "cores": 2,
  "memory": 4096,
  "disk": "local-lvm:20G",
  "net": "vmbr0",
  "iso": "local:iso/debian-12.5.0-amd64-netinst.iso",
  "ostype": "l26",
  "cpu": "host",
  "sockets": 1
}
```

Request Body (with CloudInit):
```json
{
  "vmid": "200",
  "name": "test-vm",
  "cores": 2,
  "memory": 4096,
  "disk": "local-lvm:20G",
  "net": "vmbr0",
  "ostype": "l26",
  "cpu": "host",
  "sockets": 1,
  "cloudinit": true,
  "ipconfig": {
    "0": "ip=192.168.1.100/24,gw=192.168.1.1"
  },
  "sshkeys": "ssh-rsa AAAAB3NzaC1yc2EAAA... user@example.com",
  "nameserver": "1.1.1.1 8.8.8.8",
  "searchdomain": "example.com",
  "ciuser": "clouduser",
  "cipassword": "cloudpassword"
}
```

Response:
```json
{
  "success": true,
  "data": {
    "task_id": "UPID:..."
  }
}
```

#### VM Operations (Start/Stop/Reboot)
```
POST /api/v1/vms/{vmid}/start
POST /api/v1/vms/{vmid}/stop
POST /api/v1/vms/{vmid}/reboot
```

Response:
```json
{
  "success": true
}
```

#### Create VM from Template
```
POST /api/v1/vms/template
```

Request Body (standard):
```json
{
  "node": "pve",
  "vmid": "200",
  "name": "template-vm",
  "template": "debian",
  "memory": 4096,
  "cores": 2,
  "disk": "local-lvm:20G",
  "net": "vmbr0"
}
```

Request Body (with CloudInit):
```json
{
  "node": "pve",
  "vmid": "200",
  "name": "template-vm",
  "template": "debian",
  "memory": 4096,
  "cores": 2,
  "disk": "local-lvm:20G",
  "net": "vmbr0",
  "cloudinit": true,
  "ipconfig": {
    "0": "ip=192.168.1.100/24,gw=192.168.1.1"
  },
  "sshkeys": "ssh-rsa AAAAB3NzaC1yc2EAAA... user@example.com",
  "nameserver": "1.1.1.1 8.8.8.8",
  "searchdomain": "example.com",
  "ciuser": "clouduser",
  "cipassword": "cloudpassword"
}
```

Response:
```json
{
  "success": true,
  "data": {
    "task_id": "UPID:..."
  }
}
```

#### Clone VM
```
POST /api/v1/vms/clone
```

Request Body:
```json
{
  "source_node": "pve",
  "source_vmid": "100",
  "target_node": "pve",
  "target_vmid": "101",
  "name": "clone-vm"
}
```

Response:
```json
{
  "success": true,
  "data": {
    "task_id": "UPID:..."
  }
}
```

#### Get Available Templates
```
GET /api/v1/templates
```

Response:
```json
{
  "success": true,
  "data": {
    "debian": "9000",
    "ubuntu": "9001",
    "centos": "9002"
  }
}
```

### Container Management

Container Configuration Format:
```json
{
  "node": "pve-node",
  "ctid": "100",
  "name": "container-name",
  "memory": "2000",
  "swap": "2000",
  "cores": "2",
  "disk": "8",
  "storage": "local",
  "net": "name=eth0,bridge=vmbr0,ip=dhcp",
  "password": "yourRootPassword",
  "template": "local:vztmpl/debian-12-standard_12.7-1_amd64.tar.zst",
  "unprivileged": true
}
```

### Resource Information

```
GET /api/v1/resources?node=node-name
GET /api/v1/nodes
GET /api/v1/storages?node=node-name
GET /api/v1/networks?node=node-name
GET /api/v1/isos?node=node-name
```

## License

MIT