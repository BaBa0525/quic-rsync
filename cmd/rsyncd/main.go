package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"path/filepath"

	"github.com/BaBa0525/rsync-go/internal"
	"github.com/quic-go/quic-go"
)

func main() {
	log.Println("Starting server on", internal.Addr)
	listener, err := quic.ListenAddr(internal.Addr, generateTLSConfig(), nil)
	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept(context.Background())
		internal.Unwrap(err)
		log.Println("Accept connection!")

		go handleConnection(conn)
	}

}

func handleConnection(conn quic.Connection) error {
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			return err
		}
		go handleStream(stream)
	}
}

func handleStream(stream quic.Stream) error {
	buffer := make([]byte, 12)
	nbytes, err := stream.Read(buffer)
	if err != nil {
		return err
	}

	header := internal.Header{}
	header.UnmarshalBinary(buffer[:nbytes])

	log.Println("headerType: ", header.Type)

	switch header.Type {
	case internal.SyncInfo:
		return handleSyncRequest(stream, header.Length)
	case internal.FileContent:
		return handleFileContent(stream, header.Length)
	case internal.DeleteFile:
		return handleDeleteFile(stream, header.Length)
	default:
		return nil
	}
}

func handleDeleteFile(stream quic.Stream, contentLength uint64) error {

	buffer := make([]byte, contentLength)
	nbytes, err := stream.Read(buffer)
	if err != nil {
		return err
	}

	deleteFiles := []string{}
	err = json.Unmarshal(buffer[:nbytes], &deleteFiles)
	if err != nil {
		return err
	}

	for _, path := range deleteFiles {
		log.Println("delete file: ", path)
		err = os.Remove(path)
		if err != nil {
			return err
		}
	}

	stream.Write([]byte("ok"))
	return nil
}

func handleFileContent(stream quic.Stream, headerLength uint64) error {
	receivedBytes := uint64(0)

	buffer := make([]byte, headerLength)
	nbytes, err := stream.Read(buffer)
	if err != nil {
		return err
	}

	fileContentLength := binary.BigEndian.Uint64(buffer[:8])
	path := string(buffer[8:nbytes])

	log.Println("path: ", path)

	os.MkdirAll(filepath.Dir(path), 0755)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for receivedBytes < fileContentLength {
		nbytes, err := stream.Read(buffer)
		if err != nil {
			return err
		}

		receivedBytes += uint64(nbytes)
		f.Write(buffer[:nbytes])

	}

	stream.Write([]byte("ok"))

	return nil
}

func handleSyncRequest(stream quic.Stream, headerLength uint64) error {
	buffer := make([]byte, headerLength)
	nbytes, err := stream.Read(buffer)
	if err != nil {
		return err
	}

	rootPath := string(buffer[:nbytes])

	files := []string{}
	filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}
		files = append(files, relativePath)

		return nil
	})

	fileInfo := []internal.FileInfo{}
	for _, file := range files {
		checksum, err := internal.CheckSum(filepath.Join(rootPath, file))
		if err != nil {
			return err
		}
		fileInfo = append(fileInfo, internal.FileInfo{
			Path:     file,
			CheckSum: *checksum,
		})
	}

	fileInfoPacket := internal.FileInfoPacket{Files: fileInfo}
	fileInfoPacketBytes, err := fileInfoPacket.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = stream.Write(fileInfoPacketBytes)
	if err != nil {
		return err
	}

	return nil
}

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{internal.TLSProto},
	}
}
