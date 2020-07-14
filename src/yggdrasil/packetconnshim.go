package yggdrasil

import (
	"net"
	"time"

	"github.com/Arceliar/phony"
)

type PacketConnShim struct {
	phony.Inbox
	core       *Core
	packetConn net.PacketConn
}

func NewPacketConnShim(core *Core) *PacketConnShim {
	return &PacketConnShim{
		core:       core,
		packetConn: core.PacketConn,
	}
}

func (c *PacketConnShim) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	_, _, err = c.packetConn.ReadFrom(b)
	if err != nil {
		return
	}
	var coords []byte
	wire_chop_coords(&coords, &b)
	addr = Coords(wire_coordsBytestoUint64s(coords))
	n = len(b)
	return
}

func (c *PacketConnShim) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	return c.packetConn.WriteTo(wire_put_coords(wire_coordsUint64stoBytes(c.core.Coords()), b), addr)
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
