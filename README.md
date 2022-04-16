# Tier2Pool

![GitHub last commit](https://img.shields.io/github/last-commit/tier2pool/tier2pool?style=flat-square)
![GitHub commit activity](https://img.shields.io/github/commit-activity/m/tier2pool/tier2pool?style=flat-square)
![GitHub license](https://img.shields.io/github/license/tier2pool/tier2pool?style=flat-square)

## Introduction

High performance cryptocurrency mining pool tunnel, it can easily carry TH/s. ETH and ETC and XMR are supported.

## Usage

```shell
cp ./server.yaml /etc/tier2pool/server.yaml
vim /etc/tier2pool/server.yaml
make build && cd ./build
./tier2pool_linux_amd64 server
```

## TODO

- [ ] Stratum protocol
    - [x] NiceHash
    - [ ] OpenPool
- [x] Rename worker
- [x] Custom fee percentage
- [x] TLS support
- [ ] Network firewall
    - [ ] Request and bandwidth rate limiter
    - [ ] Block or allow list
- [ ] Monitor Dashboard (Grafana + Prometheus + Timescale)
- [ ] More mining protocols
    - [x] ETH
    - [x] ETC
    - [x] XMR
    - [ ] BTC
    - [ ] LTC
    - [ ] TON
- [ ] Better fee algorithm
- [ ] Refactor extractor with channel

## Sponsor me

### ETH

```diff
+ 0x000000A52a03835517E9d193B3c27626e1Bc96b1
```

### XMR

```diff
+ 84TZwzCfHhkZ43JzygNqaN5ke6t3uRSD32rofAhV19jB1VNzDnkaciWN7c7tfqFvKt95f4Y6jyEecWzsnUHi1koZNqBveJb
```

## LICENSE

[BSD 3-Clause Clear License](LICENSE)
