---
icon: brands/grafana
---

# Grafana

Check out [using Grafana with TimescaleDB](db/timescaledb#using-grafana-with-timescaledb).

!!! warning "Documentation In Progress"

    This documentation is incomplete while we are migrating from our internal documentation store to the public GitHub.

## Installing the Hermes data source plugin

The Hermes data source plugin is distributed as a release asset on GitHub. An
install script is provided that downloads the latest published release and
extracts it into your Grafana plugins directory.

!!! note "Prerequisites"

    - [`jq`](https://jqlang.github.io/jq/) must be installed.
    - The plugin is **unsigned**, so Grafana must be configured to allow it (see below).

### Using the install script

Run the installer directly from the repository:

```bash
curl -fsSL https://raw.githubusercontent.com/nasa/hermes/main/nasa-hermes-datasource/install.sh | bash
```

The script auto-detects a Grafana plugins directory. To install into a specific
directory, pass it as an argument (everything after `bash -s --` is passed to the
script):

```bash
curl -fsSL https://raw.githubusercontent.com/nasa/hermes/main/nasa-hermes-datasource/install.sh \
  | bash -s -- /var/lib/grafana/plugins
```

The plugin is installed to `<plugins-dir>/nasa-hermes-datasource/`.

### Allowing the unsigned plugin

Grafana will not load the plugin unless it is explicitly allowed. Either set the
environment variable:

```bash
GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=nasa-hermes-datasource
```

or add the following to `grafana.ini` under `[plugins]`:

```ini
[plugins]
allow_loading_unsigned_plugins = nasa-hermes-datasource
```

Then restart Grafana.

### Running with Docker

To try the plugin in an isolated Grafana instance:

```bash
# Install the plugin into a host folder
mkdir -p ~/grafana-plugins
curl -fsSL https://raw.githubusercontent.com/nasa/hermes/main/nasa-hermes-datasource/install.sh \
  | bash -s -- ~/grafana-plugins

# Start Grafana with the plugin mounted and allowed
docker run -d -p 3000:3000 \
  -e GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=nasa-hermes-datasource \
  -v ~/grafana-plugins:/var/lib/grafana/plugins \
  --name grafana grafana/grafana:latest
```

Open [http://localhost:3000](http://localhost:3000) (default login `admin` /
`admin`), then go to **Connections → Data sources → Add data source** and search
for **hermes**.

To verify the plugin loaded, navigate to **Administration → Plugins and data →
Plugins** and search for **hermes**. The **Hermes** data source should appear in
the list, marked as an unsigned plugin. If it does not appear, confirm that the
plugin was extracted into the correct plugins directory and that unsigned plugins
are allowed (see [Allowing the unsigned plugin](#allowing-the-unsigned-plugin)).
