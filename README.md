# mDNSSupervisor

mDNSSupervisor supervises mDNSResponder on macOS, and restarts it when it appears to be going beserk (passing a certain threshold of queries per second matching a pattern). Despite its name, it only supervises unicast DNS on port 53; multicast DNS is out of scope.

mDNSSupervisor is my answer to a [system bug](https://apple.stackexchange.com/q/406617/37762) that keeps creeping up on me on macOS 11.0 Big Sur.

By default, it restarts mDNSResponder whenever the rolling average of queries per second matching `push-apple.com.akadns.net` over the last 15 seconds surpasses 100 q/s. The pattern, the rolling average interval and the threshold are all customizable.

It also logs per-minute stats to a SQLite database `/var/log/mDNSSupervisor.db` for potential further analysis.

## Installation

From the repo, with the Go toolchain:

```
make
sudo make install
```

Directly downloading artifacts:

```
sudo mkdir -p /usr/local/sbin
sudo curl -Lo /usr/local/sbin/mDNSSupervisor https://github.com/zmwangx/mDNSSupervisor/releases/download/v0.1/mDNSSupervisor
sudo curl -Lo /Library/LaunchDaemons/org.zhimingwang.mDNSSupervisor.plist https://raw.githubusercontent.com/zmwangx/mDNSSupervisor/master/launchd/org.zhimingwang.mDNSSupervisor.plist
sudo launchctl load /Library/LaunchDaemons/org.zhimingwang.mDNSSupervisor.plist
```

## Usage

mDNSSupervisor must be run as root.

A launchd daemon is provided and is the intended way of using mDNSSupervisor. One may customize its behavior by adding `ProgramArguments` to the launchd daemon plist:

```console
$ mDNSSupervisor -h
Usage of mDNSSupervisor:
  -debug
    	turn on debug logging
  -dev
    	reserved for development purposes
  -interval int
    	interval in seconds for rolling average (default 15)
  -pattern string
    	regexp pattern to watch for in tcpdump output (default "push-apple\\.com\\.akadns\\.net")
  -threshold int
    	query per second threshold for restarting mDNSResponder (default 100)
```

You can find the logs at `/var/log/mDNSSupervisor.log` (this is the general purpose textual log separate from the per-minute stats log `/var/log/mDNSSupervisor.db`).

## License

mDNSSupervisor is provided to you as is, without warranty of any kind, under WTFPL. Use at your own risk, modify and redistribute at your own pleasure.
