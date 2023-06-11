package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/BaBa0525/rsync-go/internal"
	"github.com/quic-go/quic-go"
)

func main() {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{internal.TLSProto},
	}

	conn, err := quic.DialAddr(context.Background(), "localhost:8773", tlsConf, nil)
	internal.Unwrap(err)

	stream, err := conn.OpenStreamSync(context.Background())
	internal.Unwrap(err)

	for {

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		text, err := reader.ReadString('\n')
		internal.Unwrap(err)

		_, err = stream.Write([]byte(text))
		internal.Unwrap(err)
	}

}
