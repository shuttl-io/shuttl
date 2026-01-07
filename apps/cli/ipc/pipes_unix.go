//go:build !windows

package ipc

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/containerd/fifo"
	"github.com/shuttl-ai/cli/log"
)

// createNamedPipes creates the named pipe paths for IPC (Unix version)
func (c *Client) createNamedPipes() error {
	// Create a temporary directory for the pipes
	var err error
	c.pipeDir, err = os.MkdirTemp("", "shuttl-ipc-*")
	if err != nil {
		return fmt.Errorf("failed to create pipe directory: %w", err)
	}

	c.requestPipePath = filepath.Join(c.pipeDir, "request")
	c.responsePipePath = filepath.Join(c.pipeDir, "response")

	log.DebugWithPrefix("IPC", "Created named pipe paths: request=%s, response=%s", c.requestPipePath, c.responsePipePath)
	return nil
}

// openPipes opens the named pipes for communication using containerd/fifo (Unix version)
func (c *Client) openPipes(ctx context.Context) error {
	var err error

	// Open response pipe for reading with O_CREAT to create if needed
	c.responsePipe, err = fifo.OpenFifo(ctx, c.responsePipePath, syscall.O_RDONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return fmt.Errorf("failed to open response pipe: %w", err)
	}

	// Open request pipe for writing with O_CREAT to create if needed
	c.requestPipe, err = fifo.OpenFifo(ctx, c.requestPipePath, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0600)
	if err != nil {
		c.responsePipe.Close()
		return fmt.Errorf("failed to open request pipe: %w", err)
	}

	return nil
}

// getResponsePipeReader returns a reader for the response pipe
func (c *Client) getResponsePipeReader() io.ReadCloser {
	return c.responsePipe
}

// cleanupNamedPipes removes the named pipes and their directory (Unix version)
func (c *Client) cleanupNamedPipes() {
	if c.requestPipe != nil {
		c.requestPipe.Close()
		c.requestPipe = nil
	}
	if c.responsePipe != nil {
		c.responsePipe.Close()
		c.responsePipe = nil
	}
	if c.pipeDir != "" {
		os.RemoveAll(c.pipeDir)
		log.DebugWithPrefix("IPC", "Cleaned up named pipes: %s", c.pipeDir)
		c.pipeDir = ""
	}
}

