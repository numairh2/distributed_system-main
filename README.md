# Distributed Membership & Log Query System

Two cooperating services in one Go binary:

- **Membership / failure detection.** An implementation of the
  [SWIM protocol](https://www.cs.cornell.edu/projects/Quicksilver/public_pdfs/SWIM.pdf):
  randomized probing with indirect ping-req, a suspicion mechanism refuted
  through incarnation numbers, and piggybacked dissemination of membership updates.
- **Distributed log querying.** A grep over the log files of every live member.
  Queries go to whoever the membership layer currently sees as alive, and
  unreachable nodes are reported rather than failing the query.

Originally built for CS 425 (MP1 + MP2) on ten course VMs (`fa24-cs425-35XX.cs.illinois.edu`).
It now runs on any machines that can reach each other over UDP. The default
testbed is a set of Docker containers on your laptop. The original coursework
is preserved under [`legacy/`](legacy/).

## Layout

```
cmd/node/            daemon + CLI
internal/swim/       protocol implementation and tests
internal/logquery/   distributed grep: per-node search server and fan-out client
internal/transport/  UDP transport, plus an in-memory network for tests
scripts/             Docker testbed, failure injection, end-to-end tests
site/                static interactive demo (deployed with Vercel)
legacy/              original MP1 (log querier) and MP2 (membership) code
```

## Quick start (Docker)

Requires Docker.

```sh
./scripts/start-testbed.sh 5              # node1..node5, each one stands in for a VM
docker exec node1 node ctl list_mem       # view membership from node1
docker exec node1 node grep -c ERROR      # count matches on every live node
docker logs -f node1                      # watch joins / suspicions / failures

./scripts/inject-failure.sh crash 4       # SIGKILL: detected as a failure
./scripts/inject-failure.sh restart 4     # comes back, refutes with a higher incarnation
docker stop node5                         # SIGTERM: graceful leave
./scripts/inject-failure.sh loss 3 30     # 30% packet loss on node3 (also: delay 3 200)
./scripts/inject-failure.sh heal 3

./scripts/stop-testbed.sh
```

Extra daemon flags can be passed to every node, e.g.
`SWIM_ARGS="-suspicion=false -period=500ms" ./scripts/start-testbed.sh`.

## Commands

Run `docker exec nodeN node ctl <command>`, or type commands directly when
running the daemon in a terminal:

| Command | Effect |
|---|---|
| `list_mem` | all known members with state and incarnation |
| `list_self` | this node's address, state, incarnation |
| `join <host:port>` | contact another introducer |
| `enable_sus` / `disable_sus` / `check_sus` | toggle or show the suspicion mechanism |
| `leave` / `exit` | leave the group gracefully and shut down |

## Querying logs

Each node serves the files matching `-logs` (default `logs/*.log`). In the
testbed that is `testbed-logs/nodeN/` on your machine, mounted at `/logs`. A demo
log is generated for any node with no `.log` files, so to query real data
(e.g. the course's `vmN.log` files) put them in those directories before starting.

```sh
docker exec node1 node grep 'POST /login.* 500'        # matching lines as node:file:line:text
docker exec node1 node grep -ic 'timeout|refused'       # per-node counts + total
docker exec node1 node grep --out result -w DELETE      # write result/output.<node>.txt, like MP1
```

Supported flags: `-i -v -F -E -w -x -c`, `-e PATTERN`. Patterns are Go regular
expressions (RE2, equivalent to `grep -E` for everyday use). Searching is done
in-process, so no shell or `grep` binary is involved. Per-node counts and the
total are printed after the matches (on stderr, so stdout can be piped).

## Running on real machines (cloud VMs etc.)

Open UDP port 7946 between the hosts, then on each:

```sh
go build -o node ./cmd/node
./node -addr <this-host>:7946 -seeds <host-a>:7946,<host-b>:7946 -logs '/var/log/myapp/*.log'
```

Also open TCP port 7948 between hosts for log queries. Use the same `-query` port
on every node, and don't expose it to the internet.

Flags: `-period` (default 1s), `-ack-timeout` (400ms), `-suspicion-mult`
(periods before a suspect is declared dead, default 5), `-suspicion`, `-bind`,
`-admin`, `-query` (`:7948`), `-logs`, `-query-timeout` (5s).

## Tests

```sh
go test -race ./...
```

End-to-end, against a real 5-node Docker cluster (about 45 seconds, Docker required):

```sh
./scripts/e2e-test.sh              # unit tests + cluster scenarios
./scripts/e2e-test.sh --skip-unit  # cluster scenarios only
./scripts/e2e-test.sh --keep       # leave the cluster up afterwards to inspect
```

It checks convergence, distributed grep results against fixture logs with known
counts, crash detection, restart and refutation, graceful leave, tolerance
of 30% packet loss, and the suspicion toggle. It replaces any running testbed
and exits non-zero on failure, so it can run in CI.

`internal/logquery` includes a port of MP1's `distributed_test.py`: ten
in-process machines with generated logs, checked for exact per-machine counts.
The protocol tests run whole clusters over an in-memory network that can drop
packets, cut individual links, and isolate nodes.

## Demo site

`site/index.html` is a self-contained interactive demo: a JavaScript port of the
protocol you can crash nodes in, plus a simulated distributed grep. Preview it
locally with `python3 -m http.server -d site 8000` and open http://localhost:8000.
`vercel.json` deploys that folder as a static site (no build step).

## Reports

[MP1 Report Doc](https://docs.google.com/document/d/1Z9jDQY07XTp9Nh0F0Xh1Il7eV4QC_AWjnXIp48rGRqg/edit?usp=sharing) ·
[MP2 Report Doc](https://docs.google.com/document/d/1AqJVFc-fNjXq7zUwm4aICjH8nfH83K5fQ8scUuEAK90/edit?usp=sharing)
