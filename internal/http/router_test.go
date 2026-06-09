package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	nethttp "net/http"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/t3chn0m4g3/hellpot/internal/config"
)

func configureHTTPTest(t *testing.T) {
	t.Helper()

	if _, err := config.StartLogger(false, io.Discard); err != nil {
		t.Fatalf("StartLogger() error = %v", err)
	}

	config.HTTPBind = "127.0.0.1"
	config.HTTPPort = "0"
	config.HeaderName = "X-Real-IP"
	config.FakeServerName = "nginx"
	config.MakeRobots = true
	config.CatchAll = false
	config.Paths = nil
	config.UseragentBlacklistMatchers = nil
	config.RestrictConcurrency = true
	config.MaxWorkers = 16
	config.UseUnixSocket = false
	config.UnixSocketPath = ""
	config.UnixSocketPermissions = 0o600
	config.Trace = false
	log = config.GetLogger()
}

func TestRobotsTXTUsesSlashPathsAndTextContentType(t *testing.T) {
	configureHTTPTest(t)
	config.Paths = []string{"wp-login.php", "/admin"}

	var ctx fasthttp.RequestCtx
	ctx.Request.SetRequestURI("/robots.txt")
	ctx.Request.Header.SetUserAgent("test")
	robotsTXT(&ctx)

	body := string(ctx.Response.Body())
	if !strings.Contains(body, "Disallow: /wp-login.php\r\n") {
		t.Fatalf("robots.txt body missing normalized path:\n%s", body)
	}
	if !strings.Contains(body, "Disallow: /admin\r\n") {
		t.Fatalf("robots.txt body missing slash path:\n%s", body)
	}
	if got := string(ctx.Response.Header.ContentType()); got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/plain; charset=utf-8", got)
	}
}

func TestHellPotRejectsBlacklistedUserAgent(t *testing.T) {
	configureHTTPTest(t)
	config.UseragentBlacklistMatchers = []string{"curl"}

	var ctx fasthttp.RequestCtx
	ctx.Request.SetRequestURI("/wp-login.php")
	ctx.Request.Header.SetUserAgent("curl/8.0")
	hellPot(&ctx)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want %d", got, fasthttp.StatusNotFound)
	}
}

func TestNewServerBuildsTCPOptions(t *testing.T) {
	configureHTTPTest(t)
	config.Paths = []string{"wp-login.php"}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if srv.useUnixSocket {
		t.Fatal("useUnixSocket = true, want false")
	}
	if srv.address != "127.0.0.1:0" {
		t.Fatalf("address = %q, want 127.0.0.1:0", srv.address)
	}
	if srv.srv.Concurrency != 16 {
		t.Fatalf("Concurrency = %d, want 16", srv.srv.Concurrency)
	}
	if !srv.srv.GetOnly {
		t.Fatal("GetOnly = false, want true")
	}
}

func TestTCPServerSmoke(t *testing.T) {
	configureHTTPTest(t)
	config.CatchAll = true
	config.MakeRobots = false
	config.UseragentBlacklistMatchers = []string{"curl"}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if isSocketPermissionError(err) {
			t.Skipf("socket listen is not permitted in this environment: %v", err)
		}
		t.Fatalf("Listen() error = %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.srv.Serve(ln)
	}()

	resp, err := requestWithUserAgent("http://"+ln.Addr().String()+"/anything", "curl/8.0", nil)
	if err != nil {
		t.Fatalf("HTTP request error = %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != nethttp.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, nethttp.StatusNotFound)
	}

	if err = srv.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err = <-errCh; err != nil && !isClosedNetworkError(err) {
		t.Fatalf("Serve() after shutdown error = %v", err)
	}
}

func TestHellPotStreamsUntilClientCloses(t *testing.T) {
	configureHTTPTest(t)
	config.CatchAll = true
	config.MakeRobots = false

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if isSocketPermissionError(err) {
			t.Skipf("socket listen is not permitted in this environment: %v", err)
		}
		t.Fatalf("Listen() error = %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.srv.Serve(ln)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, "http://"+ln.Addr().String()+"/anything", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla")

	client := &nethttp.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request error = %v", err)
	}
	if resp.StatusCode != nethttp.StatusOK {
		_ = resp.Body.Close()
		t.Fatalf("status = %d, want %d", resp.StatusCode, nethttp.StatusOK)
	}

	buf := make([]byte, 128*1024)
	if _, err = io.ReadFull(resp.Body, buf); err != nil {
		_ = resp.Body.Close()
		t.Fatalf("stream ended before 128 KiB: %v", err)
	}
	if !bytes.HasPrefix(buf, []byte("<html>\n<body>\n")) {
		_ = resp.Body.Close()
		t.Fatalf("stream prefix = %q, want HTML body prefix", buf[:14])
	}
	_ = resp.Body.Close()

	if err = srv.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err = <-errCh; err != nil && !isClosedNetworkError(err) {
		t.Fatalf("Serve() after shutdown error = %v", err)
	}
}

func TestUnixSocketServerSmoke(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sockets are not supported on Windows")
	}

	configureHTTPTest(t)
	config.CatchAll = true
	config.MakeRobots = false
	config.UseragentBlacklistMatchers = []string{"curl"}
	config.UseUnixSocket = true
	config.UnixSocketPath = filepath.Join(t.TempDir(), "hellpot.sock")
	config.UnixSocketPermissions = 0o600

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve()
	}()
	waitForSocket(t, config.UnixSocketPath, errCh)

	transport := &nethttp.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", config.UnixSocketPath)
		},
	}
	defer transport.CloseIdleConnections()

	resp, err := requestWithUserAgent("http://hellpot/anything", "curl/8.0", transport)
	if err != nil {
		t.Fatalf("HTTP request over unix socket error = %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != nethttp.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, nethttp.StatusNotFound)
	}

	if err = srv.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err = <-errCh; err != nil && !isClosedNetworkError(err) {
		t.Fatalf("Serve() after shutdown error = %v", err)
	}
}

func requestWithUserAgent(url, userAgent string, transport nethttp.RoundTripper) (*nethttp.Response, error) {
	client := &nethttp.Client{Transport: transport}
	req, err := nethttp.NewRequest(nethttp.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return client.Do(req)
}

func waitForSocket(t *testing.T, path string, errCh <-chan error) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	var lastDialErr error
	for time.Now().Before(deadline) {
		if conn, err := net.Dial("unix", path); err == nil {
			_ = conn.Close()
			return
		} else {
			lastDialErr = err
		}

		select {
		case err := <-errCh:
			if isSocketPermissionError(err) {
				t.Skipf("unix socket listen is not permitted in this environment: %v", err)
			}
			t.Fatalf("server stopped before socket was ready: %v", err)
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
	if isSocketPermissionError(lastDialErr) {
		t.Skipf("unix socket dial is not permitted in this environment: %v", lastDialErr)
	}
	t.Fatalf("socket %q was not ready", path)
}

func isSocketPermissionError(err error) bool {
	return errors.Is(err, syscall.EPERM) ||
		errors.Is(err, syscall.EACCES) ||
		strings.Contains(err.Error(), "operation not permitted")
}

func isClosedNetworkError(err error) bool {
	return errors.Is(err, net.ErrClosed) ||
		strings.Contains(err.Error(), "use of closed network connection")
}
