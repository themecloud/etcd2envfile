# etcd2envfile

`etcd2envfile` provide an easy way to populate environment file from an etcd backend.

It's particulary useful when using [systemd](http://www.freedesktop.org/wiki/Software/systemd/) `EnvironmentFile` or [docker](https://www.docker.com/) `--env-file`

## Usage

| Option | Default | Description |
| ------ | ------- | ----------- |
| `-etcd` | `http://127.0.0.1:2379` | Specifies the etcd endpoint |
| `-outputDir` | `/run/conf` | Specifies the output dir |
| `-etcdPrefix` | | Specifies the etcd prefix |
