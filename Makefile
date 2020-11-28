.PHONY: all install clean

all: mDNSSupervisor

mDNSSupervisor:
	go build -ldflags="-s -w"

install:
	mkdir -p /usr/local/sbin
	/usr/bin/install -m 755 mDNSSupervisor /usr/local/sbin
	/usr/bin/install -m 644 launchd/org.zhimingwang.mDNSSupervisor.plist /Library/LaunchDaemons
	launchctl load /Library/LaunchDaemons/org.zhimingwang.mDNSSupervisor.plist

clean:
	@$(RM) mDNSSupervisor
