package transport

import (
	"io"
	"net/netip"
	"sync"
	"time"
)

func addressListed(address string, entries []string) bool {
	host, _ := GetHostPort(address)
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	ip = ip.Unmap()
	for _, entry := range entries {
		if addr, err := netip.ParseAddr(entry); err == nil && addr.Unmap() == ip {
			return true
		}
		if prefix, err := netip.ParsePrefix(entry); err == nil && prefix.Contains(ip) {
			return true
		}
	}
	return false
}

// BlockedRelayAddress accepts IPs and CIDRs, including mapped IPv4 peers.
func BlockedRelayAddress(address string) bool { return addressListed(address, Config.Blocklist) }

// Token buckets allow at most one 32 KiB chunk of idle credit. Cancelled
// transfers never leave reservations that delay future sessions.
type bandwidthBudget struct {
	mu     sync.Mutex
	last   time.Time
	tokens float64
}

var totalRelayBudget bandwidthBudget

func (b *bandwidthBudget) wait(size int, mbps int64, stop <-chan struct{}) error {
	if mbps == 0 {
		return nil
	}
	rate := float64(mbps) * 1000000 / 8
	for size > 0 {
		chunk := min(size, 32*1024)
		for {
			select {
			case <-stop:
				return io.ErrClosedPipe
			default:
			}
			b.mu.Lock()
			now := time.Now()
			if b.last.IsZero() {
				b.last = now
				b.tokens = 32 * 1024
			}
			b.tokens = min(float64(32*1024), b.tokens+now.Sub(b.last).Seconds()*rate)
			b.last = now
			if b.tokens >= float64(chunk) {
				b.tokens -= float64(chunk)
				b.mu.Unlock()
				break
			}
			delay := time.Duration((float64(chunk) - b.tokens) / rate * float64(time.Second))
			b.mu.Unlock()
			timer := time.NewTimer(max(delay, time.Nanosecond))
			select {
			case <-timer.C:
			case <-stop:
				timer.Stop()
				return io.ErrClosedPipe
			}
		}
		size -= chunk
	}
	return nil
}

type relayBudget struct {
	single      bandwidthBudget
	stop        chan struct{}
	mu          sync.Mutex
	started     time.Time
	bytes       int64
	downgraded  bool
	blacklisted bool
}

func newRelayBudget(peers ...*Writer) *relayBudget {
	b := &relayBudget{stop: make(chan struct{}), started: time.Now()}
	for _, peer := range peers {
		if addressListed(peer.GetAddrStr(), Config.Blacklist) {
			b.blacklisted = true
		}
	}
	return b
}
func (b *relayBudget) speed(now time.Time, size int) int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.bytes += int64(size)
	elapsed := now.Sub(b.started).Seconds()
	if Config.SingleBandwidth > 0 && Config.DowngradeAfter > 0 && elapsed >= float64(Config.DowngradeAfter) && float64(b.bytes)*8/elapsed > float64(Config.SingleBandwidth)*1000000*Config.DowngradeThreshold {
		b.downgraded = true
	}
	if b.blacklisted || b.downgraded {
		return Config.LimitSpeed
	}
	return Config.SingleBandwidth
}
func (b *relayBudget) wait(size int) error {
	if err := b.single.wait(size, b.speed(time.Now(), size), b.stop); err != nil {
		return err
	}
	return totalRelayBudget.wait(size, Config.TotalBandwidth, b.stop)
}

type pacedWriter struct {
	dst    io.Writer
	budget *relayBudget
}

func (w pacedWriter) Write(data []byte) (int, error) {
	total := 0
	for len(data) > 0 {
		n := min(len(data), 32*1024)
		if err := w.budget.wait(n); err != nil {
			return total, err
		}
		written, err := writeAll(w.dst, data[:n])
		total += written
		if err != nil {
			return total, err
		}
		data = data[n:]
	}
	return total, nil
}
