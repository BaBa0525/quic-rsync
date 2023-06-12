package internal

import (
	"encoding/binary"
	"encoding/json"
)

type PacketType uint32

const (
	SyncInfo PacketType = iota
	FileContent
	Delete
)

type Header struct {
	Length uint64
	Type   PacketType
}

func (h *Header) MarshalBinary() []byte {
	bytes := make([]byte, 12)
	binary.BigEndian.PutUint64(bytes[:8], h.Length)
	binary.BigEndian.PutUint32(bytes[8:], uint32(h.Type))

	return bytes
}

func (h *Header) UnmarshalBinary(data []byte) {
	h.Length = binary.BigEndian.Uint64(data[:8])
	h.Type = PacketType(binary.BigEndian.Uint32(data[8:]))
}

type InitialPacket struct {
	Header Header
	Path   string
}

func (i *InitialPacket) MarshalBinary() []byte {
	i.Header.Length = uint64(len(i.Path))
	bytes := i.Header.MarshalBinary()
	bytes = append(bytes, []byte(i.Path)...)
	return bytes
}

type FileInfo struct {
	Path     string `json:"path"`
	CheckSum string `json:"checksum"`
}

type FileInfoPacket struct {
	Header Header
	Files  []FileInfo
}

func (f *FileInfoPacket) MarshalBinary() ([]byte, error) {
	bytes, err := f.MarshalJSON()
	if err != nil {
		return nil, err
	}
	f.Header.Length = uint64(len(bytes))

	return append(f.Header.MarshalBinary(), bytes...), nil
}

func (f *FileInfoPacket) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.Files)
}

func (f *FileInfoPacket) UnmarshalJSON(data []byte) error {
	var rawMessages []json.RawMessage
	err := json.Unmarshal(data, &rawMessages)
	if err != nil {
		return err
	}

	for _, rawMessage := range rawMessages {
		var fileInfo FileInfo
		err = json.Unmarshal(rawMessage, &fileInfo)
		if err != nil {
			return err
		}
		f.Files = append(f.Files, fileInfo)
	}

	return nil
}

func FileInfoPacketFromBytes(data []byte) (*FileInfoPacket, error) {
	packet := FileInfoPacket{}
	err := packet.UnmarshalJSON(data)
	if err != nil {
		return nil, err
	}
	return &packet, nil
}

type FileContentHeader struct {
	Header            Header
	FileContentLength uint64
	Path              string
}

func (f *FileContentHeader) MarshalBinary() []byte {
	f.Header.Length = uint64(len(f.Path)) + 8
	bytes := f.Header.MarshalBinary()
	bytes = binary.BigEndian.AppendUint64(bytes, f.FileContentLength)
	bytes = append(bytes, []byte(f.Path)...)
	return bytes
}
