package sftpclient

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Config struct {
	Host                  string
	Port                  int
	User                  string
	Pass                  string
	RemoteDir             string
	InsecureIgnoreHostKey bool
}

func UploadFile(ctx context.Context, cfg Config, localPath string, remoteFileName string) error {
	if cfg.Host == "" || cfg.User == "" || cfg.Pass == "" {
		return fmt.Errorf("sftp: missing env SFTP_HOST / SFTP_USER / SFTP_PASS")
	}
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if cfg.RemoteDir == "" {
		cfg.RemoteDir = "/"
	}

	cb := ssh.InsecureIgnoreHostKey()
	if !cfg.InsecureIgnoreHostKey {
		// Más adelante: reemplazar por known_hosts.
		// Por ahora mantenemos seguro/simple para dev.
		cb = ssh.InsecureIgnoreHostKey()
	}

	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.Password(cfg.Pass)},
		HostKeyCallback: cb,
		Timeout:         20 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	// ctx para timeout/cancel
	type dialRes struct {
		client *ssh.Client
		err    error
	}
	ch := make(chan dialRes, 1)
	go func() {
		c, err := ssh.Dial("tcp", addr, sshCfg)
		ch <- dialRes{client: c, err: err}
	}()

	var sshClient *ssh.Client
	select {
	case <-ctx.Done():
		return fmt.Errorf("sftp: dial canceled: %w", ctx.Err())
	case r := <-ch:
		if r.err != nil {
			return fmt.Errorf("sftp: dial error: %w", r.err)
		}
		sshClient = r.client
	}
	defer sshClient.Close()

	sftpCli, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("sftp: new client: %w", err)
	}
	defer sftpCli.Close()

	// Asegura dir destino
	if err := sftpCli.MkdirAll(cfg.RemoteDir); err != nil {
		return fmt.Errorf("sftp: mkdir %s: %w", cfg.RemoteDir, err)
	}

	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("sftp: open local file: %w", err)
	}
	defer src.Close()

	remotePath := path.Join(cfg.RemoteDir, remoteFileName)
	dst, err := sftpCli.Create(remotePath)
	if err != nil {
		return fmt.Errorf("sftp: create remote file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("sftp: upload copy: %w", err)
	}

	return nil
}
