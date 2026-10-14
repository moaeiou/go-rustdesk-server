package transport

import "io"

func (w *Writer) Copy(destination *Writer) {
	if w._type != tcp || destination == nil || destination.tConn == nil {
		return
	}
	defer destination.Close()
	defer w.Close()
	go io.Copy(destination.tConn, w.tConn)
	_, _ = io.Copy(w.tConn, destination.tConn)
}
