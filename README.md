# Uptime Go

A distributed uptime monitoring tool written in Go that checks website availability and performance metrics.

## Features
- HTTP(S) endpoint monitoring
- Response time tracking
- Custom check intervals
- Historical data storage

## Installation
```bash
git clone https://github.com/genbucyber/uptime-go
cd uptime-go
go build
```

## Configuration
```yaml
monitor:
  - url: https://example.com
    enabled: true
    interval: 5m
    response_time_threshold: 5s
```

## Usage
Run the application:
```bash
./uptime-go --config configs/uptime.yml
```

Show report:
```bash
./uptime-go report
```

Example output: 

```json
[
  {
    "url": "https://example.com",
    "is_up": true,
    "status_code": 200,
    "response_time": 1233,
    "certificate_expired_date": "2025-09-23T06:56:43Z",
    "last_up": "2025-08-15T16:19:35.081509779+08:00",
    "last_check": "2025-08-15T16:25:05.930357715+08:00"
  }
]
```

Report with specific site:

```bash
./uptime-go report --url https://example.com
```

Example output: 

```json
{
  "url": "https://example.com",
  "is_up": true,
  "status_code": 200,
  "response_time": 1233,
  "certificate_expired_date": "2025-09-23T06:56:43Z",
  "last_up": "2025-08-15T16:19:35.081509779+08:00",
  "last_check": "2025-08-15T16:25:05.930357715+08:00",
  "histories": [
    {
      "is_up": true,
      "response_time": 1233,
      "created_at": "2025-08-15T16:25:05.93061437+08:00"
    },
  ]
}
```