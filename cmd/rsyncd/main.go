package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
	"math/big"

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
		if err != nil {
			panic(err)
		}
		log.Println("Accept connection!")

		go handleConnection(conn)
	}

}

func handleConnection(conn quic.Connection) error {
	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		return err
	}

	for {
		buffer := make([]byte, 1024)
		_, err = stream.Read(buffer)
		if err != nil {
			return err
		}

		log.Print(string(buffer))
	}
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