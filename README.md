# Proxmox API Wrapper

A lightweight Golang API and CLI for managing Proxmox VE resources.

## Features

- REST API for Proxmox VE management
- Command-line interface for creating VMs from JSON configuration
- VM and Container management capabilities
- Resource listing (storage, networks, ISOs)

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

## API Endpoints

- `GET /api/v1/vms` - List all VMs
- `POST /api/v1/vms` - Create a new VM
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

## License

MIT
