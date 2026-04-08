package anytls

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"io"
	"math/big"
	stdnet "net"
	"strings"
	"testing"
	"time"

	M "github.com/sagernet/sing/common/metadata"
	"github.com/xtls/xray-core/common/buf"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/transport"
	"github.com/xtls/xray-core/transport/internet"
	"github.com/xtls/xray-core/transport/internet/stat"
	"github.com/xtls/xray-core/transport/pipe"
)

func TestOutboundProcessRelaysTCPOverAnyTLS(t *testing.T) {
	t.Parallel()

	password := "test-pass"
	target := xnet.TCPDestination(xnet.DomainAddress("example.com"), xnet.Port(443))

	server := newAnyTLSTestServer(t, password, target)
	defer server.Close()

	outbound, err := New(context.Background(), &Config{
		Address:  "127.0.0.1",
		Port:     uint32(server.Port()),
		Password: password,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	uploadReader, uploadWriter := pipe.New(pipe.WithoutSizeLimit())
	downloadReader, downloadWriter := pipe.New(pipe.WithoutSizeLimit())
	link := &transport.Link{
		Reader: uploadReader,
		Writer: downloadWriter,
	}

	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{
		Target: target,
	}})

	processErr := make(chan error, 1)
	go func() {
		processErr <- outbound.Process(ctx, link, server.Dialer())
	}()

	requestPayload := []byte("hello anytls")
	if err := uploadWriter.WriteMultiBuffer(buf.MergeBytes(nil, requestPayload)); err != nil {
		t.Fatalf("upload write error: %v", err)
	}
	_ = uploadWriter.Close()

	response, err := readPipePayload(downloadReader, 5*time.Second)
	if err != nil {
		t.Fatalf("download read error: %v", err)
	}
	if string(response) != strings.ToUpper(string(requestPayload)) {
		t.Fatalf("unexpected response payload: got %q want %q", string(response), strings.ToUpper(string(requestPayload)))
	}

	select {
	case err := <-processErr:
		if err != nil {
			t.Fatalf("Process() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Process() did not finish")
	}
}

func TestOutboundRejectsUDPTarget(t *testing.T) {
	t.Parallel()

	outbound, err := New(context.Background(), &Config{
		Address:  "127.0.0.1",
		Port:     443,
		Password: "test-pass",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	link := &transport.Link{}
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{
		Target: xnet.UDPDestination(xnet.DomainAddress("example.com"), xnet.Port(53)),
	}})

	err = outbound.Process(ctx, link, anyTLSTestDialer{})
	if err == nil {
		t.Fatal("expected UDP target to be rejected")
	}
	if !strings.Contains(err.Error(), "UDP") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type anyTLSTestServer struct {
	listener stdnet.Listener
	password string
	target   xnet.Destination
}

func newAnyTLSTestServer(t *testing.T, password string, target xnet.Destination) *anyTLSTestServer {
	t.Helper()

	cert := generateTestCertificate(t)
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{cert},
	})
	if err != nil {
		t.Fatalf("tls.Listen error: %v", err)
	}

	server := &anyTLSTestServer{
		listener: listener,
		password: password,
		target:   target,
	}

	go server.serve(t)

	return server
}

func (s *anyTLSTestServer) Close() error {
	return s.listener.Close()
}

func (s *anyTLSTestServer) Port() int {
	return s.listener.Addr().(*stdnet.TCPAddr).Port
}

func (s *anyTLSTestServer) Dialer() internet.Dialer {
	return anyTLSTestDialer{
		tlsConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
}

func (s *anyTLSTestServer) serve(t *testing.T) {
	t.Helper()

	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	if err := s.handleConn(conn); err != nil {
		t.Errorf("AnyTLS test server error: %v", err)
	}
}

func (s *anyTLSTestServer) handleConn(conn stdnet.Conn) error {
	auth := make([]byte, sha256.Size+2)
	if _, err := io.ReadFull(conn, auth); err != nil {
		return err
	}

	passwordHash := sha256.Sum256([]byte(s.password))
	if string(auth[:sha256.Size]) != string(passwordHash[:]) {
		return io.ErrUnexpectedEOF
	}

	paddingLen := binary.BigEndian.Uint16(auth[sha256.Size:])
	if paddingLen > 0 {
		padding := make([]byte, paddingLen)
		if _, err := io.ReadFull(conn, padding); err != nil {
			return err
		}
	}

	frame, err := readAnyTLSTestFrame(conn)
	if err != nil {
		return err
	}
	if frame.cmd != anyTLSCmdSettings {
		return io.ErrUnexpectedEOF
	}

	if err := writeAnyTLSTestFrame(conn, anyTLSTestFrame{
		cmd:  anyTLSCmdServerSettings,
		data: []byte("v=2"),
	}); err != nil {
		return err
	}

	frame, err = readAnyTLSTestFrame(conn)
	if err != nil {
		return err
	}
	if frame.cmd != anyTLSCmdSYN || frame.streamID != anyTLSStreamID {
		return io.ErrUnexpectedEOF
	}

	frame, err = readAnyTLSTestFrame(conn)
	if err != nil {
		return err
	}
	if frame.cmd != anyTLSCmdPSH || frame.streamID != anyTLSStreamID {
		return io.ErrUnexpectedEOF
	}

	destination, err := M.SocksaddrSerializer.ReadAddrPort(strings.NewReader(string(frame.data)))
	if err != nil {
		return err
	}
	if destination.String() != "example.com:443" {
		return io.ErrUnexpectedEOF
	}

	if err := writeAnyTLSTestFrame(conn, anyTLSTestFrame{
		cmd:      anyTLSCmdSYNACK,
		streamID: anyTLSStreamID,
	}); err != nil {
		return err
	}

	for {
		frame, err = readAnyTLSTestFrame(conn)
		if err != nil {
			return err
		}

		switch frame.cmd {
		case anyTLSCmdPSH:
			response := strings.ToUpper(string(frame.data))
			if err := writeAnyTLSTestFrame(conn, anyTLSTestFrame{
				cmd:      anyTLSCmdPSH,
				streamID: frame.streamID,
				data:     []byte(response),
			}); err != nil {
				return err
			}
		case anyTLSCmdFIN:
			return writeAnyTLSTestFrame(conn, anyTLSTestFrame{
				cmd:      anyTLSCmdFIN,
				streamID: frame.streamID,
			})
		default:
			return io.ErrUnexpectedEOF
		}
	}
}

type anyTLSTestDialer struct {
	tlsConfig *tls.Config
}

func (d anyTLSTestDialer) Dial(ctx context.Context, destination xnet.Destination) (stat.Connection, error) {
	address := stdnet.JoinHostPort(destination.Address.String(), destination.Port.String())
	return tls.DialWithDialer(&stdnet.Dialer{}, "tcp", address, d.tlsConfig)
}

func (anyTLSTestDialer) DestIpAddress() xnet.IP {
	return nil
}

func (anyTLSTestDialer) SetOutboundGateway(context.Context, *session.Outbound) {}

type anyTLSTestFrame struct {
	cmd      byte
	streamID uint32
	data     []byte
}

func readAnyTLSTestFrame(reader io.Reader) (anyTLSTestFrame, error) {
	var header [anyTLSFrameHeaderSize]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return anyTLSTestFrame{}, err
	}

	frame := anyTLSTestFrame{
		cmd:      header[0],
		streamID: binary.BigEndian.Uint32(header[1:5]),
	}
	length := int(binary.BigEndian.Uint16(header[5:7]))
	if length > 0 {
		frame.data = make([]byte, length)
		if _, err := io.ReadFull(reader, frame.data); err != nil {
			return anyTLSTestFrame{}, err
		}
	}

	return frame, nil
}

func writeAnyTLSTestFrame(writer io.Writer, frame anyTLSTestFrame) error {
	payload := make([]byte, anyTLSFrameHeaderSize+len(frame.data))
	payload[0] = frame.cmd
	binary.BigEndian.PutUint32(payload[1:5], frame.streamID)
	binary.BigEndian.PutUint16(payload[5:7], uint16(len(frame.data)))
	copy(payload[7:], frame.data)
	_, err := writer.Write(payload)
	return err
}

func readPipePayload(reader *pipe.Reader, timeout time.Duration) ([]byte, error) {
	mb, err := reader.ReadMultiBufferTimeout(timeout)
	if err != nil {
		return nil, err
	}
	defer buf.ReleaseMulti(mb)

	data := make([]byte, mb.Len())
	_, _ = buf.SplitBytes(mb, data)
	return data, nil
}

func generateTestCertificate(t *testing.T) tls.Certificate {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey error: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "127.0.0.1",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []stdnet.IP{stdnet.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("x509.CreateCertificate error: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("tls.X509KeyPair error: %v", err)
	}
	return cert
}
