# Proxmox API Wrapper

This project is a custom API wrapper designed to simplify interactions with the Proxmox API. 

> **Note**: This is **NOT** the official Proxmox API! This is a custom implementation to make accessing the Proxmox API easier.

## Features

- Fetch information about nodes and VMs.
- Perform actions like starting, stopping, creating, and deleting VMs.
- Query VMs by their name or ID.
- JSON-formatted responses for easy integration.

## Requirements

- [Go (1.19+)](https://golang.org/)
- A running Proxmox VE instance.
- API Token and appropriate permissions for the Proxmox API.

## Installation

Clone this repository:

```bash
git clone https://github.com/your-username/proxmox-api-wrapper.git
cd proxmox-api-wrapper
