package yggdrasil

import (
	"errors"
	"net"
	"time"

	"github.com/Arceliar/phony"
)

type PacketConn struct {
	phony.Inbox
	net.PacketConn
	core          *Core
	closed        bool
	readCallback  func(net.Addr, []byte)
	readBuffer    chan wire_trafficPacket
	readDeadline  *time.Time
	writeDeadline *time.Time
}

func newPacketConn(c *Core) *PacketConn {
	return &PacketConn{
		core:       c,
		readBuffer: make(chan wire_trafficPacket),
	}
}

func (c *PacketConn) _sendToReader(addr net.Addr, packet wire_trafficPacket) {
	if c.readCallback == nil {
		c.readBuffer <- packet
	} else {
		c.readCallback(addr, packet.Payload)
	}
}

// implements net.PacketConn
func (c *PacketConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	if c.readCallback != nil {
		return 0, nil, errors.New("read callback is configured")
	}
	if c.closed { // TODO: unsafe?
		return 0, nil, PacketConnError{closed: true}
	}
	packet := <-c.readBuffer
	coords := wire_coordsBytestoUint64s(packet.Coords)
	copy(b, packet.Payload)
	return len(packet.Payload), Coords(coords), nil
}

// implements net.PacketConn
func (c *PacketConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	if c.closed { // TODO: unsafe?
		return 0, PacketConnError{closed: true}
	}

	// Make sure that the net.Addr we were given was actually a
	// *crypto.BoxPubKey. If it wasn't then fail.
	coords, ok := addr.(*Coords)
	if !ok {
		return 0, errors.New("expected *yggdrasil.Coords as net.Addr")
	}

	// Create the packet.
	packet := &wire_trafficPacket{
		Coords:  wire_coordsUint64stoBytes(*coords),
		Payload: b,
	}

	// Send it to the router.
	c.core.router.out(packet.encode())

	// Wait for the checks to pass. Then return the success
	// values to the caller.
	//err = <-sendErr
	return len(b), err
}

// implements net.PacketConn
func (c *PacketConn) Close() error {
	// TODO: implement this. don't know what makes sense for net.PacketConn?
	return nil
}

// implements net.PacketConn
func (c *PacketConn) LocalAddr() net.Addr {
	return &c.core.boxPub
}

// SetReadCallback allows you to specify a function that will be called whenever
// a packet is received. This should be used if you wish to implement
// asynchronous patterns for receiving data from the remote node.
//
// Note that if a read callback has been supplied, you should no longer attempt
// to use the synchronous Read function.
func (c *PacketConn) SetReadCallback(callback func(net.Addr, []byte)) {
	c.Act(nil, func() {
		c.readCallback = callback
		c._drainReadBuffer()
	})
}

func (c *PacketConn) _drainReadBuffer() {
	if c.readCallback == nil {
		return
	}
	select {
	case bs := <-c.readBuffer:
		coords := Coords(wire_coordsBytestoUint64s(bs.Coords))
		c.readCallback(&coords, bs.Payload)
		c.Act(nil, c._drainReadBuffer) // In case there's more
	default:
	}
}

// SetDeadline is equivalent to calling both SetReadDeadline and
// SetWriteDeadline with the same value, configuring the maximum amount of time
// that synchronous Read and Write operations can block for. If no deadline is
// configured, Read and Write operations can potentially block indefinitely.
func (c *PacketConn) SetDeadline(t time.Time) error {
	c.SetReadDeadline(t)
	c.SetWriteDeadline(t)
	return nil
}

// SetReadDeadline configures the maximum amount of time that a synchronous Read
// operation can block for. A Read operation will unblock at the point that the
// read deadline is reached if no other condition (such as data arrival or
// connection closure) happens first. If no deadline is configured, Read
// operations can potentially block indefinitely.
func (c *PacketConn) SetReadDeadline(t time.Time) error {
	// TODO warn that this can block while waiting for the Conn actor to run, so don't call it from other actors...
	phony.Block(c, func() { c.readDeadline = &t })
	return nil
}

// SetWriteDeadline configures the maximum amount of time that a synchronous
// Write operation can block for. A Write operation will unblock at the point
// that the read deadline is reached if no other condition (such as data sending
// or connection closure) happens first. If no deadline is configured, Write
// operations can potentially block indefinitely.
func (c *PacketConn) SetWriteDeadline(t time.Time) error {
	// TODO warn that this can block while waiting for the Conn actor to run, so don't call it from other actors...
	phony.Block(c, func() { c.writeDeadline = &t })
	return nil
}

// PacketConnError implements the net.Error interface
type PacketConnError struct {
	error
	timeout   bool
	temporary bool
	closed    bool
	maxsize   int
}

// Timeout returns true if the error relates to a timeout condition on the
// connection.
func (e *PacketConnError) Timeout() bool {
	return e.timeout
}

// Temporary return true if the error is temporary or false if it is a permanent
// error condition.
func (e *PacketConnError) Temporary() bool {
	return e.temporary
}

// PacketTooBig returns in response to sending a packet that is too large, and
// if so, the maximum supported packet size that should be used for the
// connection.
func (e *PacketConnError) PacketTooBig() bool {
	return e.maxsize > 0
}

// PacketMaximumSize returns the maximum supported packet size. This will only
// return a non-zero value if ConnError.PacketTooBig() returns true.
func (e *PacketConnError) PacketMaximumSize() int {
	if !e.PacketTooBig() {
		return 0
	}
	return e.maxsize
}

// Closed returns if the session is already closed and is now unusable.
func (e *PacketConnError) Closed() bool {
	return e.closed
}
