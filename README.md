# Uptime Go

A distributed uptime monitoring tool written in Go that checks website availability and performance metrics.

## Features
- HTTP(S) endpoint monitoring
- Response time tracking
- Custom check intervals
- Historical data storage

## Installation
```bash
git clone https://github.com/ryanriz/uptime-go
cd uptime-go
go build
```

## Configuration
1. Copy `configs/uptime.yml` to `/var/uptime-go/etc/uptime.yml`
2. Modify settings:
```yaml
checks:
  - name: Example Site
    url: https://example.com
    interval: 60s
```

## Usage
Run the application with custom configuration:
```bash
./uptime-go --config configs/uptime.yml
```
Run the application with default configuration:
```bash
./uptime-go
```
