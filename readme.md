# bmonitor

Monitors builders via rpc and detects problems like:

- Builder has no external peers.
- Builder missing a transaction in its txpool that other builders have.
- Builder has nonce gap(s) in its txpool (e.g. there are nonces 1, 2, 4, 5
  from the same address, meaning that 4 and 5 can not be included b/c of the
  missing 3).

## TL;DR

```shell
go run github.com/flashbots/bmonitor/cmd \
    -log-level debug \
  serve \
    --monitor-builders builder-0=http://127.0.0.1:8645,builder-1=http://127.0.0.1:8646,builder-2=http://127.0.0.1:8647
```

```shell
curl -sS 127.0.0.1:8080/metrics | grep -v "^#.*$" | sort -u | grep bmonitor
```

```text
bmonitor_peers_count{builder="builder-0",type="external"} 1
bmonitor_peers_count{builder="builder-0",type="internal"} 4
bmonitor_peers_count{builder="builder-0",type="loopback"} 0
bmonitor_peers_count{builder="builder-1",type="external"} 1
bmonitor_peers_count{builder="builder-1",type="internal"} 4
bmonitor_peers_count{builder="builder-1",type="loopback"} 0
bmonitor_peers_count{builder="builder-2",type="external"} 1
bmonitor_peers_count{builder="builder-2",type="internal"} 4
bmonitor_peers_count{builder="builder-2",type="loopback"} 0
bmonitor_txpool_missing_tx_count{builder="builder-0"} 0
bmonitor_txpool_missing_tx_count{builder="builder-1"} 0
bmonitor_txpool_missing_tx_count{builder="builder-2"} 0
bmonitor_txpool_nonce_gap_length{builder="builder-0"} 0
bmonitor_txpool_nonce_gap_length{builder="builder-1"} 0
bmonitor_txpool_nonce_gap_length{builder="builder-2"} 0
```

## Usage

```text
NAME:
   bmonitor serve - run bmonitor server

USAGE:
   bmonitor serve [command options]

OPTIONS:
   MONITOR

   --monitor-builders name=url [ --monitor-builders name=url ]  list of monitored builder rpc endpoints in the format name=url [$BMONITOR_MONITOR_BUILDERS]
   --monitor-interval interval                                  interval at which to query builders for their status (default: 5s) [$BMONITOR_MONITOR_INTERVAL]
   --monitor-timeout duration                                   timeout duration for rpc queries (default: 500ms) [$BMONITOR_MONITOR_TIMEOUT]

   SERVER

   --server-listen-address host:port  host:port for the server to listen on (default: "0.0.0.0:8080") [$BMONITOR_SERVER_LISTEN_ADDRESS]
```
