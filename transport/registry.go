package transport

import (
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/os/gcache"
	"github.com/gogf/gf/v2/os/gctx"
	"time"
)

var (
	ctx          = gctx.New()
	cache        = gcache.New()
	cacheTimeOut = time.Minute
)

func GetWriter(key, writerType string) (*Writer, error) {
	value, _ := cache.Get(ctx, writerType+key)
	if value != nil {
		if writer, ok := value.Val().(*Writer); ok {
			return writer, nil
		}
	}
	return nil, errors.New("OFFLINE")
}

func addWriter(key, writerType string, writer *Writer) {
	timeout := cacheTimeOut
	if writerType == tcp {
		timeout = 0
	}
	if err := cache.Set(ctx, fmt.Sprint(writerType, key), writer, timeout); err != nil {
		Info("%v", err)
	}
}

func RemoveWriter(writer *Writer) { writer.remove() }

func (w *Writer) remove() {
	w.mu.Lock()
	key := w.key
	w.mu.Unlock()
	if w._type == udp {
		_, _ = cache.Remove(ctx, udp+w.addr.String())
	}
	if w._type == tcp {
		_, _ = cache.Remove(ctx, tcp+w.tConn.RemoteAddr().String())
	}
	if key != "" {
		_, _ = cache.Remove(ctx, udp+key)
		_, _ = cache.Remove(ctx, tcp+key)
	}
}
