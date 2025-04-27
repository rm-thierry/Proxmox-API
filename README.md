# Proxmox API Wrapper

A Go-based API wrapper for Proxmox VE that provides simplified management of VMs and containers.

## Features

- Create, delete, start, and stop VMs and containers
- List available VMs and containers
- Automatic VM and container ID generation
- Support for multiple Proxmox nodes
- RESTful API design

## Requirements

- [Go (1.19+)](https://golang.org/)
- A running Proxmox VE instance
- API Token and appropriate permissions for the Proxmox API (optional)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/Proxmox-API.git
cd Proxmox-API
```

2. Build the application:
```bash
go build -o proxmox-api src/main.go
```

## Configuration

Create an `env/.env` file with the following variables:

```
# Proxmox API Connection
APIURL=https://your-proxmox-server:8006/api2/json
NODE=your-node-name
PROXMOX_TOKEN_ID=your-token-id
PROXMOX_TOKEN_SECRET=your-token-secret

# Optional Database Configuration
DBHOST=localhost
DBUSER=dbuser
DBPASS=dbpassword
DBNAME=dbname

# Server Configuration
PORT=8080
```

## Usage

### Running the server

```bash
./proxmox-api
```

The API will be available at `http://localhost:8080/api/v1/`.

### API Endpoints

#### VMs

- `GET /api/v1/vms` - List all VMs
- `POST /api/v1/vms` - Create a new VM
- `GET /api/v1/vms/:vmid` - Get VM details
- `DELETE /api/v1/vms/:vmid` - Delete a VM
- `POST /api/v1/vms/:vmid/start` - Start a VM
- `POST /api/v1/vms/:vmid/stop` - Stop a VM

#### Request Examples

Create a VM:
```json
POST /api/v1/vms
{
  "name": "test-vm",
  "cores": "2",
  "memory": "4096",
  "disk": "local-lvm:20",
  "ostype": "debian"
}
```

## License

This project is open source and available under the [MIT License](LICENSE).

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request