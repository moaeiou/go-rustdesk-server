package transport

import "io"

func (w *Writer) Copy(destination *Writer) {
	if w._type != tcp || w.tConn == nil || destination == nil || destination._type != tcp || destination.tConn == nil {
		return
	}
	done := make(chan struct{}, 2)
	copyStream := func(dst, src io.ReadWriteCloser) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go copyStream(destination.tConn, w.tConn)
	go copyStream(w.tConn, destination.tConn)
	<-done
	// Closing both sides unblocks the other copy immediately and prevents a
	// half-closed client from retaining a relay goroutine indefinitely.
	w.Close()
	destination.Close()
	<-done
}
