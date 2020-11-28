package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/creack/pty"
	"github.com/gofrs/flock"
	log "github.com/sirupsen/logrus"
	"github.com/zmwangx/mDNSSupervisor/internal/aggregator"
)

const (
	_progName         = "mDNSSupervisor"
	_defaultPattern   = `push-apple\.com\.akadns\.net`
	_defaultInterval  = 15
	_defaultThreshold = 100
)

var (
	_pattern      string
	_interval     int
	_threshold    int
	_debug        bool
	_devMode      bool
	_re           *regexp.Regexp
	_lockPath     = "/var/run/" + _progName + ".lock"
	_databasePath = "/var/log/" + _progName + ".db"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	flag.StringVar(&_pattern, "pattern", _defaultPattern, "regexp pattern to watch for in tcpdump output")
	flag.IntVar(&_interval, "interval", _defaultInterval, "interval in seconds for rolling average")
	flag.IntVar(&_threshold, "threshold", _defaultThreshold, "query per second threshold for restarting mDNSResponder")
	flag.BoolVar(&_debug, "debug", false, "turn on debug logging")
	flag.BoolVar(&_devMode, "dev", false, "reserved for development purposes")
}

func monitorTcpdump(ra *aggregator.RollingAggregator, sa *aggregator.StaticAggregator) {
	// We have to use a pty because tcpdump doesn't do line buffering when
	// executed through exec.Command, and macOS doesn't have stdbuf by default.
	// (And we can't inject a setvbuf into tcpdump from within Go; AKAICT stdbuf
	// achieves what it does by injecting a libstdbuf.so into the target
	// process.)
	cmd := exec.Command("/usr/sbin/tcpdump", "port", "53", "-tt", "-k", "NP")
	// Must use pty.StartWithAttrs here since there's no controlling terminal
	// when launchd runs the program.
	ptmx, err := pty.StartWithAttrs(cmd, &pty.Winsize{Rows: 24, Cols: 80}, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = ptmx.Close() }()
	log.Info("monitoring started")
	scanner := bufio.NewScanner(ptmx)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "tcpdump:") {
			// This prefix indicates a log message intended for stderr
			fmt.Fprintln(os.Stderr, line)
		} else {
			if m := _re.FindStringSubmatch(line); m != nil {
				timestamp, err := strconv.ParseInt(m[1], 10, 64)
				if err != nil {
					log.Fatalf("%s: %s", m[1], err)
				}
				ra.Send(timestamp)
				sa.Send(timestamp)
			}
		}
	}
	if err := cmd.Wait(); err != nil {
		log.Fatalf("tcpdump failed: %s", err)
	}
}

func printMDNSResponderStats() {
	cmd := exec.Command("/bin/zsh", "-c",
		`/bin/ps -o pid,%cpu,rss,etime,command -p "$(/usr/bin/pgrep mDNSResponder | /usr/bin/tr '\n' , )"`)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func restartMDNSResponder() {
	cmd := exec.Command("/usr/bin/killall", "mDNSResponder")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("killall mDNSResponder failed: %s", err)
	} else {
		log.Infof("mDNSResponder restarted")
	}
}

func main() {
	flag.Parse()
	if _debug {
		log.SetLevel(log.DebugLevel)
	}
	if _interval <= 0 {
		log.Fatalf("invalid interval %d: should be positive", _interval)
	}
	if _threshold <= 0 {
		log.Fatalf("invalid threshold %d: should be positive", _threshold)
	}
	pattern := `^` +
		// Epoch timestamp (millisecond precesion) through -tt of tcpdump
		`(?P<timestamp>\d+)\.\d{6} ` +
		// Process name through -k N of tcpdump
		`.*proc mDNSResponder.*` +
		// Wrap user-supplied regexp in a non-capturing group so that it doesn't
		// accidentally modify the previous part.
		`(?:` + _pattern + `)`
	var err error
	_re, err = regexp.Compile(pattern)
	if err != nil {
		log.Fatalf("failed to compile regexp %s: %s", pattern, err)
	}
	if _devMode {
		// Change database and lock path to allow a separate development
		// instance without tainting production data.
		_lockPath = "/tmp/" + _progName + ".lock"
		_databasePath = "/tmp/" + _progName + ".db"
	}

	// Make sure only one instance is running.
	lock := flock.New(_lockPath)
	ok, err := lock.TryLock()
	if err != nil {
		log.Fatalf("failed to acquire lock on %s: %s", _lockPath, err)
	}
	if !ok {
		log.Fatalf("another instance of %s already running", _progName)
	}
	defer lock.Unlock()

	ra := aggregator.NewRollingAggregator(uint(_interval))
	sa := aggregator.NewStaticAggregator(60)
	sl := newStatsLogger()
	go ra.Process(uint(_threshold), func(timestamp int64, rollingAverage float64) {
		printMDNSResponderStats()
		restartMDNSResponder()
	})
	go sa.Process(func(timestamp int64, aggregate uint) {
		sl.log(timestamp, aggregate)
	})
	monitorTcpdump(ra, sa)
}
