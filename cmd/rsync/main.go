package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BaBa0525/rsync-go/internal"
	"github.com/quic-go/quic-go"
)

func main() {
	args := os.Args
	if len(args) != 3 {
		fmt.Println("Usage: rsync <src> <dst>")
		os.Exit(1)
	}

	srcFilePath := args[1]
	parts := strings.SplitN(args[2], "@", 2)
	host, dstFilePath := parts[0], parts[1]
	fmt.Println(host, dstFilePath)
	log.Println("clean dstFilePath:", filepath.Clean(dstFilePath))

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{internal.TLSProto},
	}

	conn, err := quic.DialAddr(context.Background(), host, tlsConf, nil)
	internal.Unwrap(err)

	stream, err := conn.OpenStreamSync(context.Background())
	internal.Unwrap(err)

	initialPacket := internal.InitialPacket{
		Path: dstFilePath,
		Header: internal.Header{
			Type: internal.SyncInfo,
		},
	}

	_, err = stream.Write(initialPacket.MarshalBinary())
	internal.Unwrap(err)

	buffer := make([]byte, 12)
	_, err = stream.Read(buffer)
	internal.Unwrap(err)

	header := internal.Header{}
	header.UnmarshalBinary(buffer)

	buffer = make([]byte, header.Length)
	nbytes, err := stream.Read(buffer)
	internal.Unwrap(err)

	dstFileInfo, err := internal.FileInfoPacketFromBytes(buffer[:nbytes])
	internal.Unwrap(err)

	chksumMap := make(map[string]string)
	for _, info := range dstFileInfo.Files {
		chksumMap[filepath.Join(srcFilePath, info.Path)] = info.CheckSum
		fmt.Println(info.Path, info.CheckSum)
	}

	files := []string{}
	filepath.Walk(srcFilePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(srcFilePath, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}
		files = append(files, relativePath)

		return nil
	})

	var wg sync.WaitGroup

	for _, file := range files {
		localPath := filepath.Join(srcFilePath, file)
		remotePath := filepath.Join(dstFilePath, file)

		dstChecksum, ok := chksumMap[localPath]
		if !ok {
			newStream, err := conn.OpenStreamSync(context.Background())
			internal.Unwrap(err)
			wg.Add(1)
			go func() {
				log.Println("in goroutine")
				defer wg.Done()
				sendFileContent(localPath, remotePath, newStream)
			}()
			log.Println("file not found in dst:", localPath)
			continue
		}
		delete(chksumMap, localPath)

		srcChecksum, err := internal.CheckSum(localPath)
		internal.Unwrap(err)

		if *srcChecksum != dstChecksum {
			log.Println("checksum not match:", localPath)
		}
	}

	for path := range chksumMap {
		log.Println("file not found in src:", path)
	}

	wg.Wait()
}

func sendFileContent(localPath string, remotePath string, stream quic.Stream) {
	log.Println("send file content:", localPath)
	defer stream.Close()

	file, err := os.Open(localPath)
	internal.Unwrap(err)
	defer file.Close()

	info, err := file.Stat()
	internal.Unwrap(err)

	header := internal.FileContentHeader{
		Header: internal.Header{
			Type: internal.FileContent,
		},
		FileContentLength: uint64(info.Size()),
		Path:              remotePath,
	}

	nbytes, err := stream.Write(header.MarshalBinary())
	internal.Unwrap(err)

	log.Println("send header:", nbytes)

	buffer := make([]byte, 1024)
	for {
		nbytes, err := file.Read(buffer)
		if err != nil {
			break
		}
		stream.Write(buffer[:nbytes])
	}
	_, err = stream.Read(buffer)
	internal.Unwrap(err)
}
