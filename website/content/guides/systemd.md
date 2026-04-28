---
title: Run as a systemd service
description: Run a bunpy app as a systemd service with automatic restarts, environment files, logging via journald, and socket activation.
---

On a Linux server, systemd is the standard way to run a long-lived process. It handles startup on boot, automatic restart on crash, log collection via journald, and dependency ordering with other services. This guide covers a production-ready unit file for a bunpy app.


## Prerequisites

- A Linux server with systemd (Ubuntu 20.04+, Debian 11+, RHEL 8+)
- bunpy installed system-wide or in a specific user's home directory
- Your app deployed to a directory on the server, e.g. `/opt/myapp`


## Install your app on the server

```bash
# Create the app directory
sudo mkdir -p /opt/myapp
sudo chown appuser:appuser /opt/myapp

# Copy files (from your CI/CD pipeline or manually)
rsync -av --exclude='.git' ./ appuser@server:/opt/myapp/

# Install bunpy as the app user
sudo -u appuser bash -c 'curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash'

# Install dependencies
sudo -u appuser bash -c 'cd /opt/myapp && /home/appuser/.bunpy/bin/bunpy install --frozen'
```

Or use the `.pyz` approach and copy a single file:

```bash
# Build locally
bunpy build src/myapp/__main__.py -o dist/myapp.pyz

# Copy to server
scp dist/myapp.pyz appuser@server:/opt/myapp/myapp.pyz
```


## Create a dedicated user

Never run application code as root. Create a dedicated system user with no login shell:

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin appuser
```

For apps that need a home directory (e.g., to store the bunpy cache):

```bash
sudo useradd --system --create-home --shell /usr/sbin/nologin appuser
```


## Environment file

Secrets and configuration belong in an environment file, not in the unit file. The unit file is often world-readable; the environment file can be restricted to root and the app user.

Create `/etc/myapp/environment`:

```bash
sudo mkdir -p /etc/myapp
sudo touch /etc/myapp/environment
sudo chmod 640 /etc/myapp/environment
sudo chown root:appuser /etc/myapp/environment
```

Edit `/etc/myapp/environment`:

```
DATABASE_URL=postgresql://appuser:secret@localhost/myapp
SECRET_KEY=change-this-to-a-random-value
PORT=8080
LOG_LEVEL=info
ENVIRONMENT=production
```

This file is not committed to source control. On a new server, provision it from a secrets manager (Vault, AWS Secrets Manager, etc.) or set it manually.


## Unit file

Create `/etc/systemd/system/myapp.service`:

```ini
[Unit]
Description=myapp - bunpy web service
Documentation=https://github.com/tamnd/myapp
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=appuser
Group=appuser
WorkingDirectory=/opt/myapp
EnvironmentFile=/etc/myapp/environment

# Full path to bunpy; adjust if installed elsewhere
ExecStart=/home/appuser/.bunpy/bin/bunpy server.py

# Restart policy
Restart=always
RestartSec=5s
StartLimitBurst=5
StartLimitIntervalSec=60s

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=myapp

# Security hardening
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ReadWritePaths=/opt/myapp/data
ProtectHome=read-only

# Resource limits
LimitNOFILE=65536
MemoryMax=512M

[Install]
WantedBy=multi-user.target
```

### Key directives explained

**`After=network.target postgresql.service`** - systemd starts myapp only after the network is up and PostgreSQL has started. Use `After=` with services your app requires to connect on startup.

**`EnvironmentFile=/etc/myapp/environment`** - injects every line from the file as an environment variable. Lines starting with `#` are comments and are ignored.

**`WorkingDirectory=/opt/myapp`** - sets the current directory for the process. Relative file paths in your app resolve against this directory.

**`Restart=always`** - restarts the process whenever it exits, regardless of exit code. Use `Restart=on-failure` if you want to preserve intentional exits (e.g., `sys.exit(0)`).

**`RestartSec=5s`** - waits 5 seconds before restarting. Prevents a crashing app from hammering a database with reconnect attempts.

**`StartLimitBurst=5` / `StartLimitIntervalSec=60s`** - if the service crashes more than 5 times in 60 seconds, systemd stops trying to restart it. This prevents runaway restart loops. After hitting the limit, restart manually with `systemctl restart myapp`.

**`ProtectSystem=strict`** - mounts the filesystem read-only for the service process except for paths listed in `ReadWritePaths`. This prevents the app from writing to unexpected locations.

**`PrivateTmp=yes`** - gives the service its own `/tmp` directory. Files written to `/tmp` by the app are not visible to other processes.


## Enable and start

```bash
# Reload systemd after writing or editing the unit file
sudo systemctl daemon-reload

# Enable the service (starts on boot)
sudo systemctl enable myapp

# Start the service now
sudo systemctl start myapp

# Check status
sudo systemctl status myapp
```

Example `status` output:

```
● myapp.service - myapp - bunpy web service
     Loaded: loaded (/etc/systemd/system/myapp.service; enabled; vendor preset: enabled)
     Active: active (running) since Mon 2026-04-28 10:23:01 UTC; 2min ago
   Main PID: 12345 (bunpy)
      Tasks: 4 (limit: 4915)
     Memory: 48.2M
        CPU: 1.234s
     CGroup: /system.slice/myapp.service
             └─12345 /home/appuser/.bunpy/bin/bunpy server.py
```


## Tailing logs

systemd sends stdout and stderr to journald. Read logs with `journalctl`:

```bash
# Follow live logs
sudo journalctl -u myapp -f

# Show logs from the last hour
sudo journalctl -u myapp --since "1 hour ago"

# Show the last 100 lines
sudo journalctl -u myapp -n 100

# Show logs for a specific time range
sudo journalctl -u myapp --since "2026-04-28 10:00" --until "2026-04-28 11:00"

# Output in JSON for log aggregation
sudo journalctl -u myapp -o json | jq '{ts: .REALTIME_TIMESTAMP, msg: .MESSAGE}'
```

To forward logs to an external service (Datadog, Loki, etc.), configure a systemd-journal remote exporter or use a log shipper like Vector or Promtail.


## Deployment via systemctl

For a code update:

```bash
# On the server, as your deploy user
cd /opt/myapp
git pull origin main
/home/appuser/.bunpy/bin/bunpy install --frozen
sudo systemctl restart myapp
sudo systemctl status myapp
```

For a zero-downtime update, deploy behind a reverse proxy (nginx or Caddy) and start the new process on a different port before switching the proxy upstream.


## Socket activation (bonus)

Socket activation allows systemd to listen on the port and hand the socket to your app when the first connection arrives. This lets the app start lazily and means no connections are refused during restart - systemd holds the socket open while the app is starting.

Create `/etc/systemd/system/myapp.socket`:

```ini
[Unit]
Description=myapp socket

[Socket]
ListenStream=8080
Accept=no

[Install]
WantedBy=sockets.target
```

Update the unit file to accept the socket from systemd:

```ini
[Unit]
Description=myapp - bunpy web service
Requires=myapp.socket
After=myapp.socket

[Service]
Type=simple
User=appuser
Group=appuser
WorkingDirectory=/opt/myapp
EnvironmentFile=/etc/myapp/environment
ExecStart=/home/appuser/.bunpy/bin/bunpy server.py
Restart=on-failure
StandardOutput=journal
StandardError=journal
SyslogIdentifier=myapp

[Install]
WantedBy=multi-user.target
```

Enable both:

```bash
sudo systemctl enable --now myapp.socket
```

Your app reads the socket from the file descriptor passed by systemd. Most ASGI/WSGI servers support this via the `--fd` flag or the `SD_LISTEN_FDS` environment variable. For a raw Python HTTP server, use the `socket` module to accept the pre-bound socket.


## Common troubleshooting

**Service fails to start:**

```bash
sudo systemctl status myapp
sudo journalctl -u myapp -n 50 --no-pager
```

Look for the exit code and the last few log lines. Common causes: wrong path to bunpy, missing environment variable, port already in use.

**`bunpy: command not found`:**

The `ExecStart` line must use the full path to the bunpy binary. Find it with:

```bash
sudo -u appuser which bunpy
# or
ls /home/appuser/.bunpy/bin/bunpy
```

**App restarts in a loop:**

Check `StartLimitBurst`. If the service hits the limit, systemd stops restarting. Run `sudo systemctl reset-failed myapp` to clear the counter, then `sudo systemctl start myapp` to try again.
