package transport

import (
	"io"
	"time"
)

type copyResult struct {
	bytes int64
	err   error
}

func (w *Writer) Copy(destination *Writer) {
	if w._type != tcp || w.tConn == nil || destination == nil || destination._type != tcp || destination.tConn == nil {
		return
	}
	started := time.Now()
	done := make(chan copyResult, 2)
	copyStream := func(dst, src io.ReadWriteCloser, pending []byte) {
		var total int64
		if len(pending) > 0 {
			n, err := writeAll(dst, pending)
			total += int64(n)
			if err != nil {
				done <- copyResult{bytes: total, err: err}
				return
			}
		}
		n, err := io.Copy(dst, src)
		done <- copyResult{bytes: total + n, err: err}
	}
	go copyStream(destination.tConn, w.tConn, w.takePending())
	go copyStream(w.tConn, destination.tConn, destination.takePending())
	first := <-done
	// Closing both sides unblocks the other copy immediately and prevents a
	// half-closed client from retaining a relay goroutine indefinitely.
	w.Close()
	destination.Close()
	second := <-done
	Info("RELAY stream closed duration=%s bytes=%d/%d first_error=%v", time.Since(started).Round(time.Millisecond), first.bytes, second.bytes, first.err)
}

func writeAll(writer io.Writer, data []byte) (int, error) {
	written := 0
	for written < len(data) {
		n, err := writer.Write(data[written:])
		written += n
		if err != nil {
			return written, err
		}
		if n == 0 {
			return written, io.ErrShortWrite
		}
	}
	return written, nil
}
