package yggdrasil

import (
	"errors"
	"net"
	"time"

	"github.com/Arceliar/phony"
)

type PacketConnShim struct {
	phony.Inbox
	core       *Core
	packetConn net.PacketConn
}

func newPacketConnShim(core *Core) *PacketConnShim {
	return &PacketConnShim{
		core:       core,
		packetConn: core.packetConn,
	}
}

func (c *PacketConnShim) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	buf := make([]byte, 65535)
	n, _, err = c.packetConn.ReadFrom(buf)
	if err != nil {
		return
	}
	var cbytes []byte
	if !wire_chop_coords(&cbytes, &buf) {
		err = errors.New("failed to chop coords")
		return
	}
	copy(b, buf[:])
	n -= len(cbytes) + 1
	addr = Coords(wire_coordsBytestoUint64s(cbytes))
	return
}

func (c *PacketConnShim) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	coords := wire_coordsUint64stoBytes(c.core.Coords())
	cbytes := wire_put_coords(coords, nil)
	b = append(cbytes, b...)
	n, err = c.packetConn.WriteTo(b, addr)
	n -= len(cbytes)
	return
}

func (c *PacketConnShim) Close() error {
	return c.packetConn.Close()
}

func (c *PacketConnShim) LocalAddr() net.Addr {
	return c.packetConn.LocalAddr()
}

func (c *PacketConnShim) SetDeadline(t time.Time) error {
	return c.packetConn.SetDeadline(t)
}

func (c *PacketConnShim) SetReadDeadline(t time.Time) error {
	return c.packetConn.SetReadDeadline(t)
}

func (c *PacketConnShim) SetWriteDeadline(t time.Time) error {
	return c.packetConn.SetWriteDeadline(t)
}
