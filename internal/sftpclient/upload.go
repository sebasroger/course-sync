package sftpclient

import (
	"bufio"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strings"
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

	// Host key pinning: "ssh-rsa AAAA..." (SIN hostname). Opcional si InsecureIgnoreHostKey=true.
	HostKey string

	KeyPath       string
	KeyPassphrase string
}

func UploadFile(ctx context.Context, cfg Config, localPath string, remoteFileName string) error {
	if cfg.Host == "" || cfg.User == "" {
		return fmt.Errorf("sftp: missing SFTP_HOST / SFTP_USER")
	}
	if cfg.Pass == "" && cfg.KeyPath == "" {
		return fmt.Errorf("sftp: no auth method configured (set SFTP_KEY_PATH or SFTP_PASS)")
	}
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if cfg.RemoteDir == "" {
		cfg.RemoteDir = "/"
	}

	// Host key callback
	var hostKeyCb ssh.HostKeyCallback
	if cfg.InsecureIgnoreHostKey {
		hostKeyCb = ssh.InsecureIgnoreHostKey()
	} else {
		if strings.TrimSpace(cfg.HostKey) == "" {
			return fmt.Errorf("sftp: host key check enabled but SFTP_HOST_KEY not set (set SFTP_HOST_KEY or set SFTP_INSECURE_IGNORE_HOSTKEY=true)")
		}
		expectedType, expectedB64, err := splitKey(cfg.HostKey)
		if err != nil {
			return fmt.Errorf("sftp: invalid SFTP_HOST_KEY: %w", err)
		}
		expectedRaw, err := base64.StdEncoding.DecodeString(expectedB64)
		if err != nil {
			return fmt.Errorf("sftp: invalid SFTP_HOST_KEY base64: %w", err)
		}
		hostKeyCb = func(hostname string, remoteAddr net.Addr, key ssh.PublicKey) error {
			if key.Type() != expectedType {
				return fmt.Errorf("sftp: host key mismatch for %s: type %s != %s", remoteAddr.String(), key.Type(), expectedType)
			}
			if subtle.ConstantTimeCompare(key.Marshal(), expectedRaw) != 1 {
				return fmt.Errorf("sftp: host key mismatch for %s", remoteAddr.String())
			}
			return nil
		}
	}

	// Auth
	var auth []ssh.AuthMethod

	if cfg.KeyPath != "" {
		keyBytes, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return fmt.Errorf("sftp: read key: %w", err)
		}

		var signer ssh.Signer
		if cfg.KeyPassphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(cfg.KeyPassphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(keyBytes)
		}
		if err != nil {
			return fmt.Errorf("sftp: parse key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}

	if cfg.Pass != "" {
		auth = append(auth, ssh.Password(cfg.Pass))
	}

	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCb,
		Timeout:         20 * time.Second,
		// Habilitar compresión para mejorar velocidad
		Config: ssh.Config{
			Ciphers: []string{
				"aes128-gcm@openssh.com",
				"aes256-gcm@openssh.com",
				"aes128-ctr",
				"aes192-ctr",
				"aes256-ctr",
			},
		},
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

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

	// Dir: intentar crear; si no se puede, validar que exista.
	if cfg.RemoteDir != "/" {
		if err := sftpCli.MkdirAll(cfg.RemoteDir); err != nil {
			// si no deja crear, al menos que exista
			if _, statErr := sftpCli.Stat(cfg.RemoteDir); statErr != nil {
				return fmt.Errorf("sftp: remote dir not accessible %s: mkdirErr=%v statErr=%v", cfg.RemoteDir, err, statErr)
			}
		}
	}

	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("sftp: open local file: %w", err)
	}
	defer src.Close()

	remotePath := path.Join(cfg.RemoteDir, remoteFileName)

	// IMPORTANTE: abrir WRITE-ONLY (evita SSH_FX_OP_UNSUPPORTED por READ flag)
	dst, err := sftpCli.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("sftp: create remote file %s: %w", remotePath, err)
	}
	defer dst.Close()

	// Usar buffer grande para mejorar rendimiento
	bufSize := 1024 * 1024 // 1MB buffer
	buf := make([]byte, bufSize)

	// Crear un escritor bufferizado para mejorar rendimiento
	bufWriter := bufio.NewWriterSize(dst, bufSize)

	// Obtener tamaño del archivo para reportar progreso
	fileInfo, err := src.Stat()
	if err != nil {
		return fmt.Errorf("sftp: get file size: %w", err)
	}
	totalSize := fileInfo.Size()

	// Iniciar tiempo para calcular velocidad
	startTime := time.Now()
	transferred := int64(0)
	lastReport := time.Now()

	// Copiar con buffer grande y reportar progreso
	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("sftp: read error: %w", err)
		}
		if n == 0 {
			break
		}

		if _, err := bufWriter.Write(buf[:n]); err != nil {
			return fmt.Errorf("sftp: write error: %w", err)
		}

		transferred += int64(n)

		// Reportar progreso cada 3 segundos
		if time.Since(lastReport) > 3*time.Second {
			elapsed := time.Since(startTime).Seconds()
			speed := float64(transferred) / elapsed / 1024 / 1024 // MB/s
			percent := float64(transferred) * 100 / float64(totalSize)
			log.Printf("SFTP: Transferido %.2f%% (%.2f MB/s)", percent, speed)
			lastReport = time.Now()
		}
	}

	// Asegurar que todos los datos se escriban
	if err := bufWriter.Flush(); err != nil {
		return fmt.Errorf("sftp: flush error: %w", err)
	}

	return nil
}

func splitKey(s string) (keyType string, b64 string, err error) {
	parts := strings.Fields(strings.TrimSpace(s))
	if len(parts) < 2 {
		return "", "", fmt.Errorf("expected format: '<type> <base64>'")
	}
	return parts[0], parts[1], nil
}
