package anytls

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	stdnet "net"
	"os"
	"sync"
	"time"

	M "github.com/sagernet/sing/common/metadata"
	"github.com/xtls/xray-core/common/buf"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/singbridge"
	"github.com/xtls/xray-core/transport/pipe"
)

const (
	anyTLSCmdWaste               = 0
	anyTLSCmdSYN                 = 1
	anyTLSCmdPSH                 = 2
	anyTLSCmdFIN                 = 3
	anyTLSCmdSettings            = 4
	anyTLSCmdAlert               = 5
	anyTLSCmdUpdatePaddingScheme = 6
	anyTLSCmdSYNACK              = 7
	anyTLSCmdHeartRequest        = 8
	anyTLSCmdHeartResponse       = 9
	anyTLSCmdServerSettings      = 10

	anyTLSFrameHeaderSize = 7
	anyTLSStreamID        = 1
	anyTLSWriteTimeout    = 5 * time.Second
)

var anyTLSClientSettings = []byte("v=2\nclient=xray-core-laochendeai\npadding-md5=disabled")

type clientStreamConn struct {
	conn     stdnet.Conn
	streamID uint32

	readPipe   *pipe.Reader
	readWriter *pipe.Writer

	readMu       sync.Mutex
	readBuffer   buf.MultiBuffer
	readDeadline time.Time
	readErr      error

	writeMu       sync.Mutex
	writeDeadline time.Time
	writeClosed   bool

	readCloseOnce sync.Once
	transportOnce sync.Once
	closeOnce     sync.Once
}

func newClientStreamConn(serverConn stdnet.Conn, password string, destination xnet.Destination) (*clientStreamConn, error) {
	readPipe, readWriter := pipe.New(pipe.WithoutSizeLimit())
	conn := &clientStreamConn{
		conn:       serverConn,
		streamID:   anyTLSStreamID,
		readPipe:   readPipe,
		readWriter: readWriter,
	}

	if err := conn.writeAuth(password); err != nil {
		_ = serverConn.Close()
		_ = readWriter.Close()
		return nil, err
	}

	if err := conn.writeFrame(anyTLSCmdSettings, 0, anyTLSClientSettings); err != nil {
		_ = serverConn.Close()
		_ = readWriter.Close()
		return nil, err
	}

	if err := conn.writeFrame(anyTLSCmdSYN, conn.streamID, nil); err != nil {
		_ = serverConn.Close()
		_ = readWriter.Close()
		return nil, err
	}

	targetPayload, err := marshalDestination(destination)
	if err != nil {
		_ = serverConn.Close()
		_ = readWriter.Close()
		return nil, err
	}
	if err := conn.writeFrame(anyTLSCmdPSH, conn.streamID, targetPayload); err != nil {
		_ = serverConn.Close()
		_ = readWriter.Close()
		return nil, err
	}

	go conn.recvLoop()

	return conn, nil
}

func marshalDestination(destination xnet.Destination) ([]byte, error) {
	var payload bytes.Buffer
	if err := M.SocksaddrSerializer.WriteAddrPort(&payload, singbridge.ToSocksaddr(destination)); err != nil {
		return nil, fmt.Errorf("encode destination: %w", err)
	}
	return payload.Bytes(), nil
}

func (c *clientStreamConn) writeAuth(password string) error {
	sum := sha256.Sum256([]byte(password))
	payload := make([]byte, sha256.Size+2)
	copy(payload[:sha256.Size], sum[:])
	return c.writeAll(payload)
}

func (c *clientStreamConn) recvLoop() {
	defer c.closeTransport()

	header := make([]byte, anyTLSFrameHeaderSize)

	for {
		if _, err := io.ReadFull(c.conn, header); err != nil {
			if err == io.EOF {
				c.closeRead(nil)
			} else {
				c.closeRead(fmt.Errorf("anytls read frame header: %w", err))
			}
			return
		}

		cmd := header[0]
		streamID := binary.BigEndian.Uint32(header[1:5])
		payloadLen := int(binary.BigEndian.Uint16(header[5:7]))

		var payload []byte
		if payloadLen > 0 {
			payload = make([]byte, payloadLen)
			if _, err := io.ReadFull(c.conn, payload); err != nil {
				c.closeRead(fmt.Errorf("anytls read frame payload: %w", err))
				return
			}
		}

		switch cmd {
		case anyTLSCmdPSH:
			if streamID != c.streamID || payloadLen == 0 {
				continue
			}
			if err := c.readWriter.WriteMultiBuffer(buf.MergeBytes(nil, payload)); err != nil {
				c.closeRead(fmt.Errorf("anytls deliver stream payload: %w", err))
				return
			}
		case anyTLSCmdFIN:
			if streamID == c.streamID {
				c.closeRead(nil)
				return
			}
		case anyTLSCmdSYNACK:
			if streamID == c.streamID && payloadLen > 0 {
				c.closeRead(fmt.Errorf("anytls remote handshake failed: %s", string(payload)))
				return
			}
		case anyTLSCmdAlert:
			if payloadLen == 0 {
				c.closeRead(fmt.Errorf("anytls remote alert"))
			} else {
				c.closeRead(fmt.Errorf("anytls remote alert: %s", string(payload)))
			}
			return
		case anyTLSCmdHeartRequest:
			if err := c.writeFrame(anyTLSCmdHeartResponse, streamID, nil); err != nil {
				c.closeRead(fmt.Errorf("anytls write heartbeat response: %w", err))
				return
			}
		case anyTLSCmdWaste, anyTLSCmdSettings, anyTLSCmdUpdatePaddingScheme, anyTLSCmdHeartResponse, anyTLSCmdServerSettings:
			continue
		default:
			continue
		}
	}
}

func (c *clientStreamConn) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	for {
		c.readMu.Lock()
		if !c.readBuffer.IsEmpty() {
			var n int
			c.readBuffer, n = buf.SplitBytes(c.readBuffer, p)
			if c.readBuffer.IsEmpty() {
				c.readBuffer = nil
			}
			c.readMu.Unlock()
			return n, nil
		}
		deadline := c.readDeadline
		readErr := c.readErr
		c.readMu.Unlock()

		if readErr != nil {
			return 0, readErr
		}

		var (
			mb  buf.MultiBuffer
			err error
		)
		if deadline.IsZero() {
			mb, err = c.readPipe.ReadMultiBuffer()
		} else {
			timeout := time.Until(deadline)
			if timeout <= 0 {
				return 0, os.ErrDeadlineExceeded
			}
			mb, err = c.readPipe.ReadMultiBufferTimeout(timeout)
		}
		if err != nil {
			if err == io.EOF {
				c.readMu.Lock()
				readErr = c.readErr
				c.readMu.Unlock()
				if readErr != nil {
					return 0, readErr
				}
			}
			return 0, err
		}

		c.readMu.Lock()
		c.readBuffer, _ = buf.MergeMulti(c.readBuffer, mb)
		c.readMu.Unlock()
	}
}

func (c *clientStreamConn) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		chunkLen := len(p)
		if chunkLen > math.MaxUint16 {
			chunkLen = math.MaxUint16
		}
		if err := c.writeFrame(anyTLSCmdPSH, c.streamID, p[:chunkLen]); err != nil {
			if total == 0 {
				return 0, err
			}
			return total, err
		}
		total += chunkLen
		p = p[chunkLen:]
	}
	return total, nil
}

func (c *clientStreamConn) Close() error {
	var releaseBuf buf.MultiBuffer
	c.closeOnce.Do(func() {
		c.closeRead(stdnet.ErrClosed)

		c.readMu.Lock()
		releaseBuf = c.readBuffer
		c.readBuffer = nil
		c.readMu.Unlock()
	})
	buf.ReleaseMulti(releaseBuf)
	return c.closeTransport()
}

func (c *clientStreamConn) CloseWrite() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.writeClosed {
		return nil
	}
	c.writeClosed = true
	return c.writeFrameLocked(anyTLSCmdFIN, c.streamID, nil)
}

func (c *clientStreamConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *clientStreamConn) SetReadDeadline(t time.Time) error {
	c.readMu.Lock()
	c.readDeadline = t
	c.readMu.Unlock()
	return nil
}

func (c *clientStreamConn) SetWriteDeadline(t time.Time) error {
	c.writeMu.Lock()
	c.writeDeadline = t
	c.writeMu.Unlock()
	return nil
}

func (c *clientStreamConn) LocalAddr() stdnet.Addr {
	return c.conn.LocalAddr()
}

func (c *clientStreamConn) RemoteAddr() stdnet.Addr {
	return c.conn.RemoteAddr()
}

func (c *clientStreamConn) closeRead(err error) {
	c.readCloseOnce.Do(func() {
		c.readMu.Lock()
		c.readErr = err
		c.readMu.Unlock()
		_ = c.readWriter.Close()
	})
}

func (c *clientStreamConn) closeTransport() error {
	var closeErr error
	c.transportOnce.Do(func() {
		c.writeMu.Lock()
		c.writeClosed = true
		c.writeMu.Unlock()
		closeErr = c.conn.Close()
	})
	return closeErr
}

func (c *clientStreamConn) writeFrame(cmd byte, streamID uint32, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.writeFrameLocked(cmd, streamID, payload)
}

func (c *clientStreamConn) writeFrameLocked(cmd byte, streamID uint32, payload []byte) error {
	if len(payload) > math.MaxUint16 {
		return fmt.Errorf("anytls frame payload too large: %d", len(payload))
	}
	if c.writeClosed && cmd != anyTLSCmdFIN {
		return io.ErrClosedPipe
	}

	frame := make([]byte, anyTLSFrameHeaderSize+len(payload))
	frame[0] = cmd
	binary.BigEndian.PutUint32(frame[1:5], streamID)
	binary.BigEndian.PutUint16(frame[5:7], uint16(len(payload)))
	copy(frame[7:], payload)

	return c.writeAll(frame)
}

func (c *clientStreamConn) writeAll(payload []byte) error {
	deadline := c.writeDeadline
	if deadline.IsZero() {
		deadline = time.Now().Add(anyTLSWriteTimeout)
	}
	if err := c.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	defer c.conn.SetWriteDeadline(time.Time{})

	for len(payload) > 0 {
		n, err := c.conn.Write(payload)
		if err != nil {
			return err
		}
		payload = payload[n:]
	}
	return nil
}
