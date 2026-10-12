# Uptime Go

A distributed uptime monitoring tool written in Go that checks website availability and performance metrics.

## Features
- HTTP(S) endpoint monitoring
- Response time tracking
- Custom check intervals
- Historical data storage
- SSL expiry monitoring with an optional auto-renew hook

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
    ip_type: ipv4
    certificate_monitoring: true
    certificate_expired_before: 31d
    bash_hook: 'certbot renew --quiet'  # optional
```

### SSL expiry hook
When the certificate has `certificate_expired_before` or less remaining (or is already expired), `bash_hook` runs once per incident. If it exits 0 and the re-checked certificate is renewed, no notification is sent. Otherwise the notification is sent with `hook_output` and `hook_error`.

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
